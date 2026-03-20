package core

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerateOTP(t *testing.T) {
	tests := []struct {
		name     string
		time     time.Time
		secret   string
		wantCode bool
		wantErr  bool
	}{
		{
			name:     "success",
			time:     time.Now(),
			secret:   "JBSWY3DPEHPK3PXP",
			wantCode: true,
			wantErr:  false,
		},
		{
			name:     "empty secret",
			time:     time.Now(),
			secret:   "",
			wantCode: false,
			wantErr:  true,
		},
		{
			name:     "invalid secret",
			time:     time.Now(),
			secret:   "INVALID_BASE32!",
			wantCode: false,
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			code, err := GenerateOTP(tt.time, tt.secret)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Empty(t, code)
			} else {
				assert.NoError(t, err)
				assert.NotEmpty(t, code)
				assert.Len(t, code, 6)
				assert.Regexp(t, "^[0-9]{6}$", code)
			}
		})
	}
}

func TestGenerateOTP_TimeBased(t *testing.T) {
	secret := "JBSWY3DPEHPK3PXP"
	now := time.Now()

	code1, err := GenerateOTP(now, secret)
	require.NoError(t, err)

	code2, err := GenerateOTP(now.Add(30*time.Second), secret)
	require.NoError(t, err)

	if code1 == code2 {
		code3, err := GenerateOTP(now.Add(35*time.Second), secret)
		require.NoError(t, err)
		assert.NotEqual(t, code1, code3)
	}
}

func TestParseOTPAuthURI(t *testing.T) {
	tests := []struct {
		name          string
		uri           string
		wantSecret    string
		wantConfig    OTPConfig
		wantErrString error
	}{
		{
			name:          "secret is too short",
			uri:           "otpauth://totp/Example:user@example.com?secret=JBSWsdY3DPEH&issuer=Example",
			wantErrString: ErrTooShortSecret,
		},
		{
			name:       "parsing with ACCOUNT",
			uri:        "otpauth://totp/Example:user?secret=JBSWY3DPEHPK3PXP&account=test",
			wantSecret: "JBSWY3DPEHPK3PXP",
			wantConfig: OTPConfig{
				Issuer:      "",
				AccountName: "user",
			},
			wantErrString: nil,
		},
		{
			name:          "parsing wrong host",
			uri:           "otpauth://hotp/Example?secret=JBSWY3DPEHPK3PXP",
			wantSecret:    "",
			wantConfig:    OTPConfig{},
			wantErrString: ErrConfigEmpty,
		},
		{
			name:          "Wrong URI #1",
			uri:           "https://example.com",
			wantSecret:    "",
			wantConfig:    OTPConfig{},
			wantErrString: ErrConfigEmpty,
		},
		{
			name:          "Wrong URI #2",
			uri:           "invalid-uri-string",
			wantSecret:    "",
			wantConfig:    OTPConfig{},
			wantErrString: ErrConfigEmpty,
		},
		{
			name:          "parsing without secret",
			uri:           "otpauth://totp/Example",
			wantConfig:    OTPConfig{},
			wantErrString: ErrTooShortSecret,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, err := ParseOTPAuthURI(tt.uri)
			if tt.wantErrString != nil {
				assert.Error(t, err)
				assert.ErrorIs(t, err, tt.wantErrString)
			} else {
				assert.NoError(t, err)
			}

			t.Logf("want: %v actual: %v", tt.wantErrString, err)
		})
	}
}
