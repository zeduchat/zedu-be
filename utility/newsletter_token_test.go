package utility

import (
	"strings"
	"testing"
)

func TestNewsletterUnsubscribeToken(t *testing.T) {
	const secret = "test-secret"

	token := NewsletterUnsubscribeToken("User@Example.com", secret)

	if !ValidNewsletterUnsubscribeToken("user@example.com", token, secret) {
		t.Fatal("token should be valid for the same email, ignoring case")
	}
	if ValidNewsletterUnsubscribeToken("other@example.com", token, secret) {
		t.Fatal("token should not be valid for a different email")
	}
	if ValidNewsletterUnsubscribeToken("user@example.com", token, "another-secret") {
		t.Fatal("token should not be valid with a different secret")
	}
	if ValidNewsletterUnsubscribeToken("user@example.com", "", secret) {
		t.Fatal("an empty token should be rejected")
	}
	if ValidNewsletterUnsubscribeToken("user@example.com", token, "") {
		t.Fatal("an empty secret should be rejected")
	}
}

func TestNewsletterUnsubscribeLink(t *testing.T) {
	link := NewsletterUnsubscribeLink("https://zedu.chat/", "User+tag@Example.com", "test-secret")

	if !strings.HasPrefix(link, "https://zedu.chat/unsubscribe?") {
		t.Fatalf("unexpected link %q", link)
	}
	if !strings.Contains(link, "email=user%2Btag%40example.com") {
		t.Fatalf("email should be lowercased and escaped in %q", link)
	}
	token := NewsletterUnsubscribeToken("user+tag@example.com", "test-secret")
	if !strings.Contains(link, "token="+token) {
		t.Fatalf("link %q should carry the token", link)
	}
}

func TestNewsletterConfirmToken(t *testing.T) {
	const secret = "test-secret"

	token := NewsletterConfirmToken("User@Example.com", secret)

	if !ValidNewsletterConfirmToken("user@example.com", token, secret) {
		t.Fatal("token should be valid for the same email, ignoring case")
	}
	if ValidNewsletterConfirmToken("other@example.com", token, secret) {
		t.Fatal("token should not be valid for a different email")
	}
	if ValidNewsletterUnsubscribeToken("user@example.com", token, secret) {
		t.Fatal("a confirmation token must not work as an unsubscribe token")
	}
	if ValidNewsletterConfirmToken("user@example.com", NewsletterUnsubscribeToken("user@example.com", secret), secret) {
		t.Fatal("an unsubscribe token must not work as a confirmation token")
	}
}

func TestNewsletterConfirmLink(t *testing.T) {
	link := NewsletterConfirmLink("https://zedu.chat", "User@Example.com", "test-secret")

	if !strings.HasPrefix(link, "https://zedu.chat/confirm-subscription?") {
		t.Fatalf("unexpected link %q", link)
	}
	if !strings.Contains(link, "token="+NewsletterConfirmToken("user@example.com", "test-secret")) {
		t.Fatalf("link %q should carry the confirmation token", link)
	}
}
