package tcpip

import (
	"encoding/binary"
	"net"
)

type IPProtocol = byte

// IPProtocol type
const (
	ICMP IPProtocol = 0x01
	TCP             = 0x06
	UDP             = 0x11
)

const (
	FlagDontFragment = 1 << 1
	FlagMoreFragment = 1 << 2
)

const IPv4HeaderSize = 20
const IPv6HeaderSize = 40

type IPPacket interface {
	Valid() bool
	Flags() byte
	SetFlags(flags byte)
	FragmentOffset() uint16
	Protocol() IPProtocol
	Payload() []byte
	SetSourceIP(ip net.IP)
	SourceIP() net.IP
	DestinationIP() net.IP
	SetDestinationIP(ip net.IP)
	PseudoSum() uint32
	ResetChecksum()
	SetTotalLength(length uint16)
	SetHeaderLen(length uint16)
	SetFragmentOffset(offset uint32)
	SetTypeOfService(tos byte)
	SetIdentification(id uint16)
	SetProtocol(protocol IPProtocol)
	SetTimeToLive(ttl byte)
	TotalLen() uint16
}

// IPv6

type IPv6Packet []byte

func (p IPv6Packet) TotalLen() uint16 {
	return uint16(len(p))
}

func (p IPv6Packet) Valid() bool {
	return len(p) >= IPv6HeaderSize && uint16(len(p)) >= p.TotalLen()
}

func (p IPv6Packet) Protocol() IPProtocol {
	return p[6]
}

func (p IPv6Packet) SetProtocol(protocol IPProtocol) {
	p[6] = protocol
}

func (p IPv6Packet) Payload() []byte {
	return p[40:]
}

func (p IPv6Packet) SourceIP() net.IP {
	return append(net.IP{}, p[8:24]...)
}

func (p IPv6Packet) SetSourceIP(ip net.IP) {
	copy(p[8:24], ip)
}

func (p IPv6Packet) DestinationIP() net.IP {
	return append(net.IP{}, p[24:40]...)
}

func (p IPv6Packet) SetDestinationIP(ip net.IP) {
	copy(p[24:40], ip)
}

func (p IPv6Packet) PseudoSum() uint32 {
	sum := Sum(p[8:40])
	sum += uint32(p.Protocol())
	sum += uint32(p.DataLen())
	return sum
}

func (p IPv6Packet) DataLen() uint16 {
	return binary.BigEndian.Uint16(p[4:])
}

func (p IPv6Packet) SetTimeToLive(ttl byte) {
	p[7] = ttl
}

//Deleted in IPv6

func (p IPv6Packet) SetHeaderLen(length uint16) {}

func (p IPv6Packet) SetTotalLength(length uint16) {}

func (p IPv6Packet) SetIdentification(id uint16) {}

func (p IPv6Packet) ResetChecksum() {}

func (p IPv6Packet) Flags() byte {
	return 0
}

func (p IPv6Packet) SetFlags(flags byte) {}

func (p IPv6Packet) FragmentOffset() uint16 {
	return 0
}

func (p IPv6Packet) SetFragmentOffset(offset uint32) {}

func (p IPv6Packet) SetTypeOfService(tos byte) {}

// IPv4

type IPv4Packet []byte

func (p IPv4Packet) TotalLen() uint16 {
	return binary.BigEndian.Uint16(p[2:])
}

func (p IPv4Packet) SetTotalLength(length uint16) {
	binary.BigEndian.PutUint16(p[2:], length)
}

func (p IPv4Packet) HeaderLen() uint16 {
	return uint16(p[0]&0xf) * 4
}

func (p IPv4Packet) SetHeaderLen(length uint16) {
	p[0] &= 0xF0
	p[0] |= byte(length / 4)
}

func (p IPv4Packet) TypeOfService() byte {
	return p[1]
}

func (p IPv4Packet) SetTypeOfService(tos byte) {
	p[1] = tos
}

func (p IPv4Packet) Identification() uint16 {
	return binary.BigEndian.Uint16(p[4:])
}

func (p IPv4Packet) SetIdentification(id uint16) {
	binary.BigEndian.PutUint16(p[4:], id)
}

func (p IPv4Packet) FragmentOffset() uint16 {
	return binary.BigEndian.Uint16([]byte{p[6] & 0x7, p[7]}) * 8
}

func (p IPv4Packet) SetFragmentOffset(offset uint32) {
	flags := p.Flags()
	binary.BigEndian.PutUint16(p[6:], uint16(offset/8))
	p.SetFlags(flags)
}

func (p IPv4Packet) DataLen() uint16 {
	return p.TotalLen() - p.HeaderLen()
}

func (p IPv4Packet) Payload() []byte {
	return p[p.HeaderLen():p.TotalLen()]
}

func (p IPv4Packet) Protocol() IPProtocol {
	return p[9]
}

func (p IPv4Packet) SetProtocol(protocol IPProtocol) {
	p[9] = protocol
}

func (p IPv4Packet) Flags() byte {
	return p[6] >> 5
}

func (p IPv4Packet) SetFlags(flags byte) {
	p[6] &= 0x1F
	p[6] |= flags << 5
}

func (p IPv4Packet) SourceIP() net.IP {
	return net.IP{p[12], p[13], p[14], p[15]}
}

func (p IPv4Packet) SetSourceIP(ip net.IP) {
	ip = ip.To4()
	if ip != nil {
		copy(p[12:16], ip)
	}
}

func (p IPv4Packet) DestinationIP() net.IP {
	return net.IP{p[16], p[17], p[18], p[19]}
}

func (p IPv4Packet) SetDestinationIP(ip net.IP) {
	ip = ip.To4()
	if ip != nil {
		copy(p[16:20], ip)
	}
}

func (p IPv4Packet) Checksum() uint16 {
	return binary.BigEndian.Uint16(p[10:])
}

func (p IPv4Packet) SetChecksum(sum [2]byte) {
	p[10] = sum[0]
	p[11] = sum[1]
}

func (p IPv4Packet) TimeToLive() byte {
	return p[8]
}

func (p IPv4Packet) SetTimeToLive(ttl byte) {
	p[8] = ttl
}

func (p IPv4Packet) ResetChecksum() {
	p.SetChecksum(zeroChecksum)
	p.SetChecksum(Checksum(0, p[:p.HeaderLen()]))
}

// PseudoSum for tcp checksum
func (p IPv4Packet) PseudoSum() uint32 {
	sum := Sum(p[12:20])
	sum += uint32(p.Protocol())
	sum += uint32(p.DataLen())
	return sum
}

func (p IPv4Packet) Valid() bool {
	return len(p) >= IPv4HeaderSize && uint16(len(p)) >= p.TotalLen()
}
