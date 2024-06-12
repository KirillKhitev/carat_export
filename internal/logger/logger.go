package logger

import (
	"github.com/KirillKhitev/carat_export/internal/config"
	"github.com/sirupsen/logrus"
	"os"
	"time"
)

type Logger struct {
	*logrus.Logger
	File *os.File
}

func NewLog() *Logger {
	return &Logger{
		Logger: logrus.New(),
	}
}

var Log = NewLog()

func (l *Logger) Restart() {
	l.File.Close()

	Initialize(config.Config.LogLevel)
}

func Initialize(level string) {
	levelLog, err := logrus.ParseLevel(level)
	if err != nil {
		Log.Info("Не удалось распарсить уровень логирования, используем Debug")
		levelLog = logrus.DebugLevel
	}

	Log.SetLevel(levelLog)
	Log.SetFormatter(&logrus.JSONFormatter{})

	filename := prepareFileName()

	file, err := os.OpenFile(filename, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err == nil {
		Log.SetOutput(file)
		Log.File = file
	} else {
		Log.Info("Не удалось открыть файл логов, используется стандартный stderr")
	}
}

func prepareFileName() string {
	result := config.Config.LogDir + string(os.PathSeparator) + time.Now().Format(time.DateOnly) + ".log"

	return result
}
