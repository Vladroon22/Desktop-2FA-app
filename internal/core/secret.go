package core

import (
	"crypto/rand"
	"encoding/base32"
	"fmt"
	"net/url"
	"strings"
)

// Generates crypto-graphical safe random secret for TOTP
func GenerateRandomSecret() (string, error) {
	secretLength := 20

	randomBytes := make([]byte, secretLength)
	if _, err := rand.Read(randomBytes); err != nil {
		return "", fmt.Errorf("failed to generate random bytes: %w", err)
	}

	secret := base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(randomBytes)

	return secret, nil
}

func GenerateSecretWithURI(issuer, accountName string) (secret, uri string, err error) {
	secret, err = GenerateRandomSecret()
	if err != nil {
		return "", "", err
	}

	uri = fmt.Sprintf("otpauth://totp/%s:%s?secret=%s&issuer=%s",
		url.QueryEscape(issuer),
		url.QueryEscape(accountName),
		secret,
		url.QueryEscape(issuer))

	return secret, uri, nil
}

// ValidateSecret check validity of secret
func ValidateSecret(secret string) error {
	secret = strings.ToUpper(strings.ReplaceAll(secret, " ", ""))

	// check len (minimum recommended - 16 symbols base32 = 80 бит)
	if len(secret) < 16 {
		return fmt.Errorf("секрет слишком короткий, минимум 16 символов")
	}

	allowedChars := "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	for _, c := range secret {
		if !strings.ContainsRune(allowedChars, c) {
			return fmt.Errorf(": %c", c)
		}
	}

	// decoding secret to validate
	if _, err := base32.StdEncoding.WithPadding(base32.NoPadding).DecodeString(secret); err != nil {
		return fmt.Errorf("incorrect format of base32: %w", err)
	}

	return nil
}
