package tun2socket

import (
	"libcore/tun"
	"net"
	"os"
	"time"

	"github.com/Dreamacro/clash/common/pool"
	"github.com/sirupsen/logrus"
	v2rayNet "github.com/v2fly/v2ray-core/v5/common/net"
)

// this file is a wrapper for libcore

type udpCloser struct {
	buf []byte
}

func (u *udpCloser) Close() error {
	return pool.Put(u.buf)
}

func New(fd int32, handler tun.Handler) (*Tun2Socket, error) {
	var stack *Tun2Socket

	device := os.NewFile(uintptr(fd), "/dev/tun")
	stack, err := StartTun2Socket(device, net.ParseIP(tun.PRIVATE_VLAN4_CLIENT), net.ParseIP(tun.PRIVATE_VLAN4_ROUTER))
	if err != nil {
		return nil, err
	}

	tcp := func() {
		defer stack.TCP().Close()

		for stack.TCP().SetDeadline(time.Time{}) == nil {
			conn, err := stack.TCP().Accept()
			if err != nil {
				logrus.Debugln("Accept TCP error:", err)
				continue
			}

			lAddr := conn.LocalAddr().(*net.TCPAddr)
			rAddr := conn.RemoteAddr().(*net.TCPAddr)

			source := v2rayNet.Destination{
				Address: v2rayNet.IPAddress(lAddr.IP),
				Port:    v2rayNet.Port(lAddr.Port),
				Network: v2rayNet.Network_TCP,
			}
			destination := v2rayNet.Destination{
				Address: v2rayNet.IPAddress(rAddr.IP),
				Port:    v2rayNet.Port(rAddr.Port),
				Network: v2rayNet.Network_TCP,
			}

			go handler.NewConnection(source, destination, conn)
		}
	}

	udp := func() {
		defer stack.UDP().Close()

		for {
			buf := pool.Get(pool.UDPBufferSize)

			n, lRAddr, rRAddr, err := stack.UDP().ReadFrom(buf)
			if err != nil {
				logrus.Debugln("ReadFrom UDP error:", err)
				return
			}

			raw := buf[:n]
			lAddr := lRAddr.(*net.UDPAddr)
			rAddr := rRAddr.(*net.UDPAddr)

			logrus.Debugln("UDP", lAddr.String(), rAddr.String(), len(raw))

			if rAddr.IP.IsLoopback() {
				pool.Put(buf)
				continue
			}

			source := v2rayNet.Destination{
				Address: v2rayNet.IPAddress(lAddr.IP),
				Port:    v2rayNet.Port(lAddr.Port),
				Network: v2rayNet.Network_UDP,
			}
			destination := v2rayNet.Destination{
				Address: v2rayNet.IPAddress(rAddr.IP),
				Port:    v2rayNet.Port(rAddr.Port),
				Network: v2rayNet.Network_UDP,
			}

			go handler.NewPacket(source, destination, raw, func(b []byte, addr *net.UDPAddr) (int, error) {
				return stack.UDP().WriteTo(b, addr, lAddr)
			}, &udpCloser{buf})
		}
	}

	go tcp()
	go udp()

	return stack, nil
}
