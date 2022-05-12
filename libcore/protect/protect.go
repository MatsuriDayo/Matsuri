package protect

import (
	"net"

	"github.com/v2fly/v2ray-core/v5/transport/internet"
)

var FdProtector Protector
var v2rayDefaultDialer = &internet.DefaultSystemDialer{}

type Protector interface {
	Protect(fd int32) bool
}

// TODO now it is v2ray's default dialer, test for bug (VPN / non-VPN)
type ProtectedDialer struct {
	Resolver func(domain string) ([]net.IP, error)
}
