package nat

import (
	"gvisor.dev/gvisor/pkg/tcpip"
	"gvisor.dev/gvisor/pkg/tcpip/header"
	"gvisor.dev/gvisor/pkg/tcpip/stack"
)

type IPHeader interface {
	PayloadLength() uint16
	Device() tcpip.Address
	Packet() *stack.PacketBuffer
	Version() tcpip.NetworkProtocolNumber
	Protocol() tcpip.TransportProtocolNumber
	UpdateChecksum()
	header.Network
}

var (
	_ IPHeader = (*IPv4Header)(nil)
	_ IPHeader = (*IPv6Header)(nil)
)

type IPv4Header struct {
	pkt *stack.PacketBuffer
	header.IPv4
}

func (h *IPv4Header) Device() tcpip.Address {
	return tcpip.Address([]uint8{172, 19, 0, 1})
}

func (h *IPv4Header) Packet() *stack.PacketBuffer {
	return h.pkt
}

func (h *IPv4Header) Version() tcpip.NetworkProtocolNumber {
	return header.IPv4Version
}

func (h *IPv4Header) Protocol() tcpip.TransportProtocolNumber {
	return tcpip.TransportProtocolNumber(h.IPv4.Protocol())
}

func (h *IPv4Header) UpdateChecksum() {
	h.IPv4.SetChecksum(0)
	h.IPv4.SetChecksum(^h.IPv4.CalculateChecksum())
}

type IPv6Header struct {
	pkt *stack.PacketBuffer
	prt tcpip.TransportProtocolNumber
	header.IPv6
}

func (h *IPv6Header) Device() tcpip.Address {
	return tcpip.Address(vlanClient6)
}

func (h *IPv6Header) Packet() *stack.PacketBuffer {
	return h.pkt
}

func (h *IPv6Header) Version() tcpip.NetworkProtocolNumber {
	return header.IPv6Version
}

func (h *IPv6Header) Protocol() tcpip.TransportProtocolNumber {
	return h.prt
}

func (h *IPv6Header) UpdateChecksum() {
}

type TCPHeader struct {
	IPHeader
	header.TCP
}

func (h *TCPHeader) UpdateChecksum() {
	h.IPHeader.UpdateChecksum()
	h.TCP.SetChecksum(0)
	h.TCP.SetChecksum(^h.TCP.CalculateChecksum(header.ChecksumCombine(
		header.PseudoHeaderChecksum(header.TCPProtocolNumber, h.SourceAddress(), h.DestinationAddress(), h.PayloadLength()),
		h.Packet().Data().AsRange().Checksum(),
	)))
}

type UDPHeader struct {
	IPHeader
	header.UDP
}

func (h *UDPHeader) Close() error {
	h.Packet().DecRef()
	return nil
}

type ICMPv4Header struct {
	IPHeader
	header.ICMPv4
}

func (h *ICMPv4Header) UpdateChecksum() {
	h.IPHeader.UpdateChecksum()
	h.ICMPv4.SetChecksum(0)
	h.ICMPv4.SetChecksum(header.ICMPv4Checksum(h.ICMPv4, h.Packet().Data().AsRange().Checksum()))
}

type ICMPv6Header struct {
	IPHeader
	header.ICMPv6
}

func (h *ICMPv6Header) UpdateChecksum() {
	h.IPHeader.UpdateChecksum()
	h.ICMPv6.SetChecksum(0)
	payload := h.Packet().Data()
	h.ICMPv6.SetChecksum(header.ICMPv6Checksum(header.ICMPv6ChecksumParams{
		Header:      h.ICMPv6,
		Src:         h.SourceAddress(),
		Dst:         h.DestinationAddress(),
		PayloadCsum: payload.AsRange().Checksum(),
		PayloadLen:  payload.Size(),
	}))
}
