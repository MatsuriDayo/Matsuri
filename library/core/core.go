package libcore

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime/debug"
	"strings"

	"github.com/sagernet/libping"
	"github.com/sirupsen/logrus"
	"github.com/v2fly/v2ray-core/v5/common"
)

func Setenv(key, value string) error {
	return os.Setenv(key, value)
}

func Unsetenv(key string) error {
	return os.Unsetenv(key)
}

func IcmpPing(address string, timeout int32) (int32, error) {
	return libping.IcmpPing(address, timeout)
}

func closeIgnore(closer ...interface{}) {
	for _, c := range closer {
		if ca, ok := c.(common.Closable); ok {
			_ = ca.Close()
		} else if ia, ok := c.(common.Interruptible); ok {
			ia.Interrupt()
		}
	}
}

func InitCore(internalAssets string, externalAssets string, prefix string, useOfficial BoolFunc,
	cachePath string, errorHandler ErrorHandler,
) {
	defer func() { // TODO receive core panic log (other goroutine)
		if r := recover(); r != nil {
			if errorHandler != nil {
				s := fmt.Sprintln("InitCore panic: ", r, debug.Stack())
				errorHandler.HandleError(s)
				logrus.Errorln(s)
			}
		}
	}()

	// Is background process
	var processName string
	var isBgProcess bool
	f, _ := os.Open("/proc/self/cmdline")
	if f != nil {
		b, _ := ioutil.ReadAll(f)
		processName = strings.Trim(string(b), "\x00")
		isBgProcess = strings.HasSuffix(processName, ":bg")
		f.Close()
	} else {
		processName = "(error)"
		isBgProcess = true
	}

	// Set up log
	s := fmt.Sprintln("InitCore called", externalAssets, cachePath, os.Getpid(), processName, isBgProcess)
	err := setupLogger(filepath.Join(cachePath, "neko.log"))
	if err == nil {
		logrus.Debugln(s)
	} else { // not fatal
		errorHandler.HandleError(fmt.Sprintln("Log not inited:", s, err.Error()))
	}

	// Set up some go component
	setupResolvers()

	// nekomura end

	Setenv("v2ray.conf.geoloader", "memconservative")

	if !isBgProcess {
		return
	}

	err = extractV2RayAssets(internalAssets, externalAssets, prefix, useOfficial)
	if err != nil {
		errorHandler.HandleError(err.Error())
	}
}
