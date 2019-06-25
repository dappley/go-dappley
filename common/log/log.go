package log

import (
	"fmt"
	"github.com/lestrrat-go/file-rotatelogs"
	"github.com/rifflock/lfshook"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"io"
	"os"
	"time"
)

func BuildLogAndInit() {
	logNew := new(Log)
	logNew.GetParameters()
	logNew.LogInit()
}

type Log struct {
	write      io.Writer
	level      logrus.Level
	name       string
	count      int
	rotateTime int
	stdout     bool
}

func (logNew *Log) GetParameters() {
	logNew.write = new(LogNullWriter)

	logName := viper.GetString("log.name")
	if logName == "" {
		logName = "dappley.log"
	}
	logNew.name = logName

	logCount := viper.GetInt("log.count")
	if logCount == 0 {
		logCount = 7
	}
	logNew.count = logCount

	rotateTime := viper.GetInt("log.rotateTime")
	if rotateTime == 0 {
		rotateTime = 86400
	}
	logNew.rotateTime = rotateTime

	logLevel := viper.GetString("log.level")
	switch logLevel {
	case "info":
		logNew.level = logrus.InfoLevel
	case "debug":
		logNew.level = logrus.DebugLevel
	case "error":
		logNew.level = logrus.ErrorLevel
	case "warn":
		logNew.level = logrus.WarnLevel
	default:
		logNew.level = logrus.InfoLevel
	}

	logNew.stdout = viper.GetBool("log.stdout")
	fmt.Println(*logNew)
}

type LogNullWriter struct {
}

func (*LogNullWriter) Write(b []byte) (n int, err error) {
	return 0, nil
}

func (logNew *Log) LogInit() {

	logrus.SetReportCaller(true)
	logrus.SetLevel(logrus.InfoLevel)
	logrus.SetFormatter(&logrus.TextFormatter{
		ForceColors:               true,
		EnvironmentOverrideColors: true,
		TimestampFormat:           "2006-01-02 15:04:05",
	})

	if logNew.stdout {
		logrus.SetOutput(os.Stdout)
	} else {
		logrus.SetOutput(logNew.write)
	}

	writer, err := rotatelogs.New(
		logNew.name+".%Y%m%d%H%M%S",
		rotatelogs.WithLinkName(logNew.name),
		rotatelogs.WithRotationTime(time.Second*time.Duration(logNew.rotateTime)),
		rotatelogs.WithMaxAge(-1),
		rotatelogs.WithRotationCount(uint(logNew.count)),
	)
	if err != nil {
		logrus.Errorf("config local file system for logger error: %v", err)
	}

	lfsHook := lfshook.NewHook(lfshook.WriterMap{
		logrus.DebugLevel: writer,
		logrus.InfoLevel:  writer,
		logrus.WarnLevel:  writer,
		logrus.ErrorLevel: writer,
		logrus.FatalLevel: writer,
		logrus.PanicLevel: writer,
	}, &logrus.TextFormatter{DisableColors: true})

	logrus.AddHook(lfsHook)
}
