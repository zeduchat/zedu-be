package test_profile

import (
	"testing"

	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/pkg/repository/storage"
	tst "github.com/hngprojects/telex_be/tests"
	"github.com/hngprojects/telex_be/utility"
)

func TestCreateInitialDefaultProfileNonExistentUser(t *testing.T) {
	logger := tst.Setup()
	db := storage.Connection()

	nonExistentUserID := utility.GenerateUUID()

	_, err := models.CreateInitialDefaultProfile(db.Postgresql, nonExistentUserID, "", logger)
	if err == nil {
		t.Fatalf("expected error when creating profile for non-existent user %s, got nil", nonExistentUserID)
	}

	expectedErrMsg := "user " + nonExistentUserID + " does not exist, cannot create profile"
	if err.Error() != expectedErrMsg {
		t.Errorf("expected error %q, got %q", expectedErrMsg, err.Error())
	}
}
