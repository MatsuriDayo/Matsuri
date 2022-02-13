package tun

import (
	"io"
	"net"

	v2rayNet "github.com/v2fly/v2ray-core/v5/common/net"
)

type Tun interface {
	io.Closer
}

type Handler interface {
	NewConnection(source v2rayNet.Destination, destination v2rayNet.Destination, conn net.Conn)
	NewPacket(source v2rayNet.Destination, destination v2rayNet.Destination, data []byte, writeBack func([]byte, *net.UDPAddr) (int, error), closer io.Closer)
}

const PRIVATE_VLAN4_CLIENT = "172.19.0.1"
const PRIVATE_VLAN4_ROUTER = "172.19.0.2"
const PRIVATE_VLAN6_CLIENT = "fdfe:dcba:9876::1"
const PRIVATE_VLAN6_ROUTER = "fdfe:dcba:9876::2"

var PRIVATE_VLAN4_CLIENT_IP = net.ParseIP(PRIVATE_VLAN4_CLIENT).To4()
var PRIVATE_VLAN6_CLIENT_IP = net.ParseIP(PRIVATE_VLAN6_CLIENT)
var PRIVATE_VLAN4_ROUTER_IP = net.ParseIP(PRIVATE_VLAN4_ROUTER).To4()
var PRIVATE_VLAN6_ROUTER_IP = net.ParseIP(PRIVATE_VLAN6_ROUTER)
