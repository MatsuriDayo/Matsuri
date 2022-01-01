package system

import (
	"gvisor.dev/gvisor/pkg/tcpip"
)

type peerKey struct {
	destinationAddress tcpip.Address
	sourcePort         uint16
}

type peerValue struct {
	sourceAddress   tcpip.Address
	destinationPort uint16
}
