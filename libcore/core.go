package libcore

import (
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime/debug"
	"strings"
	"time"
	_ "unsafe"

	"github.com/sagernet/libping"
	"github.com/sirupsen/logrus"
	"github.com/v2fly/v2ray-core/v5/common"
)

//go:linkname systemRoots crypto/x509.systemRoots
var systemRoots *x509.CertPool

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

	// force update
	if time.Now().Unix() >= GetExpireTime() {
		outdated = "Your version is too old! Please update!! 版本太旧，请升级！"
	} else if time.Now().Unix() < (GetBuildTime() - 86400) {
		outdated = "Wrong system time! 系统时间错误！"
	}

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

	if !isBgProcess {
		return
	}

	Setenv("v2ray.conf.geoloader", "memconservative")

	// Set up CA for the bg process
	x509.SystemCertPool()
	roots := x509.NewCertPool()
	roots.AppendCertsFromPEM([]byte(mozillaCA))
	systemRoots = roots

	// CA for other programs
	f, err = os.OpenFile(filepath.Join(internalAssets, "ca.pem"), os.O_CREATE|os.O_RDWR, 0644)
	if err != nil {
		errorHandler.HandleError(err.Error())
	} else {
		if b, _ := ioutil.ReadAll(f); b == nil || string(b) != mozillaCA {
			f.Truncate(0)
			f.Seek(0, 0)
			f.Write([]byte(mozillaCA))
		}
		f.Close()
	}

	// nekomura end
	err = extractV2RayAssets(internalAssets, externalAssets, prefix, useOfficial)
	if err != nil {
		errorHandler.HandleError(err.Error())
	}
}
