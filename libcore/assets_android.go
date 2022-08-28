//go:build android

package libcore

import (
	"io"
	"io/ioutil"
	"libcore/comm"
	"os"
	"path/filepath"
	"strconv"

	"github.com/sirupsen/logrus"
	"github.com/v2fly/v2ray-core/v5/common/platform/filesystem"
	"golang.org/x/mobile/asset"
)

func extractV2RayAssets(useOfficial BoolFunc) {
	useOfficialAssets := useOfficial.Invoke()

	extract := func(name string) {
		err := extractAssetName(name, useOfficialAssets)
		if err != nil {
			logrus.Warnf("Extract %s failed: %v", geoipDat, err)
		}
	}

	extract(geoipDat)
	extract(geositeDat)
	extract(browserForwarder)
}

// 这里解压的是 apk 里面的
func extractAssetName(name string, useOfficialAssets bool) error {
	// 支持非官方源的，就是 replaceable，放 Android 目录
	// 不支持非官方源的，就放 file 目录
	replaceable := true

	var version string
	switch name {
	case geoipDat:
		version = geoipVersion
	case geositeDat:
		version = geositeVersion
	case browserForwarder:
		version = coreVersion
		replaceable = false
	}

	var dir string
	if !replaceable {
		dir = internalAssetsPath
	} else {
		dir = externalAssetsPath
	}

	var localVersion string
	var assetVersion string

	// loadAssetVersion from APK
	loadAssetVersion := func() error {
		av, err := asset.Open(assetsPrefix + version)
		if err != nil {
			return newError("open version in assets").Base(err)
		}
		b, err := ioutil.ReadAll(av)
		comm.CloseIgnore(av)
		if err != nil {
			return newError("read internal version").Base(err)
		}
		assetVersion = string(b)
		return nil
	}
	if err := loadAssetVersion(); err != nil {
		return err
	}

	doExtract := false
	fileMissing := false

	if _, err := os.Stat(dir + version); err != nil {
		fileMissing = true
	} else if _, err := os.Stat(dir + name); err != nil {
		fileMissing = true
	}

	if fileMissing {
		doExtract = true
	} else if useOfficialAssets || !replaceable {
		// 官方源升级
		b, err := ioutil.ReadFile(dir + version)
		if err != nil {
			doExtract = true
			_ = os.RemoveAll(version)
		} else {
			localVersion = string(b)
			av, err := strconv.ParseUint(assetVersion, 10, 64)
			if err != nil {
				doExtract = assetVersion != localVersion
			} else {
				lv, err := strconv.ParseUint(localVersion, 10, 64)
				doExtract = err != nil || av > lv
			}
		}
	} else {
		//非官方源不升级
	}

	if !doExtract {
		return nil
	}

	err := extractAsset(assetsPrefix+name+".xz", dir+name)
	if err == nil {
		err = unxz(dir + name)

	}
	if err != nil {
		return err
	}

	o, err := os.Create(dir + version)
	if err != nil {
		return err
	}
	_, err = io.WriteString(o, assetVersion)
	comm.CloseIgnore(o)
	return err
}

func extractAsset(assetPath string, path string) error {
	i, err := asset.Open(assetPath)
	if err != nil {
		return err
	}
	defer comm.CloseIgnore(i)
	o, err := os.Create(path)
	if err != nil {
		return err
	}
	defer comm.CloseIgnore(o)
	_, err = io.Copy(o, i)
	if err == nil {
		logrus.Debugf("Extract >> %s", path)
	}
	return err
}

func setupV2rayFileSystem(internalAssets, externalAssets string) {
	filesystem.NewFileSeeker = func(path string) (io.ReadSeekCloser, error) {
		_, fileName := filepath.Split(path)

		// 直接读 APK 的
		if fileName == "index.html" {
			av, err := asset.Open(assetsPrefix + fileName)
			if err != nil {
				return nil, newError("open " + fileName + " in assets").Base(err)
			}
			return av, nil
		}

		paths := []string{
			internalAssets + fileName,
			externalAssets + fileName,
		}

		var err error

		for _, path = range paths {
			_, err = os.Stat(path)
			if err == nil {
				return os.Open(path)
			}
		}

		return nil, err
	}

	filesystem.NewFileReader = func(path string) (io.ReadCloser, error) {
		return filesystem.NewFileSeeker(path)
	}
}
