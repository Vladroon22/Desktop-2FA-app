package gpg

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// GPGWrapper представляет обертку над утилитой gpg
type GPGWrapper struct {
	binaryPath string
	homeDir    string
	armor      bool
}

// NewGPGWrapper создает новый экземпляр обертки
func NewGPGWrapper(homeDir string) *GPGWrapper {
	return &GPGWrapper{
		binaryPath: "gpg",
		homeDir:    homeDir,
		armor:      false,
	}
}

// SetArmor включает/выключает текстовый вывод (ASCII-armor)
func (g *GPGWrapper) SetArmor(useArmor bool) {
	g.armor = useArmor
}

// buildArgs формирует базовые аргументы команды
func (g *GPGWrapper) buildArgs(extraArgs ...string) []string {
	args := []string{}

	if g.homeDir != "" {
		args = append(args, "--homedir", g.homeDir)
	}

	if g.armor {
		args = append(args, "--armor")
	}

	args = append(args, extraArgs...)
	return args
}

// exec выполняет команду gpg и возвращает результат
func (g *GPGWrapper) exec(args []string, input []byte) ([]byte, error) {
	cmd := exec.Command(g.binaryPath, args...)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if input != nil {
		cmd.Stdin = bytes.NewReader(input)
	}

	err := cmd.Run()
	if err != nil {
		return nil, fmt.Errorf("gpg error: %v, stderr: %s", err, stderr.String())
	}

	return stdout.Bytes(), nil
}

// KeyInfo содержит информацию о ключе
type KeyInfo struct {
	KeyID       string
	UserID      string
	Fingerprint string
	Expires     string
	Trust       string
}

// ListKeys возвращает список открытых ключей
func (g *GPGWrapper) ListKeys() ([]KeyInfo, error) {
	args := g.buildArgs("--list-keys", "--with-colons")
	output, err := g.exec(args, nil)
	if err != nil {
		return nil, err
	}

	return g.parseKeyList(output)
}

// ListSecretKeys возвращает список секретных ключей
func (g *GPGWrapper) ListSecretKeys() ([]KeyInfo, error) {
	args := g.buildArgs("--list-secret-keys", "--with-colons")
	output, err := g.exec(args, nil)
	if err != nil {
		return nil, err
	}

	return g.parseKeyList(output)
}

// parseKeyList парсит вывод gpg --with-colons
func (g *GPGWrapper) parseKeyList(data []byte) ([]KeyInfo, error) {
	lines := strings.Split(string(data), "\n")
	var keys []KeyInfo
	var currentKey *KeyInfo

	for _, line := range lines {
		if line == "" {
			continue
		}

		fields := strings.Split(line, ":")
		if len(fields) < 5 {
			continue
		}

		switch fields[0] {
		case "pub", "sec":
			// Новый ключ
			if currentKey != nil {
				keys = append(keys, *currentKey)
			}
			currentKey = &KeyInfo{
				KeyID:   fields[4],
				Expires: fields[6],
				Trust:   fields[1],
			}
		case "uid":
			// Идентификатор пользователя
			if currentKey != nil {
				currentKey.UserID = fields[9]
			}
		case "fpr":
			// Отпечаток ключа
			if currentKey != nil {
				currentKey.Fingerprint = fields[9]
			}
		}
	}

	if currentKey != nil {
		keys = append(keys, *currentKey)
	}

	return keys, nil
}

// Encrypt шифрует данные для указанных получателей
func (g *GPGWrapper) Encrypt(data []byte, recipients []string, sign bool) ([]byte, error) {
	args := []string{"--encrypt"}

	for _, recipient := range recipients {
		args = append(args, "--recipient", recipient)
	}

	if sign {
		args = append(args, "--sign")
	}

	args = g.buildArgs(args...)
	return g.exec(args, data)
}

// Decrypt расшифровывает данные
func (g *GPGWrapper) Decrypt(data []byte) ([]byte, error) {
	args := g.buildArgs("--decrypt")
	return g.exec(args, data)
}

// Sign создает подпись для данных
func (g *GPGWrapper) Sign(data []byte, detach bool) ([]byte, error) {
	args := []string{}

	if detach {
		args = append(args, "--detach-sign")
	} else {
		args = append(args, "--sign")
	}

	args = g.buildArgs(args...)
	return g.exec(args, data)
}

// Verify проверяет подпись
func (g *GPGWrapper) Verify(signature, data []byte) (bool, error) {
	// Создаем временные файлы для подписи и данных
	sigFile, err := os.CreateTemp("", "gpg-sig-*")
	if err != nil {
		return false, err
	}
	defer os.Remove(sigFile.Name())
	defer sigFile.Close()

	dataFile, err := os.CreateTemp("", "gpg-data-*")
	if err != nil {
		return false, err
	}
	defer os.Remove(dataFile.Name())
	defer dataFile.Close()

	// Записываем данные
	if _, err := sigFile.Write(signature); err != nil {
		return false, err
	}
	if _, err := dataFile.Write(data); err != nil {
		return false, err
	}

	args := g.buildArgs("--verify", sigFile.Name(), dataFile.Name())
	_, err = g.exec(args, nil)
	return err == nil, err
}

// GenerateKey генерирует новую пару ключей
func (g *GPGWrapper) GenerateKey(name, email, passphrase string) error {
	// Создаем временный файл с параметрами для batch-режима
	batchFile, err := os.CreateTemp("", "gpg-batch-*")
	if err != nil {
		return err
	}
	defer os.Remove(batchFile.Name())
	defer batchFile.Close()

	// Формируем параметры для генерации ключа в batch-режиме
	batchContent := fmt.Sprintf(`Key-Type: RSA
Key-Length: 3072
Subkey-Type: RSA
Subkey-Length: 3072
Name-Real: %s
Name-Email: %s
Expire-Date: 0
Passphrase: %s
`, name, email, passphrase)

	if _, err := batchFile.WriteString(batchContent); err != nil {
		return err
	}

	args := g.buildArgs("--batch", "--generate-key", batchFile.Name())
	_, err = g.exec(args, nil)
	return err
}

// ImportKey импортирует ключ из данных
func (g *GPGWrapper) ImportKey(keyData []byte) error {
	args := g.buildArgs("--import")
	_, err := g.exec(args, keyData)
	return err
}

// ExportKey экспортирует открытый ключ
func (g *GPGWrapper) ExportKey(keyID string) ([]byte, error) {
	args := g.buildArgs("--export", keyID)
	return g.exec(args, nil)
}

// ExportSecretKey экспортирует секретный ключ
func (g *GPGWrapper) ExportSecretKey(keyID string) ([]byte, error) {
	args := g.buildArgs("--export-secret-keys", keyID)
	return g.exec(args, nil)
}

// DeleteKey удаляет ключ
func (g *GPGWrapper) DeleteKey(keyID string, secret bool) error {
	args := []string{}
	if secret {
		args = append(args, "--delete-secret-key")
	} else {
		args = append(args, "--delete-key")
	}
	args = append(args, keyID)

	args = g.buildArgs(args...)
	_, err := g.exec(args, nil)
	return err
}

// SignKey подписывает ключ
func (g *GPGWrapper) SignKey(keyID string, local bool) error {
	args := []string{}
	if local {
		args = append(args, "--lsign-key")
	} else {
		args = append(args, "--sign-key")
	}
	args = append(args, keyID)

	args = g.buildArgs(args...)
	_, err := g.exec(args, nil)
	return err
}

// ChangePassphrase меняет пароль на ключе
func (g *GPGWrapper) ChangePassphrase(keyID string) error {
	args := g.buildArgs("--change-passphrase", keyID)
	_, err := g.exec(args, nil)
	return err
}

// GetFingerprint получает отпечаток ключа
func (g *GPGWrapper) GetFingerprint(keyID string) (string, error) {
	args := g.buildArgs("--fingerprint", "--with-colons", keyID)
	output, err := g.exec(args, nil)
	if err != nil {
		return "", err
	}

	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		fields := strings.Split(line, ":")
		if fields[0] == "fpr" && len(fields) > 9 {
			return fields[9], nil
		}
	}

	return "", fmt.Errorf("fingerprint not found for key %s", keyID)
}
