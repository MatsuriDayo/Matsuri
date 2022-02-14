package libcore

import (
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"libcore/device"
	"os"
	"path/filepath"
	"runtime/debug"
	"strings"
	"time"
	_ "unsafe"

	"github.com/sagernet/libping"
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

func InitCore(internalAssets string, externalAssets string, prefix string, useOfficial BoolFunc, // extractV2RayAssets
	cachePath string, process string, //InitCore
	enableLog bool, maxKB int32, //SetEnableLog
) {
	defer func() {
		if r := recover(); r != nil {
			s := fmt.Sprintln("InitCore panic", time.Now().Unix(), r, string(debug.Stack()))
			forceLog(s)
		}
	}()

	isBgProcess := strings.HasSuffix(process, ":bg")

	// Set up log
	SetEnableLog(enableLog, maxKB)
	s := fmt.Sprintln("[Debug] InitCore called", externalAssets, cachePath, process, isBgProcess)
	err := setupLogger(filepath.Join(cachePath, "neko.log"))

	if err == nil {
		go forceLog(s)
	} else {
		// not fatal
		forceLog(fmt.Sprintln("Log not inited:", s, err.Error()))
	}

	// Set up some component
	go func() {
		setupResolvers()
		Setenv("v2ray.conf.geoloader", "memconservative")

		if time.Now().Unix() >= GetExpireTime() {
			outdated = "Your version is too old! Please update!! 版本太旧，请升级！"
		} else if time.Now().Unix() < (GetBuildTime() - 86400) {
			outdated = "Wrong system time! 系统时间错误！"
		}

		// Setup CA Certs
		x509.SystemCertPool()
		roots := x509.NewCertPool()
		roots.AppendCertsFromPEM([]byte(mozillaCA))
		systemRoots = roots
	}()

	if !isBgProcess {
		return
	}

	device.AutoGoMaxProcs()
	device.GoDebug()

	// CA for other programs
	go func() {
		f, err := os.OpenFile(filepath.Join(internalAssets, "ca.pem"), os.O_CREATE|os.O_RDWR, 0644)
		if err != nil {
			forceLog("open ca.pem: " + err.Error())
		} else {
			if b, _ := ioutil.ReadAll(f); b == nil || string(b) != mozillaCA {
				f.Truncate(0)
				f.Seek(0, 0)
				f.Write([]byte(mozillaCA))
			}
			f.Close()
		}
	}()

	// nekomura end
	go extractV2RayAssets(internalAssets, externalAssets, prefix, useOfficial)
}
