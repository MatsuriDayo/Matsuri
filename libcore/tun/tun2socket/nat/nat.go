package nat

import (
	"encoding/binary"
	"io"
	"net"

	"libcore/tun/tun2socket/tcpip"
)

func Start(
	device io.ReadWriter,
	gateway net.IP,
	portal net.IP,
) (*TCP, *UDP, error) {
	portal = portal.To4()
	gateway = gateway.To4()
	if portal == nil || gateway == nil {
		return nil, nil, net.InvalidAddrError("only ipv4 supported")
	}

	listener, err := net.ListenTCP("tcp4", &net.TCPAddr{IP: net.IPv4zero, Port: 0})
	if err != nil {
		return nil, nil, err
	}

	tab := newTable()
	udp := &UDP{
		calls:  map[*call]struct{}{},
		device: device,
		buf:    [65535]byte{},
	}
	tcp := &TCP{
		listener: listener,
		portal:   portal,
		table:    tab,
	}

	gatewayPort := uint16(listener.Addr().(*net.TCPAddr).Port)

	go func() {
		defer tcp.Close()
		defer udp.Close()

		buf := make([]byte, 65535)

		for {
			n, err := device.Read(buf)
			if err != nil {
				return
			}

			raw := buf[:n]

			if !tcpip.IsIPv4(raw) {
				continue
			}

			ip := tcpip.IPv4Packet(raw)
			if !ip.Valid() {
				continue
			}

			if ip.Flags()&tcpip.FlagMoreFragment != 0 {
				continue
			}

			if ip.Offset() != 0 {
				continue
			}

			switch ip.Protocol() {
			case tcpip.TCP:
				t := tcpip.TCPPacket(ip.Payload())
				if !t.Valid() {
					continue
				}

				if ip.DestinationIP().Equal(portal) {
					if ip.SourceIP().Equal(gateway) && t.SourcePort() == gatewayPort {
						tup := tab.tupleOf(t.DestinationPort())
						if tup == zeroTuple {
							continue
						}

						src := net.IP{0, 0, 0, 0}
						dst := net.IP{0, 0, 0, 0}
						binary.LittleEndian.PutUint32(src, tup.SourceIP)
						binary.LittleEndian.PutUint32(dst, tup.DestinationIP)
						ip.SetSourceIP(dst)
						ip.SetDestinationIP(src)
						t.SetDestinationPort(tup.SourcePort)
						t.SetSourcePort(tup.DestinationPort)

						ip.ResetChecksum()
						t.ResetChecksum(ip.PseudoSum())

						_, _ = device.Write(raw)
					}
				} else {
					tup := tuple{
						SourceIP:        binary.LittleEndian.Uint32(ip.SourceIP()),
						DestinationIP:   binary.LittleEndian.Uint32(ip.DestinationIP()),
						SourcePort:      t.SourcePort(),
						DestinationPort: t.DestinationPort(),
					}

					port := tab.portOf(tup)
					if port == 0 {
						if t.Flags() != tcpip.TCPSyn {
							continue
						}

						port = tab.newConn(tup)
					}

					ip.SetSourceIP(portal)
					ip.SetDestinationIP(gateway)
					t.SetSourcePort(port)
					t.SetDestinationPort(gatewayPort)

					ip.ResetChecksum()
					t.ResetChecksum(ip.PseudoSum())

					_, _ = device.Write(raw)
				}
			case tcpip.UDP:
				u := tcpip.UDPPacket(ip.Payload())
				if !u.Valid() {
					continue
				}

				udp.handleUDPPacket(ip, u)
			case tcpip.ICMP:
				i := tcpip.ICMPPacket(ip.Payload())

				if i.Type() != tcpip.ICMPTypePingRequest || i.Code() != 0 {
					continue
				}

				i.SetType(tcpip.ICMPTypePingResponse)

				source := binary.LittleEndian.Uint32(ip.SourceIP())
				destination := binary.LittleEndian.Uint32(ip.DestinationIP())
				binary.LittleEndian.PutUint32(ip.SourceIP(), destination)
				binary.LittleEndian.PutUint32(ip.DestinationIP(), source)

				ip.ResetChecksum()
				i.ResetChecksum()
			}
		}
	}()

	return tcp, udp, nil
}
