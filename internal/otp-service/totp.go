package otpservice

import (
	"fmt"
	"sync"
	"time"

	"github.com/Vladroon22/2FA/internal/core"
)

type OTPService struct {
	mu    sync.Mutex
	codes map[string]OTPConfig
}

type OTPConfig struct {
	code     string
	timeExp  int
	verified bool
}

func NewOTPService() *OTPService {
	return &OTPService{
		codes: make(map[string]OTPConfig),
	}
}

func (srv *OTPService) SaveOTP(currTime time.Time, issuer, secret string) error {
	code, err := core.GenerateOTP(currTime, secret)
	if err != nil {
		return err
	}

	go srv.regenOTP(issuer, secret)

	srv.mu.Lock()
	defer srv.mu.Unlock()

	srv.codes[issuer] = OTPConfig{
		code:     code,
		verified: false,
		timeExp:  currTime.Second(),
	}
	return nil
}

func (srv *OTPService) ValidateOTP(currTime time.Time, enteredCode, issuer string) (bool, error) {
	srv.mu.Lock()
	defer srv.mu.Unlock()

	config, ok := srv.codes[issuer]
	if !ok {
		return false, fmt.Errorf("not exists")
	}

	if config.verified {
		return false, fmt.Errorf("already verified")
	}

	if config.timeExp < 0 {
		return false, fmt.Errorf("otp expired")
	}

	if config.code != enteredCode {
		return false, fmt.Errorf("wrong otp")
	}

	return true, nil
}

func (srv *OTPService) regenOTP(is, secret string) error {
	now := time.Now()

	code, err := core.GenerateOTP(now, secret)
	if err != nil {
		return err
	}

	srv.mu.Lock()
	defer srv.mu.Unlock()

	srv.codes[is] = OTPConfig{
		code:     code,
		timeExp:  now.Second(),
		verified: false,
	}

	return nil
}

func (srv *OTPService) Clean() {
	ticker := time.NewTicker(30)
	defer ticker.Stop()

	for range ticker.C {
		srv.cleanExpired()
	}
}

func (srv *OTPService) cleanExpired() {
	srv.mu.Lock()
	defer srv.mu.Unlock()

	for is, config := range srv.codes {
		if config.verified || config.timeExp == 0 {
			delete(srv.codes, is)
		}
	}
}

func (srv *OTPService) GetHealth() []OTPConfig {
	cnfs := make([]OTPConfig, 0, len(srv.codes))
	for _, cnf := range srv.codes {
		cnfs = append(cnfs, cnf)
	}

	return cnfs
}
