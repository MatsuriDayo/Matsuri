package nat

import (
	"encoding/binary"
	"net"
	"syscall"
	"time"
)

type TCP struct {
	listener *net.TCPListener
	portal   net.IP
	table    *table
}

type conn struct {
	net.Conn

	tuple tuple
}

func (t *TCP) Accept() (net.Conn, error) {
	c, err := t.listener.AcceptTCP()
	if err != nil {
		return nil, err
	}

	addr := c.RemoteAddr().(*net.TCPAddr)
	tup := t.table.tupleOf(uint16(addr.Port))
	if !addr.IP.Equal(t.portal) || tup == zeroTuple {
		_ = c.Close()

		return nil, net.InvalidAddrError("unknown remote addr")
	}

	_ = c.SetKeepAlive(false)

	sys, err := c.SyscallConn()
	if err == nil {
		_ = sys.Control(func(fd uintptr) {
			_ = syscall.SetsockoptInt(int(fd), syscall.SOL_SOCKET, syscall.SO_NO_CHECK, 1)
		})
	}

	return &conn{
		Conn:  c,
		tuple: tup,
	}, nil
}

func (t *TCP) Close() error {
	return t.listener.Close()
}

func (t *TCP) Addr() net.Addr {
	return t.listener.Addr()
}

func (t *TCP) SetDeadline(time time.Time) error {
	return t.listener.SetDeadline(time)
}

func (c *conn) LocalAddr() net.Addr {
	ip := make(net.IP, 4)

	binary.LittleEndian.PutUint32(ip, c.tuple.SourceIP)

	return &net.TCPAddr{
		IP:   ip,
		Port: int(c.tuple.SourcePort),
	}
}

func (c *conn) RemoteAddr() net.Addr {
	ip := make(net.IP, 4)

	binary.LittleEndian.PutUint32(ip, c.tuple.DestinationIP)

	return &net.TCPAddr{
		IP:   ip,
		Port: int(c.tuple.DestinationPort),
	}
}
