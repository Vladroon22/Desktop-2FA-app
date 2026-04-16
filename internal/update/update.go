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

var (
	OS string
)

const (
	UpToDate   = "You're up-to-date"
	OverToDate = "You're is too up-to-dated"
)

type releases struct {
	TagName    string `json:"tag_name"`
	Name       string `json:"name"`
	Prerelease bool   `json:"prerelease"`
}

func fetchVersions(c context.Context) (string, error) {
	_, cancel := context.WithTimeout(c, time.Second*30)
	defer cancel()

	url := "https://api.github.com/repos/Vladroon22/Desktop-2FA-app/releases"

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
	ctx, cancel := context.WithTimeout(c, time.Second*20)
	defer cancel()

	var apiURL string

	switch OS {
	case "linux":
		apiURL = fmt.Sprintf("https://github.com/Vladroon22/Desktop-2FA-app/releases/download/v%s/2fa-%s", vers, OS)
	case "windows":
		apiURL = fmt.Sprintf("https://github.com/Vladroon22/Desktop-2FA-app/releases/download/v%s/2fa-%s.exe", vers, OS)
	}

	client := &http.Client{
		Timeout: time.Second * 20,
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

	switch OS {
	case "windows", "linux":
		if err := applyForOS(filename, resp.Body); err != nil {
			return err
		}
	default:
		return fmt.Errorf("it isn't implemented for your %s", OS)
	}

	return nil
}

func applyForOS(filename string, rb io.ReadCloser) error {
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
	if OS == "windows" {
		return nil
	}

	currExe, err := os.Executable()
	if err != nil {
		return fmt.Errorf("access to current executable wasn't got: %v", err)
	}

	if strings.Contains(currExe, "(deteled)") {
		return nil
	}

	return os.Remove(currExe)
}

func Fetch(ctx context.Context, currVers string) error {
	OS = runtime.GOOS

	var (
		err    error
		latest string
	)

	latest, err = fetchVersions(ctx)
	if err != nil {
		return fmt.Errorf("%v", err)
	}

	res := semver.Compare("v"+currVers, "v"+latest)
	switch res {
	case 0:
		return fmt.Errorf("%v", UpToDate)
	case 1:
		return fmt.Errorf("%v", OverToDate)
	}

	var newName string
	switch OS {
	case "linux":
		newName = fmt.Sprintf("2fa-v%s", latest)
	case "windows":
		newName = fmt.Sprintf("2fa-v%s.exe", latest)
	default:
		return fmt.Errorf("it isn't implemented for your %s", OS)
	}

	err = fetch(ctx, newName, latest)
	if err != nil {
		return fmt.Errorf("%v", err)
	}

	if err := delete(); err != nil && (err.Error() != UpToDate || err.Error() != OverToDate) {
		return fmt.Errorf("delete error: %v", err)
	}

	return nil
}
