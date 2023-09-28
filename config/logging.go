package config

import (
	"log/slog"
	"os"
	"time"

	"github.com/Vilsol/go-pob/utils"
	"github.com/lmittmann/tint"
)

func InitLogging(withTime bool) {
	timeFormat := time.TimeOnly
	if withTime {
		timeFormat = "2006-01-02T15:04:05.000Z07:00"
	}

	slog.SetDefault(slog.New(
		tint.NewHandler(os.Stderr, &tint.Options{
			Level:      utils.LevelTrace,
			AddSource:  true,
			TimeFormat: timeFormat,
			ReplaceAttr: func(groups []string, attr slog.Attr) slog.Attr {
				if attr.Key == slog.LevelKey {
					level := attr.Value.Any().(slog.Level)
					var levelLabel string
					switch level {
					case utils.LevelTrace:
						levelLabel = "TRACE"
					default:
						levelLabel = level.String()
					}
					attr.Value = slog.StringValue(levelLabel)
				}
				return attr
			},
		}),
	))
}
