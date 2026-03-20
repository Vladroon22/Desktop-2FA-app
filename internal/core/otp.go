package core

import (
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/pquerna/otp/totp"
)

type OTPConfig struct {
	Issuer      string
	AccountName string
}

func GenerateOTP(tm time.Time, secret string) (string, error) {
	if secret == "" {
		return "", ErrSecretEmpty
	}

	code, err := totp.GenerateCode(secret, tm)
	if err != nil {
		fmt.Println(err)
		return "", err
	}

	return code, nil
}

func ParseOTPAuthURI(uri string) (string, OTPConfig, error) {
	if uri == "" {
		return "", OTPConfig{}, ErrEmptyURI
	}

	u, err := url.Parse(uri)
	if err != nil {
		return "", OTPConfig{}, ErrInvalidURI
	}

	if u.Scheme != "otpauth" || u.Host != "totp" {
		return "", OTPConfig{}, ErrConfigEmpty
	}

	query := u.Query()
	secret := query.Get("secret")
	config := OTPConfig{
		AccountName: query.Get("ACCOUNT"),
		Issuer:      query.Get("issuer"),
	}

	if len(secret) < 16 {
		return "", OTPConfig{}, ErrTooShortSecret
	}

	path := strings.TrimPrefix(u.Path, "/")
	parts := strings.Split(path, ":")
	if len(parts) > 1 {
		config.AccountName = parts[1]
	}

	return secret, config, nil
}
