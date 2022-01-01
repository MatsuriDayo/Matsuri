package nat

import (
	"io"
	"libcore/tun"
	"net"

	"libcore/tun/tun2socket/tcpip"
)

func Start(
	device io.ReadWriter,
) (*TCP, *UDP, error) {
	listener, err := net.ListenTCP("tcp", &net.TCPAddr{IP: net.IPv6zero, Port: 0})
	if err != nil {
		return nil, nil, err
	}

	tab := newTable()
	udp := &UDP{
		device: device,
		buf:    [65535]byte{},
	}
	tcp := &TCP{
		listener: listener,
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

			var ip tcpip.IPPacket
			var gateway net.IP
			var portal net.IP

			if tcpip.IsIPv4(raw) {
				gateway = tun.PRIVATE_VLAN4_CLIENT_IP
				portal = tun.PRIVATE_VLAN4_ROUTER_IP
				ip = tcpip.IPv4Packet(raw)
			} else if tcpip.IsIPv6(raw) {
				gateway = tun.PRIVATE_VLAN6_CLIENT_IP
				portal = tun.PRIVATE_VLAN6_ROUTER_IP
				ip = tcpip.IPv6Packet(raw)
			} else {
				continue
			}

			if !ip.Valid() {
				continue
			}

			if ip.Flags()&tcpip.FlagMoreFragment != 0 {
				continue
			}

			if ip.FragmentOffset() != 0 {
				continue
			}

			if !ip.DestinationIP().IsGlobalUnicast() {
				continue
			}

			switch ip.Protocol() {
			case tcpip.TCP:
				t := tcpip.TCPPacket(ip.Payload())
				if !t.Valid() {
					continue
				}

				// nat 发出的包，改写发回用户
				if ip.DestinationIP().Equal(portal) {
					if ip.SourceIP().Equal(gateway) && t.SourcePort() == gatewayPort {
						tup := tab.tupleOf(t.DestinationPort())
						if tup == zeroTuple {
							continue
						}

						ip.SetSourceIP(net.IP(tup.DestinationIP[:]))
						ip.SetDestinationIP(net.IP(tup.SourceIP[:]))
						t.SetDestinationPort(tup.SourcePort)
						t.SetSourcePort(tup.DestinationPort)

						ip.ResetChecksum()
						t.ResetChecksum(ip.PseudoSum())

						_, _ = device.Write(raw)
					}
				} else { // 用户发出的包，改写发到 nat
					var sip16 [16]byte
					var dip16 [16]byte

					sip := ip.SourceIP().To16()
					dip := ip.DestinationIP().To16()
					copy(sip16[:], sip)
					copy(dip16[:], dip)

					tup := tuple{
						SourceIP:        sip16,
						DestinationIP:   dip16,
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

				source := ip.SourceIP()
				destination := ip.DestinationIP()
				ip.SetSourceIP(destination)
				ip.SetDestinationIP(source)

				ip.ResetChecksum()
				i.ResetChecksum()

				_, _ = device.Write(raw)
			}
		}
	}()

	return tcp, udp, nil
}
