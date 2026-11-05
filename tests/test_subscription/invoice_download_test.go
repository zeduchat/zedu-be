package test_subscription

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	tst "github.com/hngprojects/telex_be/tests"
	"github.com/hngprojects/telex_be/utility"
)

func TestDownloadInvoice_NonMemberForbidden(t *testing.T) {
	r, authController, _, _, db := SetupSubscriptionTestRouter()

	// User A with Org A
	_, orgIDA, _, _ := CreateUserWithOrganization(t, r, authController, db)

	// User B with Org B
	_, _, tokenB, _ := CreateUserWithOrganization(t, r, authController, db)

	// User B attempts to download invoice for Org A
	dummySessionID := fmt.Sprintf("cs_test_%s", utility.RandomString(24))
	req, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/subscriptions/invoice/download/%s/%s", dummySessionID, orgIDA), nil)
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %v", tokenB))

	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	tst.AssertStatusCode(t, rr.Code, http.StatusForbidden)
}
