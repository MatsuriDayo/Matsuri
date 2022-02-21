package libcore

import (
	"strings"
)

func SetConfig(tryDomainStr string, disableExtraCoreLog bool) {
	tryDomains = strings.Split(tryDomainStr, ",")

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
