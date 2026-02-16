package cliapp

import (
	"math/rand"
	"strings"
	"testing"
)

func TestDissolveTextFrame0Unchanged(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	got := dissolveText("hello world", 0, dissolveFrames, rng)
	if got != "hello world" {
		t.Fatalf("frame 0: got %q, want %q", got, "hello world")
	}
}

func TestDissolveTextFinalFrameAllSpaces(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	got := dissolveText("hello world", dissolveFrames, dissolveFrames, rng)
	if strings.TrimSpace(got) != "" {
		t.Fatalf("final frame: got %q, want all spaces", got)
	}
	if len([]rune(got)) != len([]rune("hello world")) {
		t.Fatalf("final frame length %d != original length %d", len([]rune(got)), len([]rune("hello world")))
	}
}

func TestDissolveTextIntermediateFramesMixed(t *testing.T) {
	original := "hello world"
	rng := rand.New(rand.NewSource(42))
	for frame := 1; frame < dissolveFrames; frame++ {
		got := dissolveText(original, frame, dissolveFrames, rng)
		if got == original {
			t.Errorf("frame %d: text unchanged, expected some decay", frame)
		}
		if strings.TrimSpace(got) == "" {
			t.Errorf("frame %d: fully blank, expected some original chars", frame)
		}
	}
}

func TestDissolveTextDeterministic(t *testing.T) {
	original := "deterministic test"
	for frame := 0; frame <= dissolveFrames; frame++ {
		rng1 := rand.New(rand.NewSource(99))
		rng2 := rand.New(rand.NewSource(99))
		a := dissolveText(original, frame, dissolveFrames, rng1)
		b := dissolveText(original, frame, dissolveFrames, rng2)
		if a != b {
			t.Fatalf("frame %d: not deterministic: %q != %q", frame, a, b)
		}
	}
}

func TestDissolveTextEmptyString(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	got := dissolveText("", 3, dissolveFrames, rng)
	if got != "" {
		t.Fatalf("empty string: got %q, want %q", got, "")
	}
}

func TestDissolveTextPreservesLength(t *testing.T) {
	original := "some text here"
	for frame := 0; frame <= dissolveFrames; frame++ {
		rng := rand.New(rand.NewSource(7))
		got := dissolveText(original, frame, dissolveFrames, rng)
		if len([]rune(got)) != len([]rune(original)) {
			t.Fatalf("frame %d: length %d != %d", frame, len([]rune(got)), len([]rune(original)))
		}
	}
}

func TestDissolveTextBeyondTotalFrames(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	got := dissolveText("test", dissolveFrames+5, dissolveFrames, rng)
	if strings.TrimSpace(got) != "" {
		t.Fatalf("beyond total frames: got %q, want all spaces", got)
	}
}
