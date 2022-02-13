package gvisor

import (
	"fmt"
	"github.com/sirupsen/logrus"
	v2rayNet "github.com/v2fly/v2ray-core/v5/common/net"
	"gvisor.dev/gvisor/pkg/tcpip/adapters/gonet"
	"gvisor.dev/gvisor/pkg/tcpip/stack"
	"gvisor.dev/gvisor/pkg/tcpip/transport/tcp"
	"gvisor.dev/gvisor/pkg/waiter"
	"libcore/tun"
	"net"
	"strconv"
)

func gTcpHandler(s *stack.Stack, handler tun.Handler) {
	forwarder := tcp.NewForwarder(s, 0, 2<<10, func(request *tcp.ForwarderRequest) {
		id := request.ID()
		waitQueue := new(waiter.Queue)
		endpoint, errT := request.CreateEndpoint(waitQueue)
		if errT != nil {
			// prevent potential half-open TCP connection leak.
			request.Complete(true)
			return
		}
		request.Complete(false)
		srcAddr := net.JoinHostPort(id.RemoteAddress.String(), strconv.Itoa(int(id.RemotePort)))
		src, err := v2rayNet.ParseDestination(fmt.Sprint("tcp:", srcAddr))
		if err != nil {
			logrus.Warn("[TCP] parse source address ", srcAddr, " failed: ", err)
			return
		}
		dstAddr := net.JoinHostPort(id.LocalAddress.String(), strconv.Itoa(int(id.LocalPort)))
		dst, err := v2rayNet.ParseDestination(fmt.Sprint("tcp:", dstAddr))
		if err != nil {
			logrus.Warn("[TCP] parse destination address ", dstAddr, " failed: ", err)
			return
		}
		go handler.NewConnection(src, dst, gonet.NewTCPConn(waitQueue, endpoint))
	})
	s.SetTransportProtocolHandler(tcp.ProtocolNumber, forwarder.HandlePacket)
}
