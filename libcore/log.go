package libcore

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"libcore/device"
	"log"
	"os"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/sirupsen/logrus"

	appLog "github.com/v2fly/v2ray-core/v5/app/log"
	commonLog "github.com/v2fly/v2ray-core/v5/common/log"
)

type logrusFormatter struct{}

var _logrusFormatter = &logrusFormatter{}

func (f *logrusFormatter) Format(entry *logrus.Entry) ([]byte, error) {
	msg := fmt.Sprint("[", entry.Time.Format("2006-01-02 15:04:05"), "] ")
	if entry.Level != logrus.Level(114514) {
		msg += fmt.Sprint("[", strings.Title(entry.Level.String()), "] ")
	}
	for k, v := range entry.Data {
		msg += fmt.Sprintf("[%s=%v] ", k, v)
	}
	msg += entry.Message
	msg += "\n"
	return []byte(msg), nil
}

type v2rayLogWriter struct {
}

var v2rayLogHook func(s string) string

func (w *v2rayLogWriter) Write(s string) error {

	var priority logrus.Level
	if strings.Contains(s, "[Debug]") {
		s = strings.Replace(s, "[Debug]", "", 1)
		priority = logrus.DebugLevel
	} else if strings.Contains(s, "[Info]") {
		s = strings.Replace(s, "[Info]", "", 1)
		priority = logrus.InfoLevel
	} else if strings.Contains(s, "[Warn]") {
		s = strings.Replace(s, "[Warn]", "", 1)
		priority = logrus.WarnLevel
	} else if strings.Contains(s, "[Warning]") {
		s = strings.Replace(s, "[Warning]", "", 1)
		priority = logrus.WarnLevel
	} else if strings.Contains(s, "[Error]") {
		s = strings.Replace(s, "[Error]", "", 1)
		priority = logrus.ErrorLevel
	} else {
		priority = logrus.Level(114514)
	}

	if v2rayLogHook != nil {
		s = v2rayLogHook(s)
		if s == "" {
			return nil
		}
	}

	NekoLogWrite(int32(priority), "", strings.Trim(s, " "))
	return nil
}

func (w *v2rayLogWriter) Close() error {
	return nil
}

type stdLogWriter struct{}

func (stdLogWriter) Write(p []byte) (n int, err error) {
	NekoLogWrite(int32(logrus.InfoLevel), "std", string(p))
	return len(p), nil
}

// manage log file
var _logFile = &logfile{}
var _logMaxSize = 50 * 1024

type logfile struct {
	f     *os.File
	buf   bytes.Buffer
	mutex sync.Mutex
}

func (lp *logfile) lock() {
	if lp.f != nil && runtime.GOOS != "windows" {
		device.Flock(int(lp.f.Fd()), device.LOCK_EX)
	} else {
		lp.mutex.Lock()
	}
}
func (lp *logfile) unlock() {
	if lp.f != nil && runtime.GOOS != "windows" {
		device.Flock(int(lp.f.Fd()), device.LOCK_UN)
	} else {
		lp.mutex.Unlock()
	}
}

func (lp *logfile) Write(p []byte) (n int, err error) {
	// locked, don't call NekoLogWrite or logrus here.
	lp.lock()
	defer lp.unlock()

	// Truncate long file
	if lp.f != nil {
		if offset, _ := lp.f.Seek(0, io.SeekEnd); offset > int64(_logMaxSize) {
			lp.f.Seek(0, io.SeekStart)
			data, _ := ioutil.ReadAll(lp.f)
			if len(data)-_logMaxSize > 0 {
				err := lp.f.Truncate(0)
				// TODO windows "access is denied"
				if err == nil {
					lp.f.Write(data[len(data)-_logMaxSize:])
				}
			}
		}
	} else {
		if lp.buf.Len() > _logMaxSize {
			data := lp.buf.Bytes()
			if len(data)-_logMaxSize > 0 {
				lp.buf.Reset()
				lp.buf.Write(data[len(data)-_logMaxSize:])
			}
		}
	}

	if device.IsNekoray {
		os.Stdout.Write(p)
	}

	//TODO log by entry, show color
	if lp.f != nil {
		return lp.f.Write(p)
	} else {
		return lp.buf.Write(p)
	}
}

func (lp *logfile) Clear() {
	lp.lock()
	defer lp.unlock()

	lp.f.Truncate(0)
}

func (lp *logfile) Get() []byte {
	lp.lock()
	defer lp.unlock()

	var a []byte

	if lp.f != nil {
		lp.f.Seek(0, io.SeekStart)
		a, _ = ioutil.ReadAll(lp.f)
	} else {
		a = lp.buf.Bytes()
	}

	if a == nil || len(a) == 0 { //this crash
		return []byte{0}
	}
	return a
}

func (lp *logfile) init(path string) (err error) {
	lp.lock()
	defer lp.unlock()

	if runtime.GOOS == "windows" {
		oldF, err := os.ReadFile(path)
		if err == nil && len(oldF) > _logMaxSize {
			if os.Truncate(path, 0) == nil {
				lp.buf.Write(oldF[len(oldF)-_logMaxSize:])
			}
		}
	}

	lp.f, err = os.OpenFile(path, os.O_RDWR|os.O_APPEND|os.O_CREATE, 0644)
	if err != nil {
		return
	}

	// read buffer (it's fake, can't read in diffrent process)
	b := lp.buf.Bytes()
	if len(b) > 0 {
		lp.f.Write(b)
		lp.buf.Reset()
	}
	return
}

func ForceLog(str string) {
	entry := &logrus.Entry{
		Time:    time.Now(),
		Level:   logrus.Level(114514),
		Message: str,
	}
	b, _ := _logrusFormatter.Format(entry)
	_logFile.Write(b)
}

func NekoLogWrite(level int32, tag, str string) {
	if level == 114514 {
		if logrus.GetLevel() > logrus.FatalLevel {
			ForceLog(strings.Trim(str, "\n"))
		}
	} else {
		if tag == "" {
			logrus.StandardLogger().Log(logrus.Level(level), strings.Trim(str, "\n"))
		} else {
			logrus.StandardLogger().WithField("tag", tag).Log(logrus.Level(level), strings.Trim(str, "\n"))
		}
	}
}

func NekoLogClear() {
	_logFile.Clear()
}

func NekoLogGet() []byte {
	return _logFile.Get()
}

func SetEnableLog(enableLog bool, maxKB int32) {
	if enableLog {
		logrus.SetLevel(logrus.DebugLevel)
	} else {
		logrus.SetLevel(logrus.FatalLevel)
	}
	if maxKB > 0 {
		_logMaxSize = int(maxKB) * 1024
	}
}

func setupLogger(path string) (err error) {
	//init neko logger
	logrus.SetFormatter(_logrusFormatter)
	err = _logFile.init(path)
	logrus.SetOutput(_logFile)

	//replace loggers
	log.SetOutput(stdLogWriter{})
	log.SetFlags(log.Flags() &^ log.LstdFlags)

	_ = appLog.RegisterHandlerCreator(appLog.LogType_Console, func(lt appLog.LogType,
		options appLog.HandlerCreatorOptions) (commonLog.Handler, error) {
		return commonLog.NewLogger(func() commonLog.Writer {
			return &v2rayLogWriter{}
		}), nil
	})

	return
}
