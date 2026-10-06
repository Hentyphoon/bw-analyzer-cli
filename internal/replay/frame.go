package replay

import (
	"fmt"
	"time"
)

// Frame is a game frame. Brood War counts time in frames; at Fastest speed
// one frame is 42 ms. Every frame-to-time conversion goes through this type.
type Frame int32

// FrameDuration is the length of one frame at Fastest speed.
const FrameDuration = 42 * time.Millisecond

// FramesPerMinute is the number of whole frames in one game minute.
const FramesPerMinute = Frame(time.Minute / FrameDuration)

// FrameAt returns the first frame at or after d.
func FrameAt(d time.Duration) Frame {
	return Frame((d + FrameDuration - 1) / FrameDuration)
}

// Duration returns the game time at frame f.
func (f Frame) Duration() time.Duration {
	return time.Duration(f) * FrameDuration
}

// Seconds returns the game time at frame f in seconds.
func (f Frame) Seconds() float64 {
	return f.Duration().Seconds()
}

// String formats f as m:ss, or h:mm:ss past an hour.
func (f Frame) String() string {
	total := int64(f.Duration() / time.Second)
	h, m, s := total/3600, total/60%60, total%60
	if h > 0 {
		return fmt.Sprintf("%d:%02d:%02d", h, m, s)
	}
	return fmt.Sprintf("%d:%02d", m, s)
}
