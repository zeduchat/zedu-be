package utility

import (
	"crypto/sha256"
	"encoding/hex"
	"sort"
	"strings"
)

func GenerateParticipantHash(creatorID string, participants []string) ([]string, string) {

	allParticipants := append([]string{creatorID}, participants...)

	sort.Strings(allParticipants)

	particpantString := strings.Join(allParticipants, ":")
	hasher := sha256.New()
	hasher.Write([]byte(particpantString))
	return allParticipants, hex.EncodeToString(hasher.Sum(nil))
}
