package cliapp

import (
	"math/rand"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

const (
	dissolveFrames   = 6
	dissolveInterval = 80 * time.Millisecond
)

// decayGlyphs are used to replace characters during the dissolve animation,
// ordered from heavy to light so the text appears to erode.
var decayGlyphs = []rune{'░', '▒', '▓', '·', '.', ' '}

type dissolveTickMsg struct{}

func dissolveTickCmd() tea.Cmd {
	return tea.Tick(dissolveInterval, func(_ time.Time) tea.Msg {
		return dissolveTickMsg{}
	})
}

// dissolveText returns a version of original where a fraction of characters
// (proportional to frame/totalFrames) have been replaced with decay glyphs.
// The rng parameter controls which positions are replaced, making the output
// deterministic for a given seed.
func dissolveText(original string, frame, totalFrames int, rng *rand.Rand) string {
	if frame <= 0 {
		return original
	}
	runes := []rune(original)
	n := len(runes)
	if n == 0 {
		return original
	}

	// Last frame: everything is a space.
	if frame >= totalFrames {
		out := make([]rune, n)
		for i := range out {
			out[i] = ' '
		}
		return string(out)
	}

	// Build a shuffled order of positions so each character gets a
	// deterministic "decay time".
	order := rng.Perm(n)

	// How many characters have started decaying by this frame.
	decayed := n * frame / totalFrames

	out := make([]rune, n)
	copy(out, runes)

	for i := 0; i < decayed; i++ {
		pos := order[i]
		if runes[pos] == ' ' {
			continue // keep existing spaces
		}
		// Pick a decay glyph based on how long ago this char started decaying.
		age := frame - (i * totalFrames / n)
		gi := age - 1
		if gi < 0 {
			gi = 0
		}
		if gi >= len(decayGlyphs) {
			gi = len(decayGlyphs) - 1
		}
		out[pos] = decayGlyphs[gi]
	}

	return string(out)
}
