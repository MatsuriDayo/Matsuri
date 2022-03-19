package tun2socket

import (
	"libcore/tun"
	"net"
	"os"
	"time"

	"github.com/Dreamacro/clash/common/pool"
	v2rayNet "github.com/v2fly/v2ray-core/v5/common/net"
)

// this file is a wrapper for libcore

func New(fd int32, handler tun.Handler) (*Tun2Socket, error) {
	var stack *Tun2Socket

	device := os.NewFile(uintptr(fd), "/dev/tun")
	stack, err := StartTun2Socket(device)
	if err != nil {
		return nil, err
	}

	tcp := func() {
		defer stack.TCP().Close()

		for stack.TCP().SetDeadline(time.Time{}) == nil {
			conn, err := stack.TCP().Accept()
			if err != nil {
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
				return
			}

			raw := buf[:n]
			lAddr := lRAddr.(*net.UDPAddr)
			rAddr := rRAddr.(*net.UDPAddr)

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

			go handler.NewPacket(source, destination,
				&tun.UDPPacket{
					Data: raw,
					Put:  func() { pool.Put(buf) },
				},
				func(b []byte, addr *net.UDPAddr) (int, error) {
					// this is downlink
					return stack.UDP().WriteTo(b, addr, lAddr)
				})
		}
	}

	go tcp()

	// how many uplink worker?
	go udp()
	go udp()

	return stack, nil
}
