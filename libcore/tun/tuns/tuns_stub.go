//go:build !android

package tuns

import (
	"errors"
	"libcore/tun"
	"os"
)

func NewGvisor(dev int32, mtu int32, handler tun.Handler, nicId int32, pcap bool, pcapFile *os.File, snapLen uint32, ipv6Mode int32) (tun.Tun, error) {
	return nil, errors.New("not for your platform")
}

func NewSystem(dev int32, mtu int32, handler tun.Handler, ipv6Mode int32, errorHandler func(err string)) (tun.Tun, error) {
	return nil, errors.New("not for your platform")
}

func NewTun2Socket(fd int32, handler tun.Handler) (tun.Tun, error) {
	return nil, errors.New("not for your platform")
}
