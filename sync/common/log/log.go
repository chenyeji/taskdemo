package log

import (
	"fmt"
	"io"
	"os"
	"runtime"
	"strings"

	"github.com/sirupsen/logrus"

	"gitlab.com/sync/common/config"
)

var (
	defaultFieldMap = map[string]string{}
)

func callerPrettyfier(f *runtime.Frame) (string, string) {
	fileName := fmt.Sprintf("%s:%d", f.File, f.Line)
	funcName := f.Function
	list := strings.Split(funcName, "/")
	if len(list) > 0 {
		funcName = list[len(list)-1]
	}
	return funcName, fileName
}

// for stdout
func callerFormatter(f *runtime.Frame) string {
	funcName, fileName := callerPrettyfier(f)
	return " @" + funcName + " " + fileName
}

func init() {
	logrus.SetReportCaller(true)
	logrus.SetLevel(logrus.DebugLevel)
	logrus.SetOutput(os.Stdout)
	logrus.SetFormatter(&logrus.TextFormatter{
		DisableTimestamp: false,
		CallerPrettyfier: callerPrettyfier,
	})
}

func SetLogDetailsByConfig(cfg *config.Config) {
	var err error
	var hostname string

	if hostname, err = os.Hostname(); err != nil {
		hostname = "localhost"
	}
	AddField("host", hostname)
	AddField("name", cfg.App.Name)

	if cfg.Log.Stdout.Enable {
		AddConsoleOut(cfg.Log.Stdout.Level)
	} else {
		logrus.Warn("no standard output log now")
		DisableDefaultConsole()
	}
	if cfg.Log.File.Enable {
		err = AddFileOut(cfg.Log.File.Path, cfg.Log.File.Level, cfg.Log.File.MaxAge)
		if err != nil {
			logrus.Fatal(err)
		}
	}
}

// AddField add log fields
func AddField(key, value string) {
	if len(key) == 0 {
		return
	}
	if len(value) == 0 {
		return
	}
	defaultFieldMap[key] = value
}

// DisableDefaultConsole 取消默认的控制台输出
func DisableDefaultConsole() {
	logrus.SetOutput(io.Discard)
}

func getHookLevel(level int) []logrus.Level {
	if level < 0 || level > 5 {
		level = 5
	}
	return logrus.AllLevels[:level+1]
}
