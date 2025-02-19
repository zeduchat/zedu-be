package utility

import (
	"encoding/base64"
	"fmt"
	"math/rand"
	"strings"

	"golang.org/x/crypto/bcrypt"
	"golang.org/x/crypto/chacha20poly1305"
)

func HashPassword(str string) (string, error) {
	hashed, err := bcrypt.GenerateFromPassword([]byte(str), bcrypt.DefaultCost)
	return string(hashed), err
}

func CompareHash(str string, hashed string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hashed), []byte(str)) == nil
}

// Generate external Apikey
func CreateExternalApiKey(org_id, integration_id, enc_key string) (string, error) {

	api_key, err := Encrypt(org_id, integration_id, []byte(enc_key))

	if err != nil {
		return "", err
	}

	return api_key, nil
}

// Validate External ApiKey

func ValidateExternalApiKey(api_key string, enc_key string) (string, string, error) {

	return Decrypt(api_key, []byte(enc_key))
}

// ExtractLastPart extracts the last section of the given UUID
func ExtractLastPart(id string) string {
	parts := strings.Split(id, "-")
	return parts[len(parts)-1]
}

// Encrypt concatenates last parts and encrypts the result
func Encrypt(id1, id2 string, key []byte) (string, error) {

	id1 = ExtractLastPart(id1)
	id2 = ExtractLastPart(id2)

	if len(key) != chacha20poly1305.KeySize {
		panic("Key is not supplied in env, or length not met")
	}

	cipher, err := chacha20poly1305.NewX(key)
	if err != nil {
		return "", err
	}

	nonce := make([]byte, cipher.NonceSize())
	_, err = rand.Read(nonce)
	if err != nil {
		return "", err
	}

	plaintext := id1[len(id1)-12:] + id2[len(id2)-12:]
	ciphertext := cipher.Seal(nonce, nonce, []byte(plaintext), nil)
	res := base64.StdEncoding.EncodeToString(ciphertext)
	res = "tlx-" + strings.ReplaceAll(res, "+", "-")
	return res[:len(res)-2], nil
}

// Decrypt decrypts the encrypted data and returns the extracted parts
func Decrypt(encrypted string, key []byte) (string, string, error) {

	if !strings.HasPrefix(encrypted, "tlx-") {
		return "", "", fmt.Errorf("Invalid api key supplied")
	}

	encrypted = encrypted[4:] + "=="

	encrypted = strings.ReplaceAll(encrypted, "-", "+")

	if len(key) != chacha20poly1305.KeySize {
		panic("Key is not supplied in env, or length not met")
	}

	cipher, err := chacha20poly1305.NewX(key)
	if err != nil {
		return "", "", fmt.Errorf("Internal server error")
	}

	ciphertext, err := base64.StdEncoding.DecodeString(encrypted)
	if err != nil {
		return "", "", fmt.Errorf("Invalid api key supplied")
	}

	nonceSize := cipher.NonceSize()
	if len(ciphertext) < nonceSize {
		return "", "", fmt.Errorf("Invalid api key supplied")
	}

	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]
	plaintext, err := cipher.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", "", fmt.Errorf("Invalid api key supplied")
	}

	return string(plaintext[:12]), string(plaintext[12:]), nil
}
