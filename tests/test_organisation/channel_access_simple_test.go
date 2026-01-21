package test_organisation

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/hngprojects/telex_be/external/request"
	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/pkg/controller/auth"
	"github.com/hngprojects/telex_be/pkg/controller/channel"
	"github.com/hngprojects/telex_be/pkg/controller/organisation"
	"github.com/hngprojects/telex_be/pkg/middleware"
	"github.com/hngprojects/telex_be/pkg/repository/storage"
	tst "github.com/hngprojects/telex_be/tests"
	"github.com/hngprojects/telex_be/utility"
)

func TestChannelAccessInOrganisation(t *testing.T) {
	logger := tst.Setup()
	gin.SetMode(gin.TestMode)

	validatorRef := validator.New()
	db := storage.Connection()

	// Setup Controllers
	authController := auth.Controller{Db: db, Validator: validatorRef, Logger: logger, ExtReq: request.ExternalRequest{Logger: logger, Test: true}}
	orgController := organisation.Controller{Db: db, Validator: validatorRef, Logger: logger}
	channelController := channel.Controller{Db: db, Validator: validatorRef, Logger: logger}

	r := gin.Default()

	currUUID := utility.GenerateUUID()
	userAData := models.CreateUserRequestModel{
		Email:       fmt.Sprintf("usera%v@qa.team", currUUID),
		PhoneNumber: fmt.Sprintf("+234%v", utility.GetRandomNumbersInRange(7000000000, 9099999999)),
		FirstName:   "User",
		LastName:    "A",
		Password:    "password",
		UserName:    fmt.Sprintf("usera%v", currUUID),
	}
	tst.SignupUser(t, r, authController, userAData, false)
	tokenA := tst.GetLoginToken(t, r, authController, models.LoginRequestModel{Email: userAData.Email, Password: userAData.Password})
	createOrgData := models.CreateOrgRequestModel{
		Name:        fmt.Sprintf("TestOrg%s", currUUID),
		Description: "Test Org Description",
		Email:       userAData.Email,
		Type:        "tech",
		Location:    "Remote",
		Country:     "Earth",
	}
	orgId, _, _ := tst.CreateOrganisation(t, r, db, orgController, createOrgData, tokenA)

	pubId, publicChannelName := tst.CreateChannels(t, r, channelController, db, models.CreateChannelsRequest{
		Name:           "PublicChannel",
		Username:       userAData.UserName,
		OrganisationID: orgId,
		Description:    "Public Channel",
		IsPrivate:      false,
	}, tokenA)

	privateChannelId, _ := tst.CreateChannels(t, gin.Default(), channelController, db, models.CreateChannelsRequest{
		Name:           "PrivateChannel",
		Username:       userAData.UserName,
		OrganisationID: orgId,
		Description:    "Private Channel",
		IsPrivate:      true,
	}, tokenA)

	userBData := models.CreateUserRequestModel{
		Email:       fmt.Sprintf("userb%v@qa.team", currUUID),
		PhoneNumber: fmt.Sprintf("+234%v", utility.GetRandomNumbersInRange(7000000000, 9099999999)),
		FirstName:   "User",
		LastName:    "B",
		Password:    "password",
		UserName:    fmt.Sprintf("userb%v", currUUID),
	}
	tst.SignupUser(t, gin.Default(), authController, userBData, false)
	tokenB := tst.GetLoginToken(t, gin.Default(), authController, models.LoginRequestModel{Email: userBData.Email, Password: userBData.Password})
	userBId := tst.GetUserIDFromToken(t, tokenB, db)

	userCData := models.CreateUserRequestModel{
		Email:       fmt.Sprintf("userc%v@qa.team", currUUID),
		PhoneNumber: fmt.Sprintf("+234%v", utility.GetRandomNumbersInRange(7000000000, 9099999999)),
		FirstName:   "User",
		LastName:    "C",
		Password:    "password",
		UserName:    fmt.Sprintf("userC%v", currUUID),
	}
	tst.SignupUser(t, gin.Default(), authController, userCData, false)
	tokenC := tst.GetLoginToken(t, gin.Default(), authController, models.LoginRequestModel{Email: userCData.Email, Password: userCData.Password})

	userChannel := models.UserChannels{
		ChannelsID: pubId,
		UserID:     userBId,
		Username:   userBData.UserName,
	}
	if err := db.Postgresql.Create(&userChannel).Error; err != nil {
		t.Fatalf("Failed to add user to channel: %v", err)
	}

	orgUrl := r.Group(fmt.Sprintf("%v", "/api/v1"), middleware.Authorize(db.Postgresql))
	{
		orgUrl.GET("/organisations/:org_id/channels", orgController.GetAllChannelssInOrganisation)
	}

	reqGetChannelsB, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/organisations/%s/channels", orgId), nil)
	reqGetChannelsB.Header.Set("Authorization", "Bearer "+tokenB)
	rrGetChannelsB := httptest.NewRecorder()
	r.ServeHTTP(rrGetChannelsB, reqGetChannelsB)
	tst.AssertStatusCode(t, rrGetChannelsB.Code, http.StatusOK)

	type ChannelItem struct {
		ID     string `json:"channels_id"`
		Name   string `json:"name"`
		Access bool   `json:"access"`
	}

	var parsedRespB map[string]interface{}
	json.Unmarshal(rrGetChannelsB.Body.Bytes(), &parsedRespB)
	dataBBytes, _ := json.Marshal(parsedRespB["data"])
	var channelsB []ChannelItem
	json.Unmarshal(dataBBytes, &channelsB)

	foundPublicB := false
	foundPrivateB := false

	for _, ch := range channelsB {
		if ch.Name == publicChannelName {
			foundPublicB = true
		}
		if ch.ID == privateChannelId {
			foundPrivateB = true
		}
	}

	if !foundPublicB {
		t.Errorf("User B should see public channel %s", publicChannelName)
	}
	if foundPrivateB {
		t.Errorf("User B should NOT see private channel %s", privateChannelId)
	}

	reqGetChannelsA, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/organisations/%s/channels", orgId), nil)
	reqGetChannelsA.Header.Set("Authorization", "Bearer "+tokenA)
	rrGetChannelsA := httptest.NewRecorder()
	r.ServeHTTP(rrGetChannelsA, reqGetChannelsA)
	tst.AssertStatusCode(t, rrGetChannelsA.Code, http.StatusOK)

	var parsedRespA map[string]interface{}
	json.Unmarshal(rrGetChannelsA.Body.Bytes(), &parsedRespA)
	dataABytes, _ := json.Marshal(parsedRespA["data"])
	var channelsA []ChannelItem
	json.Unmarshal(dataABytes, &channelsA)

	foundPublicA := false
	foundPrivateA := false

	for _, ch := range channelsA {
		if ch.Name == publicChannelName {
			foundPublicA = true
			if ch.Access != true {
				t.Errorf("User A should have access=true for public channel %s, got %v", publicChannelName, ch.Access)
			}
		}
		if ch.ID == privateChannelId {
			foundPrivateA = true
			if ch.Access != true {
				t.Errorf("User A should have access=true for private channel %s, got %v", privateChannelId, ch.Access)
			}
		}
	}

	if !foundPublicA {
		t.Errorf("User A should see public channel %s", publicChannelName)
	}
	if !foundPrivateA {
		t.Errorf("User A should see private channel %s", privateChannelId)
	}

	reqGetChannelsC, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/organisations/%s/channels", orgId), nil)
	reqGetChannelsC.Header.Set("Authorization", "Bearer "+tokenC)
	rrGetChannelsC := httptest.NewRecorder()
	r.ServeHTTP(rrGetChannelsC, reqGetChannelsC)
	tst.AssertStatusCode(t, rrGetChannelsC.Code, http.StatusOK)

	var parsedRespC map[string]interface{}
	json.Unmarshal(rrGetChannelsC.Body.Bytes(), &parsedRespC)
	dataCBytes, _ := json.Marshal(parsedRespC["data"])
	var channelsC []ChannelItem
	json.Unmarshal(dataCBytes, &channelsC)

	foundPublicC := false
	foundPrivateC := false

	for _, ch := range channelsC {
		if ch.Name == publicChannelName {
			foundPublicC = true
			if ch.Access != false {
				t.Errorf("User C should not have access for public channel %s, got %v", publicChannelName, ch.Access)
			}
		}
		if ch.ID == privateChannelId {
			foundPrivateC = true
			if ch.Access != false {
				t.Errorf("User C should not have access for private channel %s, got %v", privateChannelId, ch.Access)
			}
		}
	}

	if !foundPublicC {
		t.Errorf("User C should see public channel %s", publicChannelName)
	}
	if foundPrivateC {
		t.Errorf("User C should not see private channel %s", privateChannelId)
	}
}
