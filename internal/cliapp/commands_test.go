package cliapp

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"runtime"
	"strings"
	"testing"
)

func TestCompletionScripts(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want string
	}{
		{name: "bash", args: []string{"bash"}, want: bashCompletionScript},
		{name: "zsh", args: []string{"zsh"}, want: zshCompletionScript},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var out bytes.Buffer
			if err := Completion(context.Background(), &out, tc.args); err != nil {
				t.Fatalf("Completion returned error: %v", err)
			}
			if out.String() != tc.want {
				t.Fatalf("script mismatch:\n got %q\nwant %q", out.String(), tc.want)
			}
		})
	}
}

func TestCompletionUsageError(t *testing.T) {
	tests := [][]string{
		nil,
		{},
		{"fish"},
		{"bash", "extra"},
	}

	for _, args := range tests {
		var out bytes.Buffer
		err := Completion(context.Background(), &out, args)
		if err == nil {
			t.Fatalf("expected error for args %v", args)
		}
		if err.Error() != "usage: fitz completion <bash|zsh>" {
			t.Fatalf("error = %q", err.Error())
		}
		if out.Len() != 0 {
			t.Fatalf("stdout = %q, want empty", out.String())
		}
	}
}

func TestCompletionScriptsHandleBrGoByChangingDirectory(t *testing.T) {
	tests := [][]string{{"bash"}, {"zsh"}}

	for _, args := range tests {
		var out bytes.Buffer
		if err := Completion(context.Background(), &out, args); err != nil {
			t.Fatalf("Completion returned error: %v", err)
		}
		script := out.String()
		if !strings.Contains(script, `"$2" == "go"`) {
			t.Fatalf("script = %q, want br go handling", script)
		}
		if !strings.Contains(script, `dir="$(command fitz br cd "$3")" && cd "$dir"`) {
			t.Fatalf("script = %q, want cd via fitz br cd", script)
		}
	}
}

func TestCompletionScriptsIncludeReviewCommand(t *testing.T) {
	tests := [][]string{{"bash"}, {"zsh"}}

	for _, args := range tests {
		var out bytes.Buffer
		if err := Completion(context.Background(), &out, args); err != nil {
			t.Fatalf("Completion returned error: %v", err)
		}
		if !strings.Contains(out.String(), "review") {
			t.Fatalf("script = %q, want review command completion", out.String())
		}
	}
}

func TestSelectAsset(t *testing.T) {
	assets := []githubAsset{
		{Name: "fitz_linux_amd64", DownloadURL: "https://example.invalid/linux"},
		{Name: "fitz_darwin_arm64", DownloadURL: "https://example.invalid/darwin"},
	}

	got, ok := selectAsset(assets, "fitz_darwin_arm64")
	if !ok {
		t.Fatal("expected matching asset")
	}
	if got.Name != "fitz_darwin_arm64" {
		t.Fatalf("got %q", got.Name)
	}

	if _, ok := selectAsset(assets, "fitz_windows_amd64.exe"); ok {
		t.Fatal("expected no match")
	}
}

func TestAssetName(t *testing.T) {
	if got := assetName("darwin", "arm64"); got != "fitz_darwin_arm64" {
		t.Fatalf("got %q", got)
	}
	if got := assetName("windows", "amd64"); got != "fitz_windows_amd64.exe" {
		t.Fatalf("got %q", got)
	}
}

func TestLatestStableReleaseNoMatch(t *testing.T) {
	prev := httpClient
	httpClient = &http.Client{
		Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(`{"tag_name":"v0.7.0","assets":[{"name":"fitz_linux_amd64","browser_download_url":"https://example.invalid"}]}`)),
				Header:     make(http.Header),
			}, nil
		}),
	}
	t.Cleanup(func() { httpClient = prev })

	release, err := latestStableRelease(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if release.TagName != "v0.7.0" {
		t.Fatalf("tag = %q", release.TagName)
	}
	if _, ok := selectAsset(release.Assets, "fitz_darwin_arm64"); ok {
		t.Fatal("expected no match for darwin asset")
	}
}

func TestLatestReleasePreview(t *testing.T) {
	prev := httpClient
	httpClient = &http.Client{
		Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(`[{"tag_name":"v0.7.1-preview.3","assets":[{"name":"fitz_darwin_arm64","browser_download_url":"https://example.invalid/preview"}]}]`)),
				Header:     make(http.Header),
			}, nil
		}),
	}
	t.Cleanup(func() { httpClient = prev })

	release, err := latestRelease(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if release.TagName != "v0.7.1-preview.3" {
		t.Fatalf("tag = %q", release.TagName)
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func TestUpdateOutputsVersion(t *testing.T) {
	prevHTTP := httpClient
	prevExe := executablePath
	t.Cleanup(func() {
		httpClient = prevHTTP
		executablePath = prevExe
	})

	tmp, err := os.CreateTemp(t.TempDir(), "fitz-test-*")
	if err != nil {
		t.Fatal(err)
	}
	tmp.Close()
	executablePath = func() (string, error) { return tmp.Name(), nil }

	releaseJSON := `{"tag_name":"v1.2.3","assets":[{"name":"%s","browser_download_url":"https://example.invalid/download"}]}`
	releaseJSON = fmt.Sprintf(releaseJSON, assetName(runtime.GOOS, runtime.GOARCH))
	callCount := 0
	httpClient = &http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			callCount++
			if strings.Contains(req.URL.Path, "/releases/latest") {
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(releaseJSON)),
					Header:     make(http.Header),
				}, nil
			}
			// download request
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader("fake-binary")),
				Header:     make(http.Header),
			}, nil
		}),
	}

	var out bytes.Buffer
	err = Update(context.Background(), &out, "v1.0.0", false)
	if err != nil {
		t.Fatalf("Update returned error: %v", err)
	}
	if !strings.Contains(out.String(), "v1.2.3") {
		t.Fatalf("stdout = %q, want version v1.2.3", out.String())
	}
}

func TestUpdateAlreadyUpToDate(t *testing.T) {
	prevHTTP := httpClient
	t.Cleanup(func() { httpClient = prevHTTP })

	httpClient = &http.Client{
		Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(`{"tag_name":"v1.0.0","assets":[]}`)),
				Header:     make(http.Header),
			}, nil
		}),
	}

	var out bytes.Buffer
	err := Update(context.Background(), &out, "v1.0.0", false)
	if err != nil {
		t.Fatalf("Update returned error: %v", err)
	}
	if !strings.Contains(out.String(), "already up to date") {
		t.Fatalf("stdout = %q, want 'already up to date'", out.String())
	}
	if !strings.Contains(out.String(), "v1.0.0") {
		t.Fatalf("stdout = %q, want version v1.0.0", out.String())
	}
}
