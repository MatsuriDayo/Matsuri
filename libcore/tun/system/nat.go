package system

import (
	"libcore/tun"
	"net"
	"runtime/debug"

	"github.com/sirupsen/logrus"
	v2rayNet "github.com/v2fly/v2ray-core/v5/common/net"
	"gvisor.dev/gvisor/pkg/tcpip"
	"gvisor.dev/gvisor/pkg/tcpip/buffer"
	"gvisor.dev/gvisor/pkg/tcpip/header"
	"gvisor.dev/gvisor/pkg/tcpip/header/parse"
	"gvisor.dev/gvisor/pkg/tcpip/stack"
)

var _ tun.Tun = (*SystemTun)(nil)

var (
	vlanClient4 = net.ParseIP(tun.PRIVATE_VLAN4_CLIENT).To4()
	vlanClient6 = net.ParseIP(tun.PRIVATE_VLAN6_CLIENT)
)

type SystemTun struct {
	dispatcher   *readVDispatcher
	dev          int32
	mtu          int32
	handler      tun.Handler
	ipv6Mode     int32
	tcpForwarder *tcpForwarder
	errorHandler func(err string)
}

func New(dev int32, mtu int32, handler tun.Handler, ipv6Mode int32, errorHandler func(err string)) (*SystemTun, error) {
	t := &SystemTun{
		dev:          dev,
		mtu:          mtu,
		handler:      handler,
		ipv6Mode:     ipv6Mode,
		errorHandler: errorHandler,
	}
	dispatcher, err := newReadVDispatcher(int(dev), t)
	if err != nil {
		return nil, err
	}
	go dispatcher.dispatchLoop()
	t.dispatcher = dispatcher

	tcpServer, err := newTcpForwarder(t)
	if err != nil {
		return nil, err
	}
	go tcpServer.dispatchLoop()
	t.tcpForwarder = tcpServer

	return t, nil
}

func (n *SystemTun) deliverPacket(pkt *stack.PacketBuffer) {
	var ipVersion int
	if ihl, ok := pkt.Data().PullUp(1); ok {
		ipVersion = header.IPVersion(ihl)
	} else {
		return
	}

	log := "packet: "

	var ipHeader IPHeader
	switch ipVersion {
	case header.IPv4Version:
		if !parse.IPv4(pkt) {
			return
		}
		ipHeader = &IPv4Header{pkt, header.IPv4(pkt.NetworkHeader().View())}
		log += "ipv4: "
	case header.IPv6Version:
		proto, _, _, _, ok := parse.IPv6(pkt)
		if !ok {
			return
		}
		ipHeader = &IPv6Header{pkt, proto, header.IPv6(pkt.NetworkHeader().View())}
		log += "ipv6: "
	default:
		return
	}

	switch ipHeader.Protocol() {
	case header.TCPProtocolNumber:
		log += "tcp: "
		if !parse.TCP(pkt) {
			newError(log, "unable to parse").AtWarning().WriteToLog()
			return
		}
		if err := n.tcpForwarder.process(&TCPHeader{ipHeader, header.TCP(pkt.TransportHeader().View())}); err != nil {
			newError(log, "process failed").Base(err).AtWarning().WriteToLog()
			return
		}
		n.dispatcher.writePacket(pkt)
	case header.UDPProtocolNumber:
		log += "udp: "
		if !parse.UDP(pkt) {
			newError(log, "unable to parse").AtWarning().WriteToLog()
			return
		}
		n.processUDP(&UDPHeader{ipHeader, header.UDP(pkt.TransportHeader().View())})
	case header.ICMPv4ProtocolNumber:
		log += "icmp4: "
		if !parse.ICMPv4(pkt) {
			newError(log, "unable to parse").AtWarning().WriteToLog()
			return
		}
		n.processICMPv4(&ICMPv4Header{ipHeader, header.ICMPv4(pkt.TransportHeader().View())})
	case header.ICMPv6ProtocolNumber:
		log += "icmp6: "
		if !parse.ICMPv6(pkt) {
			newError(log, "unable to parse").AtWarning().WriteToLog()
			return
		}
		n.processICMPv6(&ICMPv6Header{ipHeader, header.ICMPv6(pkt.TransportHeader().View())})
	}
}

func (n *SystemTun) processUDP(hdr *UDPHeader) {
	sourceAddress := hdr.SourceAddress()
	destinationAddress := hdr.DestinationAddress()
	sourcePort := hdr.SourcePort()
	destinationPort := hdr.DestinationPort()

	source := v2rayNet.Destination{
		Address: v2rayNet.IPAddress([]byte(sourceAddress)),
		Port:    v2rayNet.Port(sourcePort),
		Network: v2rayNet.Network_UDP,
	}
	destination := v2rayNet.Destination{
		Address: v2rayNet.IPAddress([]byte(destinationAddress)),
		Port:    v2rayNet.Port(destinationPort),
		Network: v2rayNet.Network_UDP,
	}

	hdr.Packet().IncRef()
	hdr.SetDestinationAddress(sourceAddress)
	hdr.SetDestinationPort(sourcePort)

	data := hdr.Packet().Data().ExtractVV()
	go func() {
		defer func() {
			if r := recover(); r != nil {
				logrus.Warningln("processUDP panic", r)
				logrus.Warningln(string(debug.Stack()))
			}
		}()

		n.handler.NewPacket(source, destination, data.ToView(), func(bytes []byte, addr *v2rayNet.UDPAddr) (int, error) {
			ipData := stack.PayloadSince(hdr.Packet().NetworkHeader())
			udpData := stack.PayloadSince(hdr.Packet().TransportHeader())[:header.UDPMinimumSize]

			var newSourceAddress tcpip.Address
			var newSourcePort uint16

			if addr != nil {
				newSourceAddress = tcpip.Address(addr.IP)
				newSourcePort = uint16(addr.Port)
			} else {
				newSourceAddress = destinationAddress
				newSourcePort = destinationPort
			}

			switch hdr.IPHeader.(type) {
			case *IPv4Header:
				ipHdr := header.IPv4(ipData)
				ipData = ipData[:ipHdr.HeaderLength()]
				ipHdr.SetSourceAddress(newSourceAddress)
				ipHdr.SetTotalLength(uint16(len(ipData) + len(udpData) + len(bytes)))
				ipHdr.SetChecksum(0)
				ipHdr.SetChecksum(^ipHdr.CalculateChecksum())
			case *IPv6Header:
				ipHdr := header.IPv6(ipData)
				ipData = ipData[:ipData.Size()-int(ipHdr.PayloadLength())]
				ipHdr.SetSourceAddress(newSourceAddress)
				ipHdr.SetPayloadLength(uint16(len(udpData) + len(bytes)))
			}

			udpHdr := header.UDP(udpData)
			udpHdr.SetSourcePort(newSourcePort)
			udpHdr.SetLength(uint16(len(udpData) + len(bytes)))
			udpHdr.SetChecksum(0)
			udpHdr.SetChecksum(^udpHdr.CalculateChecksum(header.Checksum(bytes, header.PseudoHeaderChecksum(header.UDPProtocolNumber, newSourceAddress, sourceAddress, uint16(len(udpData)+len(bytes))))))

			replyVV := buffer.VectorisedView{}
			replyVV.AppendView(ipData)
			replyVV.AppendView(udpData)
			replyVV.AppendView(bytes)

			if err := n.dispatcher.writeRawPacket(replyVV); err != nil {
				return 0, newError(err.String())
			}

			return len(bytes), nil
		}, hdr)
	}()
}

func (n *SystemTun) processICMPv4(hdr *ICMPv4Header) {
	if hdr.Type() != header.ICMPv4Echo || hdr.Code() != header.ICMPv4UnusedCode {
		return
	}

	hdr.SetType(header.ICMPv4EchoReply)
	sourceAddress := hdr.SourceAddress()
	hdr.SetSourceAddress(hdr.DestinationAddress())
	hdr.SetDestinationAddress(sourceAddress)
	hdr.UpdateChecksum()

	n.dispatcher.writePacket(hdr.Packet())
}

func (n *SystemTun) processICMPv6(hdr *ICMPv6Header) {
	if hdr.Type() != header.ICMPv6EchoRequest {
		return
	}
	hdr.SetType(header.ICMPv6EchoReply)
	sourceAddress := hdr.SourceAddress()
	hdr.SetSourceAddress(hdr.DestinationAddress())
	hdr.SetDestinationAddress(sourceAddress)
	hdr.UpdateChecksum()

	n.dispatcher.writePacket(hdr.Packet())
}

func (n *SystemTun) Stop() {
	n.dispatcher.stop()
	n.tcpForwarder.Close()
}
