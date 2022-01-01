//go:build !windows

package protect

import (
	"context"
	"fmt"
	"net"
	"os"
	"syscall"

	v2rayNet "github.com/v2fly/v2ray-core/v5/common/net"
	"github.com/v2fly/v2ray-core/v5/nekoutils"
	"github.com/v2fly/v2ray-core/v5/transport/internet"
	"golang.org/x/sys/unix"
)

func (dialer ProtectedDialer) dial(ctx context.Context, source v2rayNet.Address, destination v2rayNet.Destination, sockopt *internet.SocketConfig) (conn net.Conn, err error) {
	fd, err := getFd(destination.Network)
	if err != nil {
		return nil, err
	}

	// Close fd to stop the connection (e.g. TCP SYN) if failed

	if FdProtector != nil && !FdProtector.Protect(int32(fd)) {
		syscall.Close(fd)
		return nil, newError("protect failed")
	}

	// Apply sockopt

	if sockopt != nil {
		internet.ApplySockopt(sockopt, destination, uintptr(fd), ctx)
	}

	// SO_SNDTIMEO default is 75s
	syscall.SetsockoptTimeval(fd, syscall.SOL_SOCKET, syscall.SO_SNDTIMEO, &syscall.Timeval{Sec: 10})

	// Do Connect

	if destination.Network == v2rayNet.Network_TCP {
		sockaddr := &unix.SockaddrInet6{
			Port: int(destination.Port.Value()),
		}
		copy(sockaddr.Addr[:], destination.Address.IP().To16())

		err = unix.Connect(fd, sockaddr)
	} else {
		err = unix.Bind(fd, &unix.SockaddrInet6{})
	}

	if err != nil {
		syscall.Close(fd)
		return nil, err
	}

	// Get go file

	file := os.NewFile(uintptr(fd), "socket")
	if file == nil {
		return nil, newError("failed to connect to fd")
	}
	defer file.Close() // old fd will closed

	// Get go conn

	switch destination.Network {
	case v2rayNet.Network_UDP:
		pc, err := net.FilePacketConn(file)
		if err == nil {
			destAddr, err := net.ResolveUDPAddr("udp", destination.NetAddr())
			if err != nil {
				return nil, err
			}
			conn = &nekoutils.PacketConnWrapper{
				PacketConn: pc,
				Dest:       destAddr,
			}
		}
	default:
		conn, err = net.FileConn(file)
	}

	if err != nil {
		return nil, err
	}

	return conn, nil
}

func getFd(network v2rayNet.Network) (fd int, err error) {
	switch network {
	case v2rayNet.Network_TCP:
		fd, err = unix.Socket(unix.AF_INET6, unix.SOCK_STREAM, unix.IPPROTO_TCP)
	case v2rayNet.Network_UDP:
		fd, err = unix.Socket(unix.AF_INET6, unix.SOCK_DGRAM, unix.IPPROTO_UDP)
	default:
		err = fmt.Errorf("unknow network")
	}
	return
}
