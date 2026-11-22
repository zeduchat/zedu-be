package utility

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"net/url"
	"strings"
)

const (
	newsletterUnsubscribePurpose = "newsletter-unsubscribe:"
	newsletterConfirmPurpose     = "newsletter-confirm:"
)

// newsletterToken returns a URL-safe HMAC of the email address. The purpose
// prefix keeps a token made for one action from working for another one, or
// for anything else that is signed with the same server secret.
func newsletterToken(purpose, email, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(purpose + strings.ToLower(strings.TrimSpace(email))))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}

func validNewsletterToken(purpose, email, token, secret string) bool {
	if secret == "" || token == "" {
		return false
	}
	expected := newsletterToken(purpose, email, secret)
	return hmac.Equal([]byte(expected), []byte(token))
}

func newsletterLink(frontendURL, path, purpose, email, secret string) string {
	email = strings.ToLower(strings.TrimSpace(email))
	query := url.Values{}
	query.Set("email", email)
	query.Set("token", newsletterToken(purpose, email, secret))
	return fmt.Sprintf("%v/%v?%v", strings.TrimRight(frontendURL, "/"), path, query.Encode())
}

// NewsletterUnsubscribeToken returns the token for the unsubscribe link in newsletter emails.
func NewsletterUnsubscribeToken(email, secret string) string {
	return newsletterToken(newsletterUnsubscribePurpose, email, secret)
}

// ValidNewsletterUnsubscribeToken checks an unsubscribe token in constant time.
func ValidNewsletterUnsubscribeToken(email, token, secret string) bool {
	return validNewsletterToken(newsletterUnsubscribePurpose, email, token, secret)
}

// NewsletterUnsubscribeLink builds the web page link that goes into newsletter emails.
func NewsletterUnsubscribeLink(frontendURL, email, secret string) string {
	return newsletterLink(frontendURL, "unsubscribe", newsletterUnsubscribePurpose, email, secret)
}

// NewsletterConfirmToken returns the token for the link in the confirmation email.
func NewsletterConfirmToken(email, secret string) string {
	return newsletterToken(newsletterConfirmPurpose, email, secret)
}

// ValidNewsletterConfirmToken checks a confirmation token in constant time.
func ValidNewsletterConfirmToken(email, token, secret string) bool {
	return validNewsletterToken(newsletterConfirmPurpose, email, token, secret)
}

// NewsletterConfirmLink builds the web page link that goes into the confirmation email.
func NewsletterConfirmLink(frontendURL, email, secret string) string {
	return newsletterLink(frontendURL, "confirm-subscription", newsletterConfirmPurpose, email, secret)
}
