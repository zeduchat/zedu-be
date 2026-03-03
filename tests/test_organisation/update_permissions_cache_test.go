package test_organisation

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/pkg/controller/auth"
	rd "github.com/hngprojects/telex_be/pkg/repository/storage/redis"
	"github.com/hngprojects/telex_be/tests"
	"github.com/hngprojects/telex_be/utility"
)

func TestUpdateOrgPermissions_CacheInvalidation(t *testing.T) {
	router, orgController := SetupOrgTestRouter()
	t.Cleanup(func() {
		tests.Cleanup(orgController.Db)
	})

	db := orgController.Db.Postgresql
	rdb := orgController.Db.Redis
	currUUID := utility.GenerateUUID()
	password, _ := utility.HashPassword("password")

	adminUser := models.User{
		ID:       utility.GenerateUUID(),
		Name:     "Cache Test Admin",
		Email:    fmt.Sprintf("cacheadmin%v@qa.team", currUUID),
		Password: password,
		Role:     int(models.RoleIdentity.SuperAdmin),
	}

	orgID := utility.GenerateUUID()
	org := models.Organisation{
		ID:      orgID,
		Name:    fmt.Sprintf("CacheOrg%v", currUUID),
		Email:   fmt.Sprintf("cacheorg%v@qa.team", currUUID),
		OwnerID: adminUser.ID,
	}

	db.Create(&adminUser)
	db.Create(&org)

	roleID := utility.GenerateUUID()
	role := models.OrgRole{
		ID:             roleID,
		Name:           fmt.Sprintf("CacheRole-%v", utility.RandomString(5)),
		Description:    "Role for cache test",
		OrganisationID: &orgID,
		IsDefault:      false,
	}

	perm := models.Permission{
		ID:        utility.GenerateUUID(),
		RoleID:    role.ID,
		IsDefault: false,
		PermissionList: models.PermissionList{
			CanCreateChannel:    true,
			CanViewChannels:     true,
			CanCommentOnThreads: true,
			CanInviteMembers:    false,
		},
	}

	db.Create(&role)
	db.Create(&perm)

	authController := auth.Controller{
		Db:        orgController.Db,
		Validator: orgController.Validator,
		Logger:    orgController.Logger,
		ExtReq:    orgController.ExtReq,
	}

	loginData := models.LoginRequestModel{
		Email:    adminUser.Email,
		Password: "password",
	}
	token := tests.GetLoginToken(t, gin.Default(), authController, loginData)

	t.Run("Cache is invalidated after permission update", func(t *testing.T) {
		// Pre-populate the Redis cache for this role
		cacheKey := "role_permissions_" + roleID
		rd.RedisSet(rdb, cacheKey, perm.PermissionList, 24*time.Hour)

		// Verify the cache entry exists
		cached, err := rd.RedisGet(rdb, cacheKey)
		if err != nil || len(cached) == 0 {
			t.Fatalf("Failed to pre-populate cache: err=%v, len=%d", err, len(cached))
		}

		// Update permissions via API
		updatedPermissions := models.PermissionList{
			CanCreateChannel:    true,
			CanViewChannels:     true,
			CanCommentOnThreads: true,
			CanInviteMembers:    true, // changed from false to true
		}

		permJSON, _ := json.Marshal(updatedPermissions)

		req, _ := http.NewRequest(
			http.MethodPut,
			fmt.Sprintf("/api/v1/organisations/%s/roles/%s/permissions", orgID, roleID),
			bytes.NewBuffer(permJSON),
		)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)

		tests.AssertStatusCode(t, resp.Code, http.StatusOK)

		// Verify the cache entry has been invalidated
		cached, err = rd.RedisGet(rdb, cacheKey)
		if err != redis.Nil && len(cached) > 0 {
			t.Errorf("Cache was NOT invalidated after permission update. Key %q still has data (len=%d)", cacheKey, len(cached))
		}
	})
}
