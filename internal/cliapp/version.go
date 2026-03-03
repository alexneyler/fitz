package cliapp

import (
	"fmt"
	"strconv"
	"strings"
)

// parseVersion extracts major, minor, patch and an optional prerelease
// identifier from a version string such as "v1.2.3" or "v1.2.3-preview.4".
func parseVersion(s string) (major, minor, patch int, prerelease string, err error) {
	s = strings.TrimPrefix(s, "v")

	if idx := strings.IndexByte(s, '-'); idx >= 0 {
		prerelease = s[idx+1:]
		s = s[:idx]
	}

	parts := strings.SplitN(s, ".", 3)
	if len(parts) != 3 {
		return 0, 0, 0, "", fmt.Errorf("invalid version: %q", s)
	}

	major, err = strconv.Atoi(parts[0])
	if err != nil {
		return 0, 0, 0, "", fmt.Errorf("invalid major version: %q", parts[0])
	}
	minor, err = strconv.Atoi(parts[1])
	if err != nil {
		return 0, 0, 0, "", fmt.Errorf("invalid minor version: %q", parts[1])
	}
	patch, err = strconv.Atoi(parts[2])
	if err != nil {
		return 0, 0, 0, "", fmt.Errorf("invalid patch version: %q", parts[2])
	}

	return major, minor, patch, prerelease, nil
}

// compareVersions returns -1, 0, or 1 comparing two version strings.
// A release version is greater than a prerelease of the same base (per semver).
// The special value "dev" is treated as older than any release.
func compareVersions(a, b string) int {
	if a == "dev" && b == "dev" {
		return 0
	}
	if a == "dev" {
		return -1
	}
	if b == "dev" {
		return 1
	}

	aMaj, aMin, aPat, aPre, aErr := parseVersion(a)
	bMaj, bMin, bPat, bPre, bErr := parseVersion(b)

	if aErr != nil && bErr != nil {
		return 0
	}
	if aErr != nil {
		return -1
	}
	if bErr != nil {
		return 1
	}

	if aMaj != bMaj {
		return cmpInt(aMaj, bMaj)
	}
	if aMin != bMin {
		return cmpInt(aMin, bMin)
	}
	if aPat != bPat {
		return cmpInt(aPat, bPat)
	}

	// Same base: release > prerelease.
	if aPre == "" && bPre == "" {
		return 0
	}
	if aPre == "" {
		return 1
	}
	if bPre == "" {
		return -1
	}

	return comparePrerelease(aPre, bPre)
}

func cmpInt(a, b int) int {
	if a < b {
		return -1
	}
	return 1
}

func comparePrerelease(a, b string) int {
	ap := strings.Split(a, ".")
	bp := strings.Split(b, ".")

	for i := 0; i < len(ap) && i < len(bp); i++ {
		aNum, aErr := strconv.Atoi(ap[i])
		bNum, bErr := strconv.Atoi(bp[i])

		if aErr == nil && bErr == nil {
			if aNum != bNum {
				return cmpInt(aNum, bNum)
			}
		} else {
			if ap[i] < bp[i] {
				return -1
			}
			if ap[i] > bp[i] {
				return 1
			}
		}
	}

	return cmpInt(len(ap), len(bp))
}
