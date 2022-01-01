package libcore

const (
	geoipDat         = "geoip.dat"
	geositeDat       = "geosite.dat"
	browserForwarder = "index.js"
	geoipVersion     = "geoip.version.txt"
	geositeVersion   = "geosite.version.txt"
	coreVersion      = "core.version.txt"
)

var assetsPrefix string
var internalAssetsPath string
var externalAssetsPath string

type BoolFunc interface {
	Invoke() bool
}
