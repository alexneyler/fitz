package cliapp

import "testing"

func TestParseVersion(t *testing.T) {
	tests := []struct {
		input                          string
		major, minor, patch            int
		prerelease                     string
		wantErr                        bool
	}{
		{input: "v0.7.0", major: 0, minor: 7, patch: 0},
		{input: "v1.2.3", major: 1, minor: 2, patch: 3},
		{input: "v0.7.1-preview.3", major: 0, minor: 7, patch: 1, prerelease: "preview.3"},
		{input: "0.7.0", major: 0, minor: 7, patch: 0},
		{input: "dev", wantErr: true},
		{input: "bad", wantErr: true},
		{input: "", wantErr: true},
	}

	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			maj, min, pat, pre, err := parseVersion(tc.input)
			if tc.wantErr {
				if err == nil {
					t.Fatal("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if maj != tc.major || min != tc.minor || pat != tc.patch {
				t.Fatalf("got %d.%d.%d, want %d.%d.%d", maj, min, pat, tc.major, tc.minor, tc.patch)
			}
			if pre != tc.prerelease {
				t.Fatalf("prerelease = %q, want %q", pre, tc.prerelease)
			}
		})
	}
}

func TestCompareVersions(t *testing.T) {
	tests := []struct {
		a, b string
		want int
	}{
		// Equal
		{"v0.7.0", "v0.7.0", 0},
		{"dev", "dev", 0},

		// Major/minor/patch ordering
		{"v1.0.0", "v0.9.9", 1},
		{"v0.7.0", "v0.8.0", -1},
		{"v0.7.1", "v0.7.0", 1},

		// Release beats prerelease of same base
		{"v0.7.1", "v0.7.1-preview.3", 1},
		{"v0.7.1-preview.3", "v0.7.1", -1},

		// Prerelease with higher base > lower base release
		{"v0.7.1-preview.3", "v0.7.0", 1},

		// Prerelease ordering
		{"v0.7.1-preview.5", "v0.7.1-preview.3", 1},
		{"v0.7.1-preview.1", "v0.7.1-preview.10", -1},

		// Dev is always oldest
		{"dev", "v0.0.1", -1},
		{"v0.0.1", "dev", 1},

		// Unparseable treated like dev
		{"garbage", "v0.1.0", -1},
		{"v0.1.0", "garbage", 1},
	}

	for _, tc := range tests {
		t.Run(tc.a+"_vs_"+tc.b, func(t *testing.T) {
			got := compareVersions(tc.a, tc.b)
			if got != tc.want {
				t.Fatalf("compareVersions(%q, %q) = %d, want %d", tc.a, tc.b, got, tc.want)
			}
		})
	}
}
