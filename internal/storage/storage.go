package storage

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"sync"

	"github.com/zalando/go-keyring"
)

type Config struct {
	ServiceName string            `json:"service_name"`
	Usernames   map[string]string `json:"usernames"`
}

type KeyManager struct {
	serviceName string
	config      *Config
	configFile  string
	mu          sync.Mutex
}

func NewKeyManager(serviceName string) (*KeyManager, error) {
	configFile := "conf.json"

	km := &KeyManager{
		serviceName: serviceName,
		configFile:  configFile,
		config: &Config{
			ServiceName: serviceName,
			Usernames:   make(map[string]string),
		},
	}

	if err := km.loadConfig(); err != nil {
		log.Printf("Warning: could not load config: %v", err)
	}

	return km, nil
}

func (km *KeyManager) withWriteAccess(operation func() error) error {
	km.mu.Lock()
	defer km.mu.Unlock()

	var originalMode os.FileMode
	if info, err := os.Stat(km.configFile); err == nil {
		originalMode = info.Mode()
	}

	if err := os.Chmod(km.configFile, 0600); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to set write perms: %w", err)
	}

	defer func() {
		if originalMode != 0 {
			os.Chmod(km.configFile, originalMode)
		}
	}()

	return operation()
}

func (km *KeyManager) withReadAccess(operation func() error) error {
	km.mu.Lock()
	defer km.mu.Unlock()

	info, err := os.Stat(km.configFile)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("failed to stat file: %w", err)
	}

	originalMode := info.Mode()

	if err := os.Chmod(km.configFile, 0400); err != nil {
		return fmt.Errorf("failed to set read perms: %w", err)
	}
	defer os.Chmod(km.configFile, originalMode)

	return operation()
}

func (km *KeyManager) SaveAPIKey(keyName string, apiKey string) error {
	username := fmt.Sprintf("key_%s", keyName)

	km.config.Usernames[keyName] = username

	if err := keyring.Set(km.serviceName, username, apiKey); err != nil {
		return fmt.Errorf("failed to save API key: %w", err)
	}

	return km.saveConfig()
}

func (km *KeyManager) GetAPIKey(keyName string) (string, error) {
	username, exists := km.config.Usernames[keyName]
	if !exists {
		return "", fmt.Errorf("API key %s not found", keyName)
	}

	secret, err := keyring.Get(km.serviceName, username)
	if err != nil {
		return "", fmt.Errorf("failed to get API key: %w", err)
	}

	return secret, nil
}

func (km *KeyManager) saveConfig() error {
	return km.withWriteAccess(func() error {
		data, err := json.MarshalIndent(km.config, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal config: %w", err)
		}
		defer km.ManageFile()

		if _, err := os.Stat(km.configFile); os.IsNotExist(err) {
			if err := os.WriteFile(km.configFile, data, 0600); err != nil {
				return fmt.Errorf("failed to write file: %w", err)
			}
			return os.Chmod(km.configFile, 0000)
		}

		return os.WriteFile(km.configFile, data, 0600)
	})
}

func (km *KeyManager) loadConfig() error {
	return km.withReadAccess(func() error {
		data, err := os.ReadFile(km.configFile)
		if err != nil {
			if os.IsNotExist(err) {
				return nil
			}
			return fmt.Errorf("failed to read file: %w", err)
		}
		defer km.ManageFile()

		if err := json.Unmarshal(data, &km.config); err != nil {
			return fmt.Errorf("failed to unmarshal config: %w", err)
		}

		return nil
	})
}

func (km *KeyManager) Delete(keyName string) error {
	username, exists := km.config.Usernames[keyName]
	if !exists {
		return fmt.Errorf("key %s not found", keyName)
	}

	if err := keyring.Delete(km.serviceName, username); err != nil {
		return fmt.Errorf("failed to delete from keyring: %w", err)
	}

	delete(km.config.Usernames, keyName)

	return km.saveConfig()
}

func (km *KeyManager) ManageFile() error {
	return os.Chmod(km.configFile, 0000)
}

type Data struct {
	secret string
	name   string
}

func (km *KeyManager) List() map[string]string {
	if err := km.loadConfig(); err != nil {
		log.Printf("Warning: could not load config: %v", err)
		return nil
	}

	m := make(map[string]string, len(km.config.Usernames))
	for k := range km.config.Usernames {
		s, err := km.GetAPIKey(k)
		if err != nil {
			log.Println(err)
			continue
		}

		m[k] = s
	}

	return m
}
