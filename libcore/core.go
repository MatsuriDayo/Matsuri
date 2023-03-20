package libcore

import (
	"fmt"
	"libcore/device"
	"os"
	"path/filepath"
	"strings"
)

func Setenv(key, value string) error {
	return os.Setenv(key, value)
}

func Unsetenv(key string) error {
	return os.Unsetenv(key)
}

func initCoreDefer() {
	device.AllDefer("InitCore", ForceLog)
}

func InitCore(internalAssets string, externalAssets string, prefix string, useOfficial BoolFunc, // extractV2RayAssets
	cachePath string, process string, //InitCore
	enableLog bool, maxKB int32, //SetEnableLog
) {
	defer initCoreDefer()

	isBgProcess := strings.HasSuffix(process, ":bg")

	// Working dir
	if !device.IsNekoray {
		os.Chdir(filepath.Join(cachePath, "../no_backup"))
	}

	// Set up log
	SetEnableLog(enableLog, maxKB)
	s := fmt.Sprintln("InitCore called", externalAssets, cachePath, process, isBgProcess)
	s = strings.TrimRight(s, "\n")
	err := setupLogger(filepath.Join(cachePath, "neko.log"))

	if err == nil {
		go ForceLog(s)
	} else {
		// not fatal
		ForceLog(fmt.Sprintln("Log not inited:", s, err.Error()))
	}

	// Set up some component
	go func() {
		defer initCoreDefer()
		device.GoDebug(process)

		externalAssetsPath = externalAssets
		internalAssetsPath = internalAssets
		assetsPrefix = prefix
		Setenv("v2ray.conf.geoloader", "memconservative")

		setupV2rayFileSystem(internalAssets, externalAssets)
		setupResolvers()

		// Extract assets
		if isBgProcess {
			extractV2RayAssets(useOfficial)
		}
	}()

	if !isBgProcess {
		return
	}

	device.AutoGoMaxProcs()
}
