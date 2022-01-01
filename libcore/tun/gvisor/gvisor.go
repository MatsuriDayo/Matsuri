package gvisor

import (
	"errors"
	"io"
	"os"

	"libcore/comm"
	"libcore/tun"

	"github.com/sirupsen/logrus"
	"github.com/v2fly/v2ray-core/v5/common/buf"
	"gvisor.dev/gvisor/pkg/tcpip"
	"gvisor.dev/gvisor/pkg/tcpip/header"
	"gvisor.dev/gvisor/pkg/tcpip/link/sniffer"
	"gvisor.dev/gvisor/pkg/tcpip/network/ipv4"
	"gvisor.dev/gvisor/pkg/tcpip/network/ipv6"
	"gvisor.dev/gvisor/pkg/tcpip/stack"
	"gvisor.dev/gvisor/pkg/tcpip/transport/icmp"
	"gvisor.dev/gvisor/pkg/tcpip/transport/tcp"
	"gvisor.dev/gvisor/pkg/tcpip/transport/udp"
)

//go:generate go run ../errorgen

var _ tun.Tun = (*GVisor)(nil)

type GVisor struct {
	Endpoint stack.LinkEndpoint
	PcapFile *os.File
	Stack    *stack.Stack
}

func (t *GVisor) Stop() {
	t.Stack.Close()
	if t.PcapFile != nil {
		_ = t.PcapFile.Close()
	}
}

const DefaultNIC tcpip.NICID = 0x01

func New(dev int32, mtu int32, handler tun.Handler, nicId tcpip.NICID, pcap bool, pcapFile *os.File, snapLen uint32, ipv6Mode int32) (*GVisor, error) {
	var endpoint stack.LinkEndpoint
	endpoint, _ = newRwEndpoint(dev, mtu)
	if pcap {
		pcapEndpoint, err := sniffer.NewWithWriter(endpoint, &pcapFileWrapper{pcapFile}, snapLen)
		if err != nil {
			return nil, err
		}
		endpoint = pcapEndpoint
	}
	var o stack.Options
	switch ipv6Mode {
	case comm.IPv6Disable:
		o = stack.Options{
			NetworkProtocols: []stack.NetworkProtocolFactory{
				ipv4.NewProtocol,
			},
			TransportProtocols: []stack.TransportProtocolFactory{
				tcp.NewProtocol,
				udp.NewProtocol,
				icmp.NewProtocol4,
			},
		}
	case comm.IPv6Only:
		o = stack.Options{
			NetworkProtocols: []stack.NetworkProtocolFactory{
				ipv6.NewProtocol,
			},
			TransportProtocols: []stack.TransportProtocolFactory{
				tcp.NewProtocol,
				udp.NewProtocol,
				icmp.NewProtocol6,
			},
		}
	default:
		o = stack.Options{
			NetworkProtocols: []stack.NetworkProtocolFactory{
				ipv4.NewProtocol,
				ipv6.NewProtocol,
			},
			TransportProtocols: []stack.TransportProtocolFactory{
				tcp.NewProtocol,
				udp.NewProtocol,
				icmp.NewProtocol4,
				icmp.NewProtocol6,
			},
		}
	}
	s := stack.New(o)
	s.SetRouteTable([]tcpip.Route{
		{
			Destination: header.IPv4EmptySubnet,
			NIC:         nicId,
		},
		{
			Destination: header.IPv6EmptySubnet,
			NIC:         nicId,
		},
	})

	bufSize := buf.Size
	s.SetTransportProtocolOption(tcp.ProtocolNumber, &tcpip.TCPReceiveBufferSizeRangeOption{
		Min:     1,
		Default: bufSize,
		Max:     bufSize,
	})
	s.SetTransportProtocolOption(tcp.ProtocolNumber, &tcpip.TCPSendBufferSizeRangeOption{
		Min:     1,
		Default: bufSize,
		Max:     bufSize,
	})

	sOpt := tcpip.TCPSACKEnabled(true)
	s.SetTransportProtocolOption(tcp.ProtocolNumber, &sOpt)

	mOpt := tcpip.TCPModerateReceiveBufferOption(true)
	s.SetTransportProtocolOption(tcp.ProtocolNumber, &mOpt)

	gTcpHandler(s, handler)
	gUdpHandler(s, handler)
	gMust(s.CreateNIC(nicId, endpoint))
	gMust(s.SetSpoofing(nicId, true))
	gMust(s.SetPromiscuousMode(nicId, true))

	return &GVisor{endpoint, pcapFile, s}, nil
}

type pcapFileWrapper struct {
	io.Writer
}

func (w *pcapFileWrapper) Write(p []byte) (n int, err error) {
	n, err = w.Writer.Write(p)
	if err != nil {
		logrus.Debug("write pcap file failed: ", err)
	}
	return n, nil
}

func gMust(err tcpip.Error) {
	if err != nil {
		logrus.Panicln(err.String())
	}
}

func tcpipErr(err tcpip.Error) error {
	return errors.New(err.String())
}
