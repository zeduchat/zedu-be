package test_fcm

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/pkg/repository/storage"
	tst "github.com/hngprojects/telex_be/tests"
	"github.com/hngprojects/telex_be/utility"
)

func TestUserDeviceToken_MultiPlatformSupport(t *testing.T) {
	tst.Setup()
	db := storage.Connection()

	t.Cleanup(func() {
		tst.Cleanup(db)
	})

	user := CreateTestUser(t, db)

	platforms := []string{"web", "android", "ios"}
	tokens := make([]string, len(platforms))

	for i, plat := range platforms {
		tokens[i] = "fcm_token_" + plat + "_" + utility.GenerateUUID()
		devToken := models.UserDeviceToken{
			UserId:   user.ID,
			Token:    tokens[i],
			Platform: plat,
		}
		err := devToken.Upsert(db.Postgresql)
		require.NoError(t, err)
	}

	fetchedTokens, err := models.ListDeviceTokensByUserIDs(db.Postgresql, []string{user.ID})
	require.NoError(t, err)
	assert.Equal(t, 3, len(fetchedTokens), "expected all 3 platform tokens to be returned for the user")

	platformMap := make(map[string]string)
	for _, ft := range fetchedTokens {
		platformMap[ft.Platform] = ft.Token
	}

	for i, plat := range platforms {
		assert.Equal(t, tokens[i], platformMap[plat], "token for platform %s should match", plat)
	}
}

func TestUserDeviceToken_SamePlatform_UpdatesToken(t *testing.T) {
	tst.Setup()
	db := storage.Connection()

	t.Cleanup(func() {
		tst.Cleanup(db)
	})

	user := CreateTestUser(t, db)

	for _, plat := range []string{"web", "android", "ios"} {
		tokenFirst := "fcm_token_initial_" + plat + "_" + utility.GenerateUUID()
		tokenSecond := "fcm_token_updated_" + plat + "_" + utility.GenerateUUID()

		dev1 := models.UserDeviceToken{UserId: user.ID, Token: tokenFirst, Platform: plat}
		dev2 := models.UserDeviceToken{UserId: user.ID, Token: tokenSecond, Platform: plat}

		require.NoError(t, dev1.Upsert(db.Postgresql))
		require.NoError(t, dev2.Upsert(db.Postgresql))

		fetched, err := models.ListDeviceTokensByUserIDs(db.Postgresql, []string{user.ID})
		require.NoError(t, err)

		var matched []models.UserDeviceToken
		for _, token := range fetched {
			if token.Platform == plat {
				matched = append(matched, token)
			}
		}

		require.Equal(t, 1, len(matched), "expected only 1 token for platform %s", plat)
		assert.Equal(t, tokenSecond, matched[0].Token, "expected token to be updated to latest value for %s", plat)
	}
}

func TestUserDeviceToken_Upsert_Atomic(t *testing.T) {
	tst.Setup()
	db := storage.Connection()

	t.Cleanup(func() {
		tst.Cleanup(db)
	})

	user := CreateTestUser(t, db)
	sharedToken := "fcm_shared_token_" + utility.GenerateUUID()

	var wg sync.WaitGroup
	errCh := make(chan error, 10)

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			dev := models.UserDeviceToken{
				UserId:   user.ID,
				Token:    sharedToken,
				Platform: "android",
			}
			if err := dev.Upsert(db.Postgresql); err != nil {
				errCh <- err
			}
		}()
	}

	wg.Wait()
	close(errCh)

	for err := range errCh {
		assert.NoError(t, err, "atomic upsert should not fail on concurrent calls")
	}

	var count int64
	db.Postgresql.Model(&models.UserDeviceToken{}).Where("token = ?", sharedToken).Count(&count)
	assert.Equal(t, int64(1), count)
}

func TestUserDeviceToken_ReassignUser(t *testing.T) {
	tst.Setup()
	db := storage.Connection()

	t.Cleanup(func() {
		tst.Cleanup(db)
	})

	userA := CreateTestUser(t, db)
	userB := CreateTestUser(t, db)
	tokenVal := "fcm_device_token_" + utility.GenerateUUID()

	devA := models.UserDeviceToken{UserId: userA.ID, Token: tokenVal, Platform: "android"}
	require.NoError(t, devA.Upsert(db.Postgresql))

	tokensA, err := models.ListDeviceTokensByUserIDs(db.Postgresql, []string{userA.ID})
	require.NoError(t, err)
	assert.Equal(t, 1, len(tokensA))
	assert.Equal(t, tokenVal, tokensA[0].Token)

	devB := models.UserDeviceToken{UserId: userB.ID, Token: tokenVal, Platform: "android"}
	require.NoError(t, devB.Upsert(db.Postgresql))

	tokensAfterA, err := models.ListDeviceTokensByUserIDs(db.Postgresql, []string{userA.ID})
	require.NoError(t, err)
	assert.Equal(t, 0, len(tokensAfterA), "token should no longer belong to user A")

	tokensAfterB, err := models.ListDeviceTokensByUserIDs(db.Postgresql, []string{userB.ID})
	require.NoError(t, err)
	assert.Equal(t, 1, len(tokensAfterB), "token should now belong to user B")
	assert.Equal(t, tokenVal, tokensAfterB[0].Token)
}

func TestUserDeviceToken_DeleteDeviceToken_SpecificDeviceOnly(t *testing.T) {
	tst.Setup()
	db := storage.Connection()

	t.Cleanup(func() {
		tst.Cleanup(db)
	})

	user := CreateTestUser(t, db)

	tokenWeb := "fcm_token_web_" + utility.GenerateUUID()
	tokenAndroid := "fcm_token_android_" + utility.GenerateUUID()
	tokenIOS := "fcm_token_ios_" + utility.GenerateUUID()

	require.NoError(t, (&models.UserDeviceToken{UserId: user.ID, Token: tokenWeb, Platform: "web"}).Upsert(db.Postgresql))
	require.NoError(t, (&models.UserDeviceToken{UserId: user.ID, Token: tokenAndroid, Platform: "android"}).Upsert(db.Postgresql))
	require.NoError(t, (&models.UserDeviceToken{UserId: user.ID, Token: tokenIOS, Platform: "ios"}).Upsert(db.Postgresql))

	require.NoError(t, models.DeleteDeviceToken(db.Postgresql, user.ID, tokenWeb))

	remaining, err := models.ListDeviceTokensByUserIDs(db.Postgresql, []string{user.ID})
	require.NoError(t, err)
	assert.Equal(t, 2, len(remaining), "only the web token should have been deleted")

	tokensLeft := map[string]bool{
		remaining[0].Token: true,
		remaining[1].Token: true,
	}
	assert.False(t, tokensLeft[tokenWeb])
	assert.True(t, tokensLeft[tokenAndroid])
	assert.True(t, tokensLeft[tokenIOS])
}

func TestUserDeviceToken_DeleteDeviceTokens_Batch(t *testing.T) {
	tst.Setup()
	db := storage.Connection()

	t.Cleanup(func() {
		tst.Cleanup(db)
	})

	user := CreateTestUser(t, db)

	token1 := "fcm_invalid_1_" + utility.GenerateUUID()
	token2 := "fcm_invalid_2_" + utility.GenerateUUID()
	token3 := "fcm_valid_3_" + utility.GenerateUUID()

	require.NoError(t, (&models.UserDeviceToken{UserId: user.ID, Token: token1, Platform: "web"}).Upsert(db.Postgresql))
	require.NoError(t, (&models.UserDeviceToken{UserId: user.ID, Token: token2, Platform: "android"}).Upsert(db.Postgresql))
	require.NoError(t, (&models.UserDeviceToken{UserId: user.ID, Token: token3, Platform: "ios"}).Upsert(db.Postgresql))

	require.NoError(t, models.DeleteDeviceTokens(db.Postgresql, []string{token1, token2}))

	remaining, err := models.ListDeviceTokensByUserIDs(db.Postgresql, []string{user.ID})
	require.NoError(t, err)
	assert.Equal(t, 1, len(remaining))
	assert.Equal(t, token3, remaining[0].Token)
}
