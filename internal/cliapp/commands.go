package cliapp

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

func Version(_ context.Context, w io.Writer, version string) error {
	_, err := fmt.Fprintf(w, "fitz %s\n", version)
	return err
}

func Update(ctx context.Context, w io.Writer) error {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	targetAsset := assetName(runtime.GOOS, runtime.GOARCH)
	asset, err := latestReleaseAsset(ctx, targetAsset)
	if err != nil {
		return err
	}

	exePath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("%w: resolve executable path: %v", errReplaceBinary, err)
	}

	if err := downloadAndReplace(ctx, asset.DownloadURL, exePath); err != nil {
		return err
	}

	_, err = fmt.Fprintf(w, "updated from %s\n", asset.Name)
	return err
}

const (
	githubLatestReleaseAPI = "https://api.github.com/repos/alexneyler/fitz/releases/latest"
	// Release assets must be named as: fitz_<goos>_<goarch> (or .exe suffix on windows).
	releaseAssetPrefix = "fitz_"
)

var (
	httpClient       = &http.Client{Timeout: 20 * time.Second}
	errReleaseFetch  = errors.New("network/API failure")
	errNoAsset       = errors.New("no matching release asset")
	errReplaceBinary = errors.New("permission/replace failure")
)

type githubRelease struct {
	Assets []githubAsset `json:"assets"`
}

type githubAsset struct {
	Name        string `json:"name"`
	DownloadURL string `json:"browser_download_url"`
}

func assetName(goos, goarch string) string {
	name := releaseAssetPrefix + goos + "_" + goarch
	if goos == "windows" {
		return name + ".exe"
	}
	return name
}

func latestReleaseAsset(ctx context.Context, targetName string) (githubAsset, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, githubLatestReleaseAPI, nil)
	if err != nil {
		return githubAsset{}, fmt.Errorf("%w: build request: %v", errReleaseFetch, err)
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "fitz-updater")

	res, err := httpClient.Do(req)
	if err != nil {
		return githubAsset{}, fmt.Errorf("%w: request latest release: %v", errReleaseFetch, err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return githubAsset{}, fmt.Errorf("%w: latest release status %d", errReleaseFetch, res.StatusCode)
	}

	var release githubRelease
	if err := json.NewDecoder(res.Body).Decode(&release); err != nil {
		return githubAsset{}, fmt.Errorf("%w: decode release payload: %v", errReleaseFetch, err)
	}

	asset, ok := selectAsset(release.Assets, targetName)
	if !ok {
		return githubAsset{}, fmt.Errorf("%w: expected asset %q", errNoAsset, targetName)
	}

	return asset, nil
}

func selectAsset(assets []githubAsset, targetName string) (githubAsset, bool) {
	for _, asset := range assets {
		if strings.TrimSpace(asset.Name) == targetName && strings.TrimSpace(asset.DownloadURL) != "" {
			return asset, true
		}
	}
	return githubAsset{}, false
}

func downloadAndReplace(ctx context.Context, downloadURL, exePath string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, downloadURL, nil)
	if err != nil {
		return fmt.Errorf("%w: build download request: %v", errReleaseFetch, err)
	}
	req.Header.Set("User-Agent", "fitz-updater")

	res, err := httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("%w: download release asset: %v", errReleaseFetch, err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("%w: download status %d", errReleaseFetch, res.StatusCode)
	}

	dir := filepath.Dir(exePath)
	tmp, err := os.CreateTemp(dir, "fitz-update-*")
	if err != nil {
		return fmt.Errorf("%w: create temporary binary: %v", errReplaceBinary, err)
	}
	tmpPath := tmp.Name()
	defer func() {
		_ = tmp.Close()
		_ = os.Remove(tmpPath)
	}()

	mode := os.FileMode(0o755)
	if info, statErr := os.Stat(exePath); statErr == nil {
		mode = info.Mode().Perm()
	}
	if err := tmp.Chmod(mode); err != nil {
		return fmt.Errorf("%w: set executable mode: %v", errReplaceBinary, err)
	}

	if _, err := io.Copy(tmp, res.Body); err != nil {
		return fmt.Errorf("%w: write temporary binary: %v", errReplaceBinary, err)
	}
	if err := tmp.Sync(); err != nil {
		return fmt.Errorf("%w: sync temporary binary: %v", errReplaceBinary, err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("%w: close temporary binary: %v", errReplaceBinary, err)
	}

	if err := os.Rename(tmpPath, exePath); err != nil {
		return fmt.Errorf("%w: replace executable: %v", errReplaceBinary, err)
	}

	return nil
}

func Completion(_ context.Context, w io.Writer, args []string) error {
	if len(args) != 1 {
		return errors.New("usage: fitz completion <bash|zsh>")
	}

	var script string
	switch args[0] {
	case "bash":
		script = bashCompletionScript
	case "zsh":
		script = zshCompletionScript
	default:
		return errors.New("usage: fitz completion <bash|zsh>")
	}

	_, err := io.WriteString(w, script)
	return err
}

const bashCompletionScript = `fitz() {
  if [[ "$1" == "br" && "$2" == "cd" && -n "$3" ]]; then
    local dir
    dir="$(command fitz br cd "$3")" && cd "$dir"
  else
    command fitz "$@"
  fi
}

_fitz_completion() {
  local cur prev
  cur="${COMP_WORDS[COMP_CWORD]}"
  prev="${COMP_WORDS[COMP_CWORD-1]}"

  if [[ ${COMP_CWORD} -eq 1 ]]; then
    COMPREPLY=( $(compgen -W "help version update completion br" -- "$cur") )
    return
  fi

  if [[ ${COMP_CWORD} -eq 2 && "$prev" == "completion" ]]; then
    COMPREPLY=( $(compgen -W "bash zsh" -- "$cur") )
    return
  fi

  if [[ ${COMP_CWORD} -eq 2 && "$prev" == "br" ]]; then
    COMPREPLY=( $(compgen -W "new go rm list cd help" -- "$cur") )
    return
  fi
}

complete -F _fitz_completion fitz
`

const zshCompletionScript = `#compdef fitz

fitz() {
  if [[ "$1" == "br" && "$2" == "cd" && -n "$3" ]]; then
    local dir
    dir="$(command fitz br cd "$3")" && cd "$dir"
  else
    command fitz "$@"
  fi
}

_fitz() {
  local -a commands shells br_cmds
  commands=(help version update completion br)
  shells=(bash zsh)
  br_cmds=(new go rm list cd help)

  if (( CURRENT == 2 )); then
    compadd -- $commands
    return
  fi

  if (( CURRENT == 3 )) && [[ "${words[2]}" == "completion" ]]; then
    compadd -- $shells
    return
  fi

  if (( CURRENT == 3 )) && [[ "${words[2]}" == "br" ]]; then
    compadd -- $br_cmds
    return
  fi
}

compdef _fitz fitz
`
