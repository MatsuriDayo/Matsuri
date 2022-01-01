package libcore

import (
	"net"
	"strings"

	"github.com/v2fly/v2ray-core/v5/nekoutils"
)

func SetConfig(tryDomainStr string, disableExtraCoreLog bool, nekoConnLog bool) {
	tryDomains = strings.Split(tryDomainStr, ",")
	staticHosts = make(map[string][]net.IP)

	nekoutils.Connection_V2Ray_Enabled = nekoConnLog
	ResetAllConnections(false)

	if disableExtraCoreLog {
		v2rayLogHook = func(s string) string {
			patterns := []string{
				"Logger closing",
				"Logger started",
				"DNS: created",
			}
			for _, p := range patterns {
				if strings.Contains(s, p) {
					return ""
				}
			}
			return s
		}
	} else {
		v2rayLogHook = nil
	}
}
