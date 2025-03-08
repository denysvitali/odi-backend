package logutils

import "github.com/sirupsen/logrus"

var log = logrus.StandardLogger()

func SetLoggerLevel(level string) {
	lvl, err := logrus.ParseLevel(level)
	if err != nil {
		lvl = logrus.InfoLevel
	}
	log.SetLevel(lvl)
}
