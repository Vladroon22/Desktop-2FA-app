package update

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"runtime"
	"strings"
	"time"

	"golang.org/x/mod/semver"
)

type releases struct {
	TagName    string `json:"tag_name"`
	Name       string `json:"name"`
	Prerelease bool   `json:"prerelease"`
}

func fetchVersions(c context.Context) (string, error) {
	_, cancel := context.WithTimeout(c, time.Second*15)
	defer cancel()

	url := ("https://api.github.com/repos/Vladroon22/Desktop-2FA-app/releases")

	resp, err := http.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var releases []releases
	if err := json.NewDecoder(resp.Body).Decode(&releases); err != nil {
		return "", err
	}

	var versions []string
	for _, release := range releases {
		version := strings.TrimPrefix(release.TagName, "v")
		versions = append(versions, version)
	}

	return versions[0], nil
}

func fetch(c context.Context, filename, vers string) error {
	ctx, cancel := context.WithTimeout(c, time.Second*15)
	defer cancel()

	var apiURL = fmt.Sprintf("https://github.com/Vladroon22/Desktop-2FA-app/releases/download/v%s/2fa", vers)

	client := &http.Client{
		Timeout: time.Second * 15,
	}

	req, err := http.NewRequestWithContext(ctx, "GET", apiURL, nil)
	if err != nil {
		return fmt.Errorf("%v", err)
	}

	req.Header.Set("User-Agent", "Checker-for-Updates")

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("%v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("resp status code: %s", resp.Status)
	}

	OS := runtime.GOOS
	switch OS {
	case "windows":
		if err := applyForWin(filename, resp.Body); err != nil {
			return err
		}
	case "linux":
		if err := applyForUnix(filename, resp.Body); err != nil {
			return err
		}
	default:
		return fmt.Errorf("it isn't implemented for your %s", OS)
	}

	return nil
}

func applyForWin(filename string, rb io.ReadCloser) error {
	newFile, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("create file: %v", err.Error())
	}
	defer newFile.Close()

	if _, err := io.Copy(newFile, rb); err != nil {
		return fmt.Errorf("%v", err.Error())
	}

	if err := newFile.Chmod(0755); err != nil {
		return fmt.Errorf("chmod warning (Windows): %v", err)
	}

	return nil
}

func applyForUnix(filename string, rb io.ReadCloser) error {
	newFile, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("create file: %v", err.Error())
	}
	defer newFile.Close()

	if _, err := io.Copy(newFile, rb); err != nil {
		return fmt.Errorf("%v", err.Error())
	}

	if err := newFile.Chmod(0755); err != nil {
		return fmt.Errorf("chmod +x wasn't executed: %v", err)
	}

	return nil
}

func delete() error {
	currExe, err := os.Executable()
	if err != nil {
		return fmt.Errorf("access to current executable wasn't got: %v", err)
	}

	if strings.Contains(currExe, "(deteled)") {
		return nil
	}

	oldFile, err := os.Stat(currExe)
	if err != nil {
		return fmt.Errorf("%v", err)
	}

	return os.Remove(oldFile.Name())
}

func Fetch(ctx context.Context, currVers string) error {
	var (
		err    error
		latest string
	)

	defer func(err error) {
		if err == nil {
			delete()
		}
	}(err)

	latest, err = fetchVersions(ctx)
	if err != nil {
		err = fmt.Errorf("%v", err)
		return err
	}

	if semver.Compare("v"+currVers, "v"+latest) == 0 {
		err = fmt.Errorf("You're up-to-date")
		return err
	}

	newName := fmt.Sprintf("2fa-v%s", latest)
	err = fetch(ctx, newName, latest)
	if err != nil {
		err = fmt.Errorf("%v", err)
		return err
	}

	return nil
}
