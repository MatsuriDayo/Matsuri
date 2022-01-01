package libcore

import "github.com/sagernet/libping"

func IcmpPing(address string, timeout int32) (int32, error) {
	return libping.IcmpPing(address, timeout)
}
