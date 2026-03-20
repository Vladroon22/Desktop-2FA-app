package update

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"runtime"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func testFetch(ctx context.Context, v string) int {
	var apiURL string
	OS := runtime.GOOS

	switch OS {
	case "linux":
		apiURL = fmt.Sprintf("https://github.com/Vladroon22/Desktop-2FA-app/releases/download/v%s/2fa-%s", v, OS)
	case "windows":
		apiURL = fmt.Sprintf("https://github.com/Vladroon22/Desktop-2FA-app/releases/download/v%s/2fa-%s.exe", v, OS)
	}

	client := &http.Client{
		Timeout: time.Second * 15,
	}

	req, err := http.NewRequestWithContext(ctx, "GET", apiURL, nil)
	if err != nil {
		return req.Response.StatusCode
	}

	req.Header.Set("User-Agent", "Checker-for-Updates")

	resp, err := client.Do(req)
	if err != nil {
		return resp.StatusCode
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return resp.StatusCode
	}

	return 200
}

func TestFetch(t *testing.T) {
	tests := []struct {
		name      string
		mockFunc  func(ctx context.Context, version string) (int, error)
		version   string
		wantCode  int
		wantError bool
	}{
		{
			name: "internal server error at github",
			mockFunc: func(ctx context.Context, version string) (int, error) {
				return 500, errors.New("internal server error")
			},
			wantCode:  500,
			wantError: true,
		},
		{
			name: "context timeout",
			mockFunc: func(ctx context.Context, version string) (int, error) {
				select {
				case <-ctx.Done():
					return 0, ctx.Err()
				case <-time.After(100 * time.Millisecond):
					return 200, nil
				}
			},
			version:   "1.0.5",
			wantError: true,
		},
		{
			name: "not found 404",
			mockFunc: func(ctx context.Context, version string) (int, error) {
				code := testFetch(ctx, version)
				if code != 200 {
					return 404, errors.New("such version not exists")
				}
				return 0, nil
			},
			version:   "9.9.9",
			wantCode:  404,
			wantError: true,
		},
		{
			name: "OK",
			mockFunc: func(ctx context.Context, version string) (int, error) {
				code := testFetch(ctx, version)

				return code, nil
			},
			version:   "1.0.5",
			wantCode:  200,
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			if tt.name == "context timeout" {
				var cancel context.CancelFunc
				ctx, cancel = context.WithTimeout(ctx, 1*time.Nanosecond)
				defer cancel()
			}

			code, err := tt.mockFunc(ctx, tt.version)

			if tt.wantError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			assert.Equal(t, tt.wantCode, code)
		})
	}
}
