package system

import (
	"errors"
	"net"
	"time"

	"libcore/comm"

	"github.com/Dreamacro/clash/common/cache"
	"github.com/sirupsen/logrus"
	v2rayNet "github.com/v2fly/v2ray-core/v5/common/net"
	"gvisor.dev/gvisor/pkg/tcpip"
	"gvisor.dev/gvisor/pkg/tcpip/header"
)

type tcpForwarder struct {
	tun      *SystemTun
	port     uint16
	listener *net.TCPListener
	sessions *cache.LruCache
}

func newTcpForwarder(tun *SystemTun) (*tcpForwarder, error) {
	var network string
	address := &net.TCPAddr{}
	if tun.ipv6Mode == comm.IPv6Disable {
		network = "tcp4"
		address.IP = net.IP(vlanClient4)
	} else {
		network = "tcp"
		address.IP = net.IPv6zero
	}
	listener, err := net.ListenTCP(network, address)
	if err != nil {
		return nil, newError("failed to create tcp forwarder at ", address.IP).Base(err)
	}
	addr := listener.Addr().(*net.TCPAddr)
	port := uint16(addr.Port)
	newError("tcp forwarder started at ", addr).AtDebug().WriteToLog()
	return &tcpForwarder{tun, port, listener, cache.NewLRUCache(
		cache.WithAge(300),
		cache.WithUpdateAgeOnGet(),
	)}, nil
}

func (t *tcpForwarder) dispatch() (bool, error) {
	conn, err := t.listener.AcceptTCP()
	if err != nil {
		return true, err
	}
	addr := conn.RemoteAddr().(*net.TCPAddr)
	if ip4 := addr.IP.To4(); ip4 != nil {
		addr.IP = ip4
	}
	key := peerKey{tcpip.Address(addr.IP), uint16(addr.Port)}
	var session *peerValue
	iSession, ok := t.sessions.Get(peerKey{key.destinationAddress, key.sourcePort})
	if ok {
		session = iSession.(*peerValue)
	} else {
		conn.Close()
		return false, newError("dropped unknown tcp session with source port ", key.sourcePort, " to destination address ", key.destinationAddress)
	}

	source := v2rayNet.Destination{
		Address: v2rayNet.IPAddress([]byte(session.sourceAddress)),
		Port:    v2rayNet.Port(key.sourcePort),
		Network: v2rayNet.Network_TCP,
	}
	destination := v2rayNet.Destination{
		Address: v2rayNet.IPAddress([]byte(key.destinationAddress)),
		Port:    v2rayNet.Port(session.destinationPort),
		Network: v2rayNet.Network_TCP,
	}

	go func() {
		t.tun.handler.NewConnection(source, destination, conn)
		time.Sleep(time.Second * 5)
		t.sessions.Delete(key)
	}()

	return false, nil
}

func (t *tcpForwarder) dispatchLoop() {
	for {
		stop, err := t.dispatch()
		if err != nil {
			e := newError("dispatch tcp conn failed").Base(err)
			e.WriteToLog()
			if stop {
				if !errors.Is(err, net.ErrClosed) {
					t.Close()
					t.tun.errorHandler(e.String())
				}
				return
			}
		}
	}
}

func (t *tcpForwarder) processIPv4(ipHdr header.IPv4, tcpHdr header.TCP) {
	sourceAddress := ipHdr.SourceAddress()
	destinationAddress := ipHdr.DestinationAddress()
	sourcePort := tcpHdr.SourcePort()
	destinationPort := tcpHdr.DestinationPort()

	var session *peerValue

	if sourcePort != t.port {

		key := peerKey{destinationAddress, sourcePort}
		iSession, ok := t.sessions.Get(key)
		if ok {
			session = iSession.(*peerValue)
		} else {
			session = &peerValue{sourceAddress, destinationPort}
			t.sessions.Set(key, session)
		}

		ipHdr.SetSourceAddress(destinationAddress)
		ipHdr.SetDestinationAddress(vlanClient4)
		tcpHdr.SetDestinationPort(t.port)

	} else {

		iSession, ok := t.sessions.Get(peerKey{destinationAddress, destinationPort})
		if ok {
			session = iSession.(*peerValue)
		} else {
			logrus.Warn("unknown tcp session with source port ", destinationPort, " to destination address ", destinationAddress)
			return
		}
		ipHdr.SetSourceAddress(destinationAddress)
		tcpHdr.SetSourcePort(session.destinationPort)
		ipHdr.SetDestinationAddress(session.sourceAddress)
	}

	ipHdr.SetChecksum(0)
	ipHdr.SetChecksum(^ipHdr.CalculateChecksum())
	tcpHdr.SetChecksum(0)
	tcpHdr.SetChecksum(^tcpHdr.CalculateChecksum(header.ChecksumCombine(
		header.PseudoHeaderChecksum(header.TCPProtocolNumber, ipHdr.SourceAddress(), ipHdr.DestinationAddress(), uint16(len(tcpHdr))),
		header.Checksum(tcpHdr.Payload(), 0),
	)))

	t.tun.writeBuffer(ipHdr)
}

func (t *tcpForwarder) processIPv6(ipHdr header.IPv6, tcpHdr header.TCP) {
	sourceAddress := ipHdr.SourceAddress()
	destinationAddress := ipHdr.DestinationAddress()
	sourcePort := tcpHdr.SourcePort()
	destinationPort := tcpHdr.DestinationPort()

	var session *peerValue

	if sourcePort != t.port {

		key := peerKey{destinationAddress, sourcePort}
		iSession, ok := t.sessions.Get(key)
		if ok {
			session = iSession.(*peerValue)
		} else {
			session = &peerValue{sourceAddress, destinationPort}
			t.sessions.Set(key, session)
		}

		ipHdr.SetSourceAddress(destinationAddress)
		ipHdr.SetDestinationAddress(vlanClient6)
		tcpHdr.SetDestinationPort(t.port)

	} else {

		iSession, ok := t.sessions.Get(peerKey{destinationAddress, destinationPort})
		if ok {
			session = iSession.(*peerValue)
		} else {
			logrus.Warn("unknown tcp session with source port ", destinationPort, " to destination address ", destinationAddress)
			return
		}

		ipHdr.SetSourceAddress(destinationAddress)
		tcpHdr.SetSourcePort(session.destinationPort)
		ipHdr.SetDestinationAddress(session.sourceAddress)
	}

	tcpHdr.SetChecksum(0)
	tcpHdr.SetChecksum(^tcpHdr.CalculateChecksum(header.ChecksumCombine(
		header.PseudoHeaderChecksum(header.TCPProtocolNumber, ipHdr.SourceAddress(), ipHdr.DestinationAddress(), uint16(len(tcpHdr))),
		header.Checksum(tcpHdr.Payload(), 0),
	)))

	t.tun.writeBuffer(ipHdr)
}

func (t *tcpForwarder) Close() error {
	return t.listener.Close()
}
