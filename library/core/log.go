//go:build android
// +build android

package libcore

/*
   #cgo LDFLAGS: -landroid -llog

   #include <android/log.h>
   #include <string.h>
   #include <stdlib.h>
*/
import "C"
import (
	"fmt"
	"github.com/sirupsen/logrus"
	"log"
	"strings"
	"unsafe"

	appLog "github.com/v2fly/v2ray-core/v4/app/log"
	commonLog "github.com/v2fly/v2ray-core/v4/common/log"
)

var (
	tag      = C.CString("libcore")
	tagV2Ray = C.CString("v2ray-core")
)

var levels = []logrus.Level{
	logrus.PanicLevel,
	logrus.FatalLevel,
	logrus.ErrorLevel,
	logrus.WarnLevel,
	logrus.InfoLevel,
	logrus.DebugLevel,
}

type androidHook struct {
}

type androidFormatter struct{}

func (f *androidFormatter) Format(entry *logrus.Entry) ([]byte, error) {

	msgWithLevel := fmt.Sprint("[", strings.Title(entry.Level.String()), "] ", entry.Message)
	return []byte(msgWithLevel), nil
}

func (hook *androidHook) Levels() []logrus.Level {
	return levels
}

func (hook *androidHook) Fire(e *logrus.Entry) error {
	formatted, err := logrus.StandardLogger().Formatter.Format(e)
	if err != nil {
		return err
	}
	str := C.CString(string(formatted))

	var priority C.int
	switch e.Level {
	case logrus.PanicLevel:
		priority = C.ANDROID_LOG_FATAL
	case logrus.FatalLevel:
		priority = C.ANDROID_LOG_FATAL
	case logrus.ErrorLevel:
		priority = C.ANDROID_LOG_ERROR
	case logrus.WarnLevel:
		priority = C.ANDROID_LOG_WARN
	case logrus.InfoLevel:
		priority = C.ANDROID_LOG_INFO
	case logrus.DebugLevel:
		priority = C.ANDROID_LOG_DEBUG
	}
	C.__android_log_write(priority, tag, str)
	C.free(unsafe.Pointer(str))
	return nil
}

type v2rayLogWriter struct {
}

func (w *v2rayLogWriter) Write(s string) error {

	var priority C.int
	if strings.Contains(s, "[Debug]") {
		s = strings.Replace(s, "[Debug]", "", 1)
		priority = C.ANDROID_LOG_DEBUG
	} else if strings.Contains(s, "[Info]") {
		s = strings.Replace(s, "[Info]", "", 1)
		priority = C.ANDROID_LOG_INFO
	} else if strings.Contains(s, "[Warn]") {
		s = strings.Replace(s, "[Warn]", "", 1)
		priority = C.ANDROID_LOG_WARN
	} else if strings.Contains(s, "[Error]") {
		s = strings.Replace(s, "[Error]", "", 1)
		priority = C.ANDROID_LOG_ERROR
	} else {
		priority = C.ANDROID_LOG_DEBUG
	}

	str := C.CString(strings.TrimSpace(s))
	C.__android_log_write(priority, tagV2Ray, str)
	C.free(unsafe.Pointer(str))
	return nil
}

func (w *v2rayLogWriter) Close() error {
	return nil
}

type stdLogWriter struct{}

func (stdLogWriter) Write(p []byte) (n int, err error) {
	str := C.CString(string(p))
	C.__android_log_write(C.ANDROID_LOG_INFO, tag, str)
	C.free(unsafe.Pointer(str))
	return len(p), nil
}

func init() {
	log.SetOutput(stdLogWriter{})
	log.SetFlags(log.Flags() &^ log.LstdFlags)
	logrus.SetFormatter(&androidFormatter{})
	logrus.AddHook(&androidHook{})

	_ = appLog.RegisterHandlerCreator(appLog.LogType_Console, func(lt appLog.LogType,
		options appLog.HandlerCreatorOptions) (commonLog.Handler, error) {
		return commonLog.NewLogger(func() commonLog.Writer {
			return &v2rayLogWriter{}
		}), nil
	})
}
