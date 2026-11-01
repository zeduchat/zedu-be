package utility

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"net/url"
	"strings"
)

const newsletterUnsubscribePurpose = "newsletter-unsubscribe:"

// NewsletterUnsubscribeToken returns a URL-safe HMAC of the email address.
// The purpose prefix keeps the token from being valid for anything else
// that is signed with the same server secret.
func NewsletterUnsubscribeToken(email, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(newsletterUnsubscribePurpose + strings.ToLower(strings.TrimSpace(email))))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}

// ValidNewsletterUnsubscribeToken checks a token in constant time.
func ValidNewsletterUnsubscribeToken(email, token, secret string) bool {
	if secret == "" || token == "" {
		return false
	}
	expected := NewsletterUnsubscribeToken(email, secret)
	return hmac.Equal([]byte(expected), []byte(token))
}

// NewsletterUnsubscribeLink builds the web page link that goes into newsletter emails.
func NewsletterUnsubscribeLink(frontendURL, email, secret string) string {
	email = strings.ToLower(strings.TrimSpace(email))
	query := url.Values{}
	query.Set("email", email)
	query.Set("token", NewsletterUnsubscribeToken(email, secret))
	return fmt.Sprintf("%v/unsubscribe?%v", strings.TrimRight(frontendURL, "/"), query.Encode())
}
