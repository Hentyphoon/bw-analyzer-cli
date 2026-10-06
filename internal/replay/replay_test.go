package replay

import (
	"math"
	"strings"
	"testing"
	"time"
)

func TestFrameConversions(t *testing.T) {
	tests := []struct {
		frame   Frame
		seconds float64
		str     string
	}{
		{0, 0, "0:00"},
		{1, 0.042, "0:00"},
		{24, 1.008, "0:01"},
		{1952, 81.984, "1:21"},
		{1953, 82.026, "1:22"},
		{30979, 1301.118, "21:41"},
		{85715, 3600.03, "1:00:00"},
	}
	for _, tt := range tests {
		if got := tt.frame.Seconds(); math.Abs(got-tt.seconds) > 1e-9 {
			t.Errorf("Frame(%d).Seconds() = %v, want %v", tt.frame, got, tt.seconds)
		}
		if got := tt.frame.String(); got != tt.str {
			t.Errorf("Frame(%d).String() = %q, want %q", tt.frame, got, tt.str)
		}
	}
}

func TestFrameAt(t *testing.T) {
	tests := []struct {
		d    time.Duration
		want Frame
	}{
		{0, 0},
		{42 * time.Millisecond, 1},
		{43 * time.Millisecond, 2},
		{time.Minute, 1429},
	}
	for _, tt := range tests {
		if got := FrameAt(tt.d); got != tt.want {
			t.Errorf("FrameAt(%v) = %d, want %d", tt.d, got, tt.want)
		}
	}
	if FramesPerMinute != 1428 {
		t.Errorf("FramesPerMinute = %d, want 1428", FramesPerMinute)
	}
}

func TestSkipReason(t *testing.T) {
	human := func(id byte, race Race, team int) Player {
		return Player{ID: id, Race: race, Team: team, Human: true}
	}
	obs := Player{ID: 2, Race: Terran, Team: 3, Human: true, Observer: true}
	cpu := Player{ID: 255, Race: Zerg, Team: 2}
	long := FrameAt(10 * time.Minute)

	tests := []struct {
		name    string
		players []Player
		frames  Frame
		want    string // substring; "" means kept
	}{
		{"1v1", []Player{human(0, Zerg, 1), human(1, Terran, 2)}, long, ""},
		{"1v1 with observer", []Player{human(0, Zerg, 1), human(1, Terran, 2), obs}, long, ""},
		{"exactly two minutes", []Player{human(0, Zerg, 1), human(1, Terran, 2)}, FrameAt(MinDuration), ""},
		{"too short", []Player{human(0, Zerg, 1), human(1, Terran, 2)}, FrameAt(MinDuration) - 1, "too short"},
		{"no players", nil, long, "0 non-observer players"},
		{"one player", []Player{human(0, Zerg, 1)}, long, "1 non-observer players"},
		{"2v2", []Player{human(0, Zerg, 1), human(1, Zerg, 1), human(2, Terran, 2), human(3, Protoss, 2)}, long, "4 non-observer players"},
		{"vs computer", []Player{human(0, Zerg, 1), cpu}, long, "computer player"},
		{"same team", []Player{human(0, Zerg, 1), human(1, Terran, 1)}, long, "same team"},
		{"unknown race", []Player{human(0, RaceUnknown, 1), human(1, Terran, 2)}, long, "unknown race"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &Replay{Players: tt.players, Frames: tt.frames}
			got := r.SkipReason()
			if tt.want == "" && got != "" {
				t.Fatalf("SkipReason() = %q, want kept", got)
			}
			if tt.want != "" && !strings.Contains(got, tt.want) {
				t.Fatalf("SkipReason() = %q, want it to contain %q", got, tt.want)
			}
		})
	}
}

func TestContentHash(t *testing.T) {
	// sha256 of the empty input.
	const empty = "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
	if got := ContentHash(nil); got != empty {
		t.Errorf("ContentHash(nil) = %s", got)
	}
}
