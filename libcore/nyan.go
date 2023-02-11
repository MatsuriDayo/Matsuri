package libcore

import (
	"net"
	"runtime"
	"strings"

	"github.com/sirupsen/logrus"
)

func SetConfig(tryDomainStr string, filterCoreLog bool) {
	// this is orphan

	tryDomains = strings.Split(tryDomainStr, ",")
	staticHosts = make(map[string][]net.IP)

	if filterCoreLog {
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

func ForceGc() {
	go func() {
		logrus.Infoln("[APP] request force GC")
		runtime.GC()
	}()
}
