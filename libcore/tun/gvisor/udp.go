package gvisor

import (
	"fmt"
	"net"

	"libcore/tun"

	"gvisor.dev/gvisor/pkg/tcpip"
	"gvisor.dev/gvisor/pkg/tcpip/buffer"
	"gvisor.dev/gvisor/pkg/tcpip/header"
	"gvisor.dev/gvisor/pkg/tcpip/stack"
	"gvisor.dev/gvisor/pkg/tcpip/transport/udp"
)

func gUdpHandler(s *stack.Stack, handler tun.Handler) {
	s.SetTransportProtocolHandler(udp.ProtocolNumber, func(id stack.TransportEndpointID, buffer *stack.PacketBuffer) bool {
		// Ref: gVisor pkg/tcpip/transport/udp/endpoint.go HandlePacket
		udpHdr := header.UDP(buffer.TransportHeader().View())
		if int(udpHdr.Length()) > buffer.Data().Size()+header.UDPMinimumSize {
			// Malformed packet.
			return true
		}

		srcAddr := &net.UDPAddr{
			IP:   net.ParseIP(id.RemoteAddress.String()),
			Port: int(id.RemotePort),
		}
		dstAddr := &net.UDPAddr{
			IP:   net.ParseIP(id.LocalAddress.String()),
			Port: int(id.LocalPort),
		}

		data := buffer.Data().ExtractVV()
		packet := &gUdpPacket{
			s:        s,
			id:       &id,
			nicID:    buffer.NICID,
			netHdr:   buffer.Network(),
			netProto: buffer.NetworkProtocolNumber,
		}

		handler.HandlePacket(&tun.UDPPacket{
			Src:  srcAddr,
			Dst:  dstAddr,
			Data: data.ToView(),
			Put:  nil, // DecRef by dispatcher
			WriteBack: func(bytes []byte, addr *net.UDPAddr) (int, error) {
				return packet.WriteBack(bytes, addr)
			},
		})
		return true
	})
}

type gUdpPacket struct {
	s        *stack.Stack
	id       *stack.TransportEndpointID
	nicID    tcpip.NICID
	netHdr   header.Network
	netProto tcpip.NetworkProtocolNumber
}

func (p *gUdpPacket) WriteBack(b []byte, addr *net.UDPAddr) (int, error) {
	v := buffer.View(b)
	if len(v) > header.UDPMaximumPacketSize {
		// Payload can't possibly fit in a packet.
		return 0, fmt.Errorf("%s", &tcpip.ErrMessageTooLong{})
	}

	var (
		localAddress tcpip.Address
		localPort    uint16
	)

	if addr == nil {
		localAddress = p.netHdr.DestinationAddress()
		localPort = p.id.LocalPort
	} else {
		localAddress = tcpip.Address(addr.IP)
		localPort = uint16(addr.Port)
	}

	route, err := p.s.FindRoute(p.nicID, localAddress, p.netHdr.SourceAddress(), p.netProto, false /* multicastLoop */)
	if err != nil {
		return 0, fmt.Errorf("%#v find route: %s", p.id, err)
	}
	defer route.Release()

	data := v.ToVectorisedView()
	if err = gSendUDP(route, data, localPort, p.id.RemotePort); err != nil {
		return 0, fmt.Errorf("%v", err)
	}
	return data.Size(), nil
}

// gSendUDP sends a UDP segment via the provided network endpoint and under the
// provided identity.
func gSendUDP(r *stack.Route, data buffer.VectorisedView, localPort, remotePort uint16) tcpip.Error {
	pkt := stack.NewPacketBuffer(stack.PacketBufferOptions{
		ReserveHeaderBytes: header.UDPMinimumSize + int(r.MaxHeaderLength()),
		Data:               data,
	})
	defer pkt.DecRef()

	// Initialize the UDP header.
	udpHdr := header.UDP(pkt.TransportHeader().Push(header.UDPMinimumSize))
	pkt.TransportProtocolNumber = udp.ProtocolNumber

	length := uint16(pkt.Size())
	udpHdr.Encode(&header.UDPFields{
		SrcPort: localPort,
		DstPort: remotePort,
		Length:  length,
	})

	// Set the checksum field unless TX checksum offload is enabled.
	// On IPv4, UDP checksum is optional, and a zero value indicates the
	// transmitter skipped the checksum generation (RFC768).
	// On IPv6, UDP checksum is not optional (RFC2460 Section 8.1).
	if r.RequiresTXTransportChecksum() && r.NetProto() == header.IPv6ProtocolNumber {
		xsum := r.PseudoHeaderChecksum(udp.ProtocolNumber, length)
		for _, v := range data.Views() {
			xsum = header.Checksum(v, xsum)
		}
		udpHdr.SetChecksum(^udpHdr.CalculateChecksum(xsum))
	}

	ttl := r.DefaultTTL()

	if err := r.WritePacket(stack.NetworkHeaderParams{
		Protocol: udp.ProtocolNumber,
		TTL:      ttl,
		TOS:      0, /* default */
	}, pkt); err != nil {
		r.Stats().UDP.PacketSendErrors.Increment()
		return err
	}

	// Track count of packets sent.
	r.Stats().UDP.PacketsSent.Increment()
	return nil
}
