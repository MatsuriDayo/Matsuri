package tun

import (
	"net"

	v2rayNet "github.com/v2fly/v2ray-core/v5/common/net"
)

type Tun interface {
	Stop() //Should stop goroutines but not close the tun fd
}

// For UDP downlink
type WriteBackFunc func([]byte, *net.UDPAddr) (int, error)

// For UDP upLink
type UDPPacket struct {
	Src       *net.UDPAddr
	Dst       *net.UDPAddr
	Data      []byte
	Put       func() // put cache for a packet
	PutHeader func() // put cache for a connection(header)
	WriteBack WriteBackFunc
}

type Handler interface {
	NewConnection(source v2rayNet.Destination, destination v2rayNet.Destination, conn net.Conn)
	HandlePacket(p *UDPPacket) // Handle Uplink UDP Packet
}

const PRIVATE_VLAN4_CLIENT = "172.19.0.1"
const PRIVATE_VLAN4_ROUTER = "172.19.0.2"
const PRIVATE_VLAN6_CLIENT = "fdfe:dcba:9876::1"
const PRIVATE_VLAN6_ROUTER = "fdfe:dcba:9876::2"

var PRIVATE_VLAN4_CLIENT_IP = net.ParseIP(PRIVATE_VLAN4_CLIENT).To4()
var PRIVATE_VLAN6_CLIENT_IP = net.ParseIP(PRIVATE_VLAN6_CLIENT)
var PRIVATE_VLAN4_ROUTER_IP = net.ParseIP(PRIVATE_VLAN4_ROUTER).To4()
var PRIVATE_VLAN6_ROUTER_IP = net.ParseIP(PRIVATE_VLAN6_ROUTER)

var FAKEDNS_VLAN4_CLIENT_IPNET = net.IPNet{IP: net.ParseIP("198.18.0.0").To4(), Mask: net.CIDRMask(15, 32)}
