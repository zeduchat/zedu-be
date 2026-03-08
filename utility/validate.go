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
	"time"

	"github.com/go-playground/validator/v10"
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
	}

	blockedDomains = append(blockedDomains, additionalDomains...)
	return &URLValidator{
		blockedDomains: blockedDomains,
	}
}
func (v *URLValidator) Validate(urlStr string) error {
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

func ValidateDocument(doc map[string]any) error {
	if len(doc) == 0 {
		return errors.New("document cannot be empty")
	}

	for key, value := range doc {
		switch v := value.(type) {
		case string:
			if v == "" {
				return fmt.Errorf("field '%s' cannot be an empty string", key)
			}
		case map[string]any:
			if err := ValidateDocument(v); err != nil {
				return fmt.Errorf("field '%s': %v", key, err)
			}
		case []any:
			for i, item := range v {
				if nestedMap, ok := item.(map[string]any); ok {
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

// emojiRegex matches Unicode emoji characters using standard Unicode property classes
// \p{So} = Symbol, other (includes most emojis)
// \p{Sk} = Symbol, modifier (includes skin tones and combining marks)
// Also includes zero-width joiner (ZWJ) and variation selectors for emoji sequences
var emojiRegex = regexp.MustCompile(`^[\p{So}\p{Sk}\x{200D}\x{FE0F}\x{20E3}]*$`)

// IsValidEmoji validates that a string contains only valid Unicode emoji characters.
// It uses regex with Unicode property classes to match emoji characters.
// This is a public function that can be called from other packages for manual validation.
func IsValidEmoji(s string) bool {
	if s == "" {
		return true // Empty string is valid (optional field)
	}
	// Use regex to match emoji characters using Unicode property classes
	// \p{So} = Symbol, other (includes most emojis)
	// \p{Sk} = Symbol, modifier (includes skin tones)
	// \x{200D} = Zero Width Joiner (for emoji sequences)
	// \x{FE0F} = Variation Selector-16 (emoji presentation)
	// \x{20E3} = Combining Enclosing Keycap
	return emojiRegex.MatchString(s)
}

func RegisterCustomValidations(v *validator.Validate) {

	_ = v.RegisterValidation("timezone", func(fl validator.FieldLevel) bool {
		tz := fl.Field().String()
		_, err := time.LoadLocation(tz)
		return err == nil
	})

	_ = v.RegisterValidation("no_whitespace", func(fl validator.FieldLevel) bool {
		value := fl.Field().String()
		return !strings.ContainsAny(value, " \t\n\r")
	})

	_ = v.RegisterValidation("emoji", func(fl validator.FieldLevel) bool {
		value := fl.Field().String()
		return IsValidEmoji(value)
	})

	_ = v.RegisterValidation("status_expiry", func(fl validator.FieldLevel) bool {
		value := fl.Field().String()
		if value == "" {
			return true
		}

		lowerValue := strings.ToLower(strings.TrimSpace(value))

		validOptions := []string{
			"30 minutes",
			"30 minute",
			"1 hour",
			"1 hr",
			"today",
			"this week",
			"don't remove",
			"dont remove",
			"do not remove",
		}

		for _, opt := range validOptions {
			if lowerValue == opt {
				return true
			}
		}
		return false
	})
}

func ValidateTimeRange(value string) bool {

	// Validate format using regex
	re := regexp.MustCompile(`^(0?[1-9]|1[0-2]):[0-5][0-9]\s?(AM|PM)\s?-\s?(0?[1-9]|1[0-2]):[0-5][0-9]\s?(AM|PM)$`)
	if !re.MatchString(value) {
		return false
	}

	// Optional: validate logical order
	parts := strings.Split(value, "-")
	if len(parts) != 2 {
		return false
	}
	_, err1 := time.Parse("3:04 PM", strings.TrimSpace(parts[0]))
	_, err2 := time.Parse("3:04 PM", strings.TrimSpace(parts[1]))

	if err1 != nil || err2 != nil {
		return false
	}
	return true
}
func IsValidURL(urlStr string) bool {
	u, err := url.ParseRequestURI(urlStr)
	if err != nil {
		return false
	}

	scheme := strings.ToLower(u.Scheme)
	if scheme != "http" && scheme != "https" {
		return false
	}

	if u.Host == "" {
		return false
	}

	if !strings.Contains(u.Host, ".") {
		return false
	}

	return true
}
