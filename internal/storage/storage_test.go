package storage

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zalando/go-keyring"
)

func init() {
	keyring.MockInit()
}

func TestNewKeyManager(t *testing.T) {
	tmpDir := t.TempDir()
	originalDir, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(originalDir)

	tests := []struct {
		name          string
		serviceName   string
		setupFunc     func()
		wantErr       bool
		wantEmptyConf bool
	}{
		{
			name:        "successfull creating manager",
			serviceName: "test-service",
			wantErr:     false,
		},
		{
			name:        "blank name",
			serviceName: "",
			wantErr:     false,
		},
		{
			name:        "success load of existing file",
			serviceName: "test-service",
			setupFunc: func() {
				config := Config{
					ServiceName: "test-service",
					Usernames: map[string]string{
						"test-key": "key_test-key",
					},
				}
				data, _ := json.Marshal(config)
				os.WriteFile("conf.json", data, 0644)
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setupFunc != nil {
				tt.setupFunc()
			}

			km, err := NewKeyManager(tt.serviceName)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, km)
				assert.Equal(t, tt.serviceName, km.serviceName)
				assert.NotNil(t, km.config)
			}
		})
	}
}

func TestKeyManager_SaveAPIKey(t *testing.T) {
	tmpDir := t.TempDir()
	originalDir, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(originalDir)

	km, err := NewKeyManager("test-service")
	require.NoError(t, err)

	tests := []struct {
		name      string
		keyName   string
		apiKey    string
		setupFunc func()
		wantErr   bool
	}{
		{
			name:    "successfull creating of API key",
			keyName: "test-key",
			apiKey:  "api-secret-123",
			wantErr: false,
		},
		{
			name:    "Saving with duplicated key",
			keyName: "test-key",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setupFunc != nil {
				tt.setupFunc()
			}

			err := km.SaveAPIKey(tt.keyName, tt.apiKey)

			if tt.wantErr {
				assert.Error(t, err)
			}

		})
	}
}

func TestKeyManager_GetAPIKey(t *testing.T) {
	tmpDir := t.TempDir()
	originalDir, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(originalDir)

	km, err := NewKeyManager("test-service")
	require.NoError(t, err)

	err = km.SaveAPIKey("existing-key", "test-secret-123")
	require.NoError(t, err)

	tests := []struct {
		name      string
		keyName   string
		wantKey   string
		wantErr   bool
		errString string
	}{
		{
			name:    "successfull getting of key",
			keyName: "existing-key",
			wantKey: "test-secret-123",
			wantErr: false,
		},
		{
			name:      "getting of non-existing key",
			keyName:   "unknown-key",
			wantErr:   true,
			errString: "not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			key, err := km.getAPIKey(tt.keyName)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errString != "" {
					assert.Contains(t, err.Error(), tt.errString)
				}
				assert.Empty(t, key)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.wantKey, key)
			}
		})
	}
}

func TestKeyManager_SaveConfig_Permissions(t *testing.T) {
	tmpDir := t.TempDir()
	originalDir, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(originalDir)

	km, err := NewKeyManager("test-service")
	require.NoError(t, err)

	t.Run("Save new config", func(t *testing.T) {
		os.Remove("conf.json")

		km.config.Usernames["test"] = "key_test"
		err := km.saveConfig()
		assert.NoError(t, err)

		_, err = os.Stat("conf.json")
		assert.NoError(t, err)
	})

	t.Run("Saving of existing config", func(t *testing.T) {
		testDir := t.TempDir()
		configFile := filepath.Join(testDir, "conf.json")

		km := &KeyManager{
			serviceName: "test-service",
			configFile:  configFile,
			config: &Config{
				ServiceName: "test-service",
				Usernames:   make(map[string]string),
			},
		}

		data := []byte(`{"service_name":"test-service","usernames":{}}`)
		err := os.WriteFile(configFile, data, 0644)
		require.NoError(t, err)

		km.config.Usernames["another"] = "key_another"
		err = km.saveConfig()
		assert.NoError(t, err)

		os.Chmod(configFile, 0644)

		savedData, err := os.ReadFile(configFile)
		assert.NoError(t, err)

		var config Config
		json.Unmarshal(savedData, &config)
		assert.Contains(t, config.Usernames, "another")
	})
}
