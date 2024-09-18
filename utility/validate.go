package utility

import (
	"errors"
	"net/mail"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/microcosm-cc/bluemonday"
	"github.com/nyaruka/phonenumbers"
)

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
	if archived != true && archived != false {
		return errors.New("archived field must be provided")
	}
	return nil
}
