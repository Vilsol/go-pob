package calculator

import (
	"bytes"
	"log/slog"
	"testing"
)

// TestSpecialModNamedCaptureLogsNothing: a special mod whose first capture is a name ("Frostbite") passes Lua's
// tonumber nil, without a warning per parse.
func TestSpecialModNamedCaptureLogsNothing(t *testing.T) {
	var logged bytes.Buffer
	prev := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&logged, &slog.HandlerOptions{Level: slog.LevelWarn})))
	t.Cleanup(func() { slog.SetDefault(prev) })

	if ParseMod("Curse Enemies with Frostbite on Hit", false) == nil {
		t.Fatal("the line did not parse")
	}
	if logged.Len() != 0 {
		t.Fatalf("parsing logged: %s", logged.String())
	}
}
