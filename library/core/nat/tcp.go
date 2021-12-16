package nat

import (
	"net"
	"time"

	"github.com/Dreamacro/clash/common/cache"
	v2rayNet "github.com/v2fly/v2ray-core/v4/common/net"
	"gvisor.dev/gvisor/pkg/tcpip"
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
	if tun.ipv6Mode == 0 {
		network = "tcp4"
		address.IP = vlanClient4
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
		t.sessions.SetWithExpire(key, session, time.Now().Add(time.Second*10))
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
				t.Close()
				t.tun.errorHandler(e.String())
				return
			}
		}
	}
}

func (t *tcpForwarder) process(hdr *TCPHeader) error {
	sourceAddress := hdr.SourceAddress()
	destinationAddress := hdr.DestinationAddress()
	sourcePort := hdr.SourcePort()
	destinationPort := hdr.DestinationPort()

	var session *peerValue

	if sourcePort != t.port {

		key := peerKey{destinationAddress, sourcePort}
		iSession, ok := t.sessions.Get(key)
		if ok {
			session = iSession.(*peerValue)
		} else {
			/*if hdr.Flags() != header.TCPFlagSyn {
				return newError("unable to create session: not tcp syn flag")
			}*/
			session = &peerValue{sourceAddress, destinationPort}
			t.sessions.Set(key, session)
		}

		hdr.SetSourceAddress(destinationAddress)
		hdr.SetDestinationAddress(hdr.Device())
		hdr.SetDestinationPort(t.port)
		hdr.UpdateChecksum()

		// destinationAddress:sourcePort -> device:tcpServerPort

	} else {

		// device:tcpServerPort -> destinationAddress:sourcePort
		iSession, ok := t.sessions.Get(peerKey{destinationAddress, destinationPort})
		if ok {
			session = iSession.(*peerValue)
		} else {
			return newError("unknown tcp session with source port ", destinationPort, " to destination address ", destinationAddress)
		}
		hdr.SetSourceAddress(destinationAddress)
		hdr.SetSourcePort(session.destinationPort)
		hdr.SetDestinationAddress(session.sourceAddress)
		hdr.UpdateChecksum()
	}

	return nil
}

func (t *tcpForwarder) Close() error {
	return t.listener.Close()
}
