package cliapp

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
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

func TestLatestReleaseAssetNoMatch(t *testing.T) {
	prev := httpClient
	httpClient = &http.Client{
		Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(`{"assets":[{"name":"fitz_linux_amd64","browser_download_url":"https://example.invalid"}]}`)),
				Header:     make(http.Header),
			}, nil
		}),
	}
	t.Cleanup(func() { httpClient = prev })

	_, err := latestReleaseAsset(context.Background(), "fitz_darwin_arm64")
	if err == nil {
		t.Fatal("expected error")
	}
	if !errors.Is(err, errNoAsset) {
		t.Fatalf("error = %v", err)
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}
