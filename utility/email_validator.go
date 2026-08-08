package utility

import (
	"context"
	"errors"
	"net"
	"net/mail"
	"strings"
	"time"
)

var disposableDomains = map[string]bool{
	"mailinator.com":     true,
	"tempmail.com":       true,
	"10minutemail.com":   true,
	"yopmail.com":        true,
	"guerrillamail.com":  true,
	"dispostable.com":    true,
	"trashmail.com":      true,
	"getnada.com":        true,
	"throwawaymail.com":  true,
	"maildrop.cc":        true,
	"sharklasers.com":    true,
	"guerrillamail.block": true,
	"disposable.com":     true,
}

func ValidateSignupEmail(email string) (string, error) {
	email = strings.ToLower(strings.TrimSpace(email))

	if len(email) == 0 || len(email) > 254 {
		return "", errors.New("email address is invalid")
	}

	parts := strings.Split(email, "@")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", errors.New("email address is invalid")
	}

	if len(parts[0]) > 64 {
		return "", errors.New("email address is invalid")
	}

	if _, err := mail.ParseAddress(email); err != nil {
		return "", errors.New("email address is invalid")
	}

	domain := parts[1]
	if disposableDomains[domain] {
		return "", errors.New("disposable email domains are not allowed for registration")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	mxs, err := net.DefaultResolver.LookupMX(ctx, domain)
	if err != nil || len(mxs) == 0 {
		return "", errors.New("email address is invalid")
	}

	return email, nil
}
