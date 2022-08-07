package config

import (
	"os"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func InitLogging(withTime bool) {
	zerolog.SetGlobalLevel(zerolog.TraceLevel)

	var timeFormatter func(i interface{}) string
	if !withTime {
		timeFormatter = func(i interface{}) string {
			return ""
		}
	}

	log.Logger = log.Output(zerolog.ConsoleWriter{
		Out:             os.Stderr,
		TimeFormat:      "2006-01-02T15:04:05.000Z07:00",
		FormatTimestamp: timeFormatter,
	})
}
