package test_buzz

import (
	"testing"

	"github.com/hngprojects/telex_be/pkg/repository/agora"
	tst "github.com/hngprojects/telex_be/tests"
)

// TestAgoraServiceInitialization tests that the Agora service is properly initialized
func TestAgoraServiceInitialization(t *testing.T) {
	logger := tst.Setup()

	// Verify Agora client is initialized
	if agora.Client.Service == nil {
		t.Fatal("Agora service was not initialized in test setup")
	}

	// Verify we can get the App ID
	appId := agora.Client.Service.GetAppId()
	if appId == "" {
		t.Error("Agora App ID is empty")
	}

	// Verify service health
	if !agora.Client.Service.IsHealthy() {
		t.Error("Agora service is not healthy")
	}

	logger.Info("Agora service initialized with App ID: %s", appId)
}

// TestAgoraTokenGeneration tests token generation functionality
func TestAgoraTokenGeneration(t *testing.T) {
	logger := tst.Setup()

	service := agora.Client.Service
	if service == nil {
		t.Fatal("Agora service not initialized")
	}

	// Test parameters
	channelName := "test-channel-123"
	userID := "test-user-456"
	uid := "test-uid-789"
	expireTime := uint32(3600) // 1 hour

	// Generate token
	token, err := service.GenerateRTCToken(channelName, userID, uid, expireTime)
	if err != nil {
		t.Fatalf("Failed to generate RTC token: %v", err)
	}

	if token == "" {
		t.Error("Generated token is empty")
	}

	// Token should be a long string (Agora tokens are typically 200+ characters)
	if len(token) < 100 {
		t.Errorf("Token seems too short: %d characters", len(token))
	}

	logger.Info("Successfully generated token of length: %d", len(token))
}

// TestAgoraDefaultTokenExpiration tests that we're using 4-hour expiration by default
func TestAgoraDefaultTokenExpiration(t *testing.T) {
	logger := tst.Setup()

	service := agora.Client.Service
	if service == nil {
		t.Fatal("Agora service not initialized")
	}

	// Test parameters
	channelName := "test-channel-4h"
	userID := "test-user-4h"
	uid := "test-uid-4h"
	expireTime := uint32(agora.DefaultTokenExpirationSeconds) // 4 hours

	// Generate token with 4-hour expiration
	token, err := service.GenerateRTCToken(channelName, userID, uid, expireTime)
	if err != nil {
		t.Fatalf("Failed to generate RTC token with 4-hour expiration: %v", err)
	}

	if token == "" {
		t.Error("Generated token is empty")
	}

	logger.Info("Successfully generated token with 4-hour expiration (length: %d)", len(token))
}

// TestAgoraServiceHealth tests the health check functionality
func TestAgoraServiceHealth(t *testing.T) {
	logger := tst.Setup()

	service := agora.Client.Service
	if service == nil {
		t.Fatal("Agora service not initialized")
	}

	// Check service health
	if !service.IsHealthy() {
		t.Error("Agora service should be healthy when properly configured")
	}

	// Verify App ID is set
	appId := service.GetAppId()
	if appId == "" {
		t.Error("App ID should not be empty for healthy service")
	}

	logger.Info("Agora service health check passed")
}
