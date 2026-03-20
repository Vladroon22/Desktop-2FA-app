package core

import (
	"crypto/rand"
	"encoding/base32"
	"errors"
	"fmt"
	"net/url"
	"strings"
)

var (
	ErrEmptyURI         = errors.New("empty URI")
	ErrInvalidURI       = errors.New("invalid URI format")
	ErrWrongURI         = errors.New("wrong URI scheme or host")
	ErrSecretEmpty      = errors.New("secret is empty")
	ErrTooShortSecret   = errors.New("secret is too short")
	ErrWrongSecret      = errors.New("invalid secret format")
	ErrInvalidPath      = errors.New("invalid path format")
	ErrEmptyAccountName = errors.New("account name is empty")
	ErrEmptyIssuer      = errors.New("issuer name is empty")
	ErrConfigEmpty      = errors.New("wrong config")
)

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

func ValidateSecret(secret string) error {
	secret = strings.ToUpper(strings.ReplaceAll(secret, " ", ""))

	if len(secret) < 16 {
		return ErrTooShortSecret
	}

	allowedChars := "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	for _, c := range secret {
		if !strings.ContainsRune(allowedChars, c) {
			return fmt.Errorf(": %c", c)
		}
	}

	if _, err := base32.StdEncoding.WithPadding(base32.NoPadding).DecodeString(secret); err != nil {
		return fmt.Errorf("incorrect format of base32: %w", err)
	}

	return nil
}
