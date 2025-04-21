package utility

import (
	"crypto/sha256"
	"encoding/hex"
	"sort"
	"strings"
)

func GenerateParticipantHash(participants []string) ([]string, string) {

	sort.Strings(participants)

	particpantString := strings.Join(participants, ":")
	hasher := sha256.New()
	hasher.Write([]byte(particpantString))
	return participants, hex.EncodeToString(hasher.Sum(nil))
}
