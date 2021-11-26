package core

import (
	"net"
)

type TCPConnHandler interface {
	Handle(conn net.Conn) error
}

type UDPConnHandler interface {
	ReceiveTo(conn UDPConn, data []byte, addr *net.UDPAddr) error
}

var tcpConnHandler TCPConnHandler
var udpConnHandler UDPConnHandler

func RegisterTCPConnHandler(h TCPConnHandler) {
	tcpConnHandler = h
}

func RegisterUDPConnHandler(h UDPConnHandler) {
	udpConnHandler = h
}
