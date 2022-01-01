package protect

import (
	"context"
	"net"
	"runtime"
	"time"
	_ "unsafe"

	"github.com/sirupsen/logrus"
	"github.com/v2fly/v2ray-core/v5/common/buf"
	v2rayNet "github.com/v2fly/v2ray-core/v5/common/net"
	"github.com/v2fly/v2ray-core/v5/features/dns"
	"github.com/v2fly/v2ray-core/v5/transport/internet"
)

//go:linkname effectiveSystemDialer_ github.com/v2fly/v2ray-core/v5/transport/internet.effectiveSystemDialer
var effectiveSystemDialer_ internet.SystemDialer

var v2rayDefaultDialer *internet.DefaultSystemDialer

func init() {
	var ok bool
	v2rayDefaultDialer, ok = effectiveSystemDialer_.(*internet.DefaultSystemDialer)
	if !ok {
		panic("v2rayDefaultDialer not found")
	}
}

// non-Windows
// May be a func to apply fwmark to the fd (implement by Android or Nekoray)
type Protector interface {
	Protect(fd int32) bool
}

var FdProtector Protector

// Use my dial function on non-Windows platforms
// Use v2ray's dial on Windows
type ProtectedDialer struct {
	Resolver func(domain string) ([]net.IP, error)
}

func (dialer ProtectedDialer) Dial(ctx context.Context, source v2rayNet.Address, destination v2rayNet.Destination, sockopt *internet.SocketConfig) (conn net.Conn, err error) {
	if destination.Network == v2rayNet.Network_Unknown || destination.Address == nil {
		buffer := buf.StackNew()
		buffer.Resize(0, int32(runtime.Stack(buffer.Extend(buf.Size), false)))
		logrus.Warn("connect to invalid destination:\n", buffer.String())
		buffer.Release()

		return nil, newError("invalid destination")
	}

	var ips []net.IP
	if destination.Address.Family().IsDomain() {
		if dialer.Resolver == nil {
			return nil, newError("no resolver")
		}
		ips, err = dialer.Resolver(destination.Address.Domain())
		if err == nil && len(ips) == 0 {
			err = dns.ErrEmptyResponse
		}
		if err != nil {
			return nil, err
		}
	} else {
		ip := destination.Address.IP()
		if ip.IsLoopback() { // is it more effective
			return v2rayDefaultDialer.Dial(ctx, source, destination, sockopt)
		}
		ips = append(ips, ip)
	}

	for i, ip := range ips {
		if i > 0 {
			if err == nil {
				break
			} else {
				logrus.Warn("dial system failed: ", err)
				time.Sleep(time.Millisecond * 200)
			}
			logrus.Debug("trying next address: ", ip.String())
		}
		destination.Address = v2rayNet.IPAddress(ip)
		conn, err = dialer.dial(ctx, source, destination, sockopt)
	}

	return conn, err
}
