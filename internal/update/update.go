package update

import (
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

func fetchVersions() ([]string, error) {
	url := ("https://api.github.com/repos/Vladroon22/Desktop-2FA-app/releases")

	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var releases []releases
	if err := json.NewDecoder(resp.Body).Decode(&releases); err != nil {
		return nil, err
	}

	var versions []string
	for _, release := range releases {
		version := strings.TrimPrefix(release.TagName, "v")
		versions = append(versions, version)
	}

	return versions, nil
}

func fetch(filename, vers string) error {
	var apiURL = fmt.Sprintf("https://github.com/Vladroon22/Desktop-2FA-app/releases/download/v%s/2fa", vers)

	client := &http.Client{
		Timeout: time.Second * 30,
	}

	req, err := http.NewRequest("GET", apiURL, nil)
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
		return fmt.Errorf("chmod +x wasnt't executed: %v", err)
	}

	return nil
}

func Fetch(currVers string) error {
	currExe, err := os.Executable()
	if err != nil {
		return fmt.Errorf("access to current executable wasn't got: %v", err)
	}

	if strings.Contains(currExe, "(deteled)") {
		return fmt.Errorf("executable is deleted")
	}

	oldFile, err := os.Stat(currExe)
	if err != nil {
		return fmt.Errorf("%v", err)
	}

	versions, err := fetchVersions()
	if err != nil {
		return fmt.Errorf("%v", err)
	}

	vers := versions[len(versions)-1]
	if semver.Compare(currVers, vers) == 0 {
		return fmt.Errorf("You're up-to-date")
	}

	if err := fetch(oldFile.Name(), vers); err != nil {
		return fmt.Errorf("%v", err)
	}

	return nil
}
