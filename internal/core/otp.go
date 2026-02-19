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
	code, err := totp.GenerateCode(secret, tm)
	if err != nil {
		fmt.Println(err)
		return "", err
	}

	return code, nil
}

func ParseOTPAuthURI(uri string) (string, OTPConfig, error) {
	u, err := url.Parse(uri)
	if err != nil {
		return "", OTPConfig{}, err
	}

	if u.Scheme != "otpauth" || u.Host != "totp" {
		return "", OTPConfig{}, fmt.Errorf("неверный URI схемы")
	}

	query := u.Query()
	secret := query.Get("secret")
	config := OTPConfig{
		AccountName: query.Get("ACCOUNT"),
		Issuer:      query.Get("issuer"),
	}

	path := strings.TrimPrefix(u.Path, "/")
	parts := strings.Split(path, ":")
	if len(parts) > 1 {
		config.AccountName = parts[1]
	}

	return secret, config, nil
}
