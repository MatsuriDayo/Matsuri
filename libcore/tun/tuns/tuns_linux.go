package tuns

import (
	"libcore/tun"
	"libcore/tun/gvisor"
	"libcore/tun/system"
	"libcore/tun/tun2socket"
	"os"

	"gvisor.dev/gvisor/pkg/tcpip"
)

func NewGvisor(dev int32, mtu int32, handler tun.Handler, nicId int32, pcap bool, pcapFile *os.File, snapLen uint32, ipv6Mode int32) (tun.Tun, error) {
	return gvisor.New(dev, mtu, handler, tcpip.NICID(nicId), pcap, pcapFile, snapLen, ipv6Mode)
}

func NewSystem(dev int32, mtu int32, handler tun.Handler, ipv6Mode int32, errorHandler func(err string)) (tun.Tun, error) {
	return system.New(dev, mtu, handler, ipv6Mode, errorHandler)
}

func NewTun2Socket(fd int32, handler tun.Handler) (tun.Tun, error) {
	return tun2socket.New(fd, handler)
}
