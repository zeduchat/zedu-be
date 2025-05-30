package utility

import (
	"errors"
	"fmt"
	"net/mail"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/microcosm-cc/bluemonday"
	"github.com/nyaruka/phonenumbers"
)

type URLValidator struct {
	blockedDomains []string
}

func NewURLValidator(additionalDomains ...string) *URLValidator {
	blockedDomains := []string{
		// Ngrok domains
		"ngrok.io",
		"ngrok.app",
		"ngrok-free.app",

		// Render Domains
		"render.com",
		"render.app",
		"onrender.com",
		"app.render.com",
	}

	blockedDomains = append(blockedDomains, additionalDomains...)
	return &URLValidator{
		blockedDomains: blockedDomains,
	}
}
func (v *URLValidator) Validate (urlStr string) error {
	parsedURL, err := url.Parse(urlStr)
	if err != nil {
		return fmt.Errorf("invalid URL format: %w", err)
	}

	if parsedURL.Host == "" {
		return errors.New("URL must contain a host")
	}
	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		return errors.New("URL must start with http:// or https://")
	}
	for _, domain := range v.blockedDomains {
		if strings.Contains(parsedURL.Host, domain) {
			return fmt.Errorf("URL contains blocked domain: %s", domain)
		}
	}
	return nil
}

func EmailValid(email string) (string, bool) {
	// made some change to parse the formated email
	e, err := mail.ParseAddress(email)
	if err != nil {
		return "", false
	}
	return e.Address, err == nil
}

func PhoneValid(phone string) (string, bool) {
	parsed, err := phonenumbers.Parse(phone, "")
	if err != nil {
		return phone, false
	}

	if !phonenumbers.IsValidNumber(parsed) {
		return phone, false
	}

	formattedNum := phonenumbers.Format(parsed, phonenumbers.NATIONAL)
	return formattedNum, true
}

func fileExists(filename string) bool {
	info, err := os.Stat(filename)
	if os.IsNotExist(err) {
		return false
	}
	return !info.IsDir()
}

func CleanStringInput(input string) string {
	policy := bluemonday.UGCPolicy()
	cleanedInput := policy.Sanitize(input)
	re := regexp.MustCompile(`[^\w\s]`)
	cleanedInput = re.ReplaceAllString(cleanedInput, "")

	return cleanedInput
}

func SplitEmailString(email string) string {
	arr := strings.Split(email, "@")
	name := arr[0]

	p := len(name) - 1

	for p > 0 {
		_, err := strconv.Atoi(string(name[p]))
		if err != nil {
			break
		}
		p--
	}

	if p == 0 {
		return ""
	}
	return name[:p+1]
}

func ArchiveValidator(archived bool) error {
	if !archived && archived {
		return errors.New("archived field must be provided")
	}
	return nil
}

func ValidateDocument(doc map[string]interface{}) error {
	if len(doc) == 0 {
		return errors.New("document cannot be empty")
	}

	for key, value := range doc {
		switch v := value.(type) {
		case string:
			if v == "" {
				return fmt.Errorf("field '%s' cannot be an empty string", key)
			}
		case map[string]interface{}:
			if err := ValidateDocument(v); err != nil {
				return fmt.Errorf("field '%s': %v", key, err)
			}
		case []interface{}:
			for i, item := range v {
				if nestedMap, ok := item.(map[string]interface{}); ok {
					if err := ValidateDocument(nestedMap); err != nil {
						return fmt.Errorf("field '%s[%d]': %v", key, i, err)
					}
				} else if str, ok := item.(string); ok && str == "" {
					return fmt.Errorf("field '%s[%d]' cannot be an empty string", key, i)
				}
			}
		}
	}
	return nil
}
