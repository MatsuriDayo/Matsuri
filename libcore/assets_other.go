//go:build !android

package libcore

func extractV2RayAssets(useOfficial BoolFunc) {}

func setupV2rayFileSystem(internalAssets, externalAssets string) {}
