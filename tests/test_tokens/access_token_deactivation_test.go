package test_tokens

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt"
	"github.com/hngprojects/telex_be/internal/config"
	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/pkg/repository/storage"
	tst "github.com/hngprojects/telex_be/tests"
	"github.com/hngprojects/telex_be/utility"
)

func TestRevokeUserTokensByOrg_JWTClaimsCheck(t *testing.T) {
	tst.Setup()
	db := storage.Connection()
	if db == nil || db.Postgresql == nil {
		t.Skip("PostgreSQL database connection not available")
		return
	}

	userID := utility.GenerateUUID()
	org1ID := utility.GenerateUUID()
	org2ID := utility.GenerateUUID()
	cfg := config.GetConfig()

	createTokenWithOrg := func(orgID string) string {
		claims := jwt.MapClaims{
			"user_id":     userID,
			"access_uuid": utility.GenerateUUID(),
			"org_id":      orgID,
			"exp":         time.Now().Add(1 * time.Hour).Unix(),
			"authorised":  true,
		}
		token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
		tokenString, err := token.SignedString([]byte(cfg.Server.Secret))
		if err != nil {
			t.Fatalf("failed to sign token: %v", err)
		}
		return tokenString
	}

	token1Str := createTokenWithOrg(org1ID)
	token2Str := createTokenWithOrg(org2ID)

	accToken1 := models.AccessToken{
		ID:               utility.GenerateUUID(),
		OwnerID:          userID,
		LoginAccessToken: token1Str,
		IsLive:           true,
	}
	accToken2 := models.AccessToken{
		ID:               utility.GenerateUUID(),
		OwnerID:          userID,
		LoginAccessToken: token2Str,
		IsLive:           true,
	}

	if err := db.Postgresql.Create(&accToken1).Error; err != nil {
		t.Fatalf("failed to create accToken1: %v", err)
	}
	if err := db.Postgresql.Create(&accToken2).Error; err != nil {
		t.Fatalf("failed to create accToken2: %v", err)
	}

	defer func() {
		_ = db.Postgresql.Where("id IN ?", []string{accToken1.ID, accToken2.ID}).Delete(&models.AccessToken{}).Error
	}()

	var tokenModel models.AccessToken
	if err := tokenModel.RevokeUserTokensByOrg(db.Postgresql, userID, org1ID); err != nil {
		t.Fatalf("RevokeUserTokensByOrg failed: %v", err)
	}

	var fetched1, fetched2 models.AccessToken
	if err := db.Postgresql.Where("id = ?", accToken1.ID).First(&fetched1).Error; err != nil {
		t.Fatalf("failed to fetch fetched1: %v", err)
	}
	if err := db.Postgresql.Where("id = ?", accToken2.ID).First(&fetched2).Error; err != nil {
		t.Fatalf("failed to fetch fetched2: %v", err)
	}

	if fetched1.IsLive {
		t.Errorf("expected token for org1 (%s) to be revoked (is_live=false), got is_live=true", org1ID)
	}
	if !fetched2.IsLive {
		t.Errorf("expected token for org2 (%s) to remain active (is_live=true), got is_live=false", org2ID)
	}
}
