package config

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/Vilsol/go-pob/utils"
	"github.com/lmittmann/tint"
)

const (
	ansiReset         = "\033[0m"
	ansiBold          = "\033[1m"
	ansiWhite         = "\033[38m"
	ansiBrightMagenta = "\033[95m"
)

func InitLogging(withTime bool) {
	slog.SetDefault(slog.New(
		TimeStripper{
			Upstream: tint.NewHandler(os.Stderr, &tint.Options{
				Level:      utils.LevelTrace,
				AddSource:  true,
				TimeFormat: "2006-01-02T15:04:05.000Z07:00",
				ReplaceAttr: func(groups []string, attr slog.Attr) slog.Attr {
					if attr.Key == slog.LevelKey {
						level := attr.Value.Any().(slog.Level)
						switch level {
						case utils.LevelTrace:
							attr.Value = slog.StringValue("TRC")
						case slog.LevelDebug:
							attr.Value = slog.StringValue(ansiBrightMagenta + "DBG" + ansiReset)
						}
					} else if attr.Key == slog.MessageKey {
						attr.Value = slog.StringValue(ansiBold + ansiWhite + fmt.Sprint(attr.Value.Any()) + ansiReset)
					}
					return attr
				},
			}),
			Strip: !withTime,
		},
	))
}

var _ slog.Handler = (*TimeStripper)(nil)

type TimeStripper struct {
	Upstream slog.Handler
	Strip    bool
}

func (t TimeStripper) Enabled(ctx context.Context, level slog.Level) bool {
	return t.Upstream.Enabled(ctx, level)
}

func (t TimeStripper) Handle(ctx context.Context, record slog.Record) error {
	if t.Strip {
		record.Time = time.Time{}
	}

	return t.Upstream.Handle(ctx, record) //nolint:wrapcheck
}

func (t TimeStripper) WithAttrs(attrs []slog.Attr) slog.Handler {
	return t.Upstream.WithAttrs(attrs)
}

func (t TimeStripper) WithGroup(name string) slog.Handler {
	return t.Upstream.WithGroup(name)
}
