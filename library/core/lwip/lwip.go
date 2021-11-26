package lwip

import (
	"fmt"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/v2fly/v2ray-core/v4/common/bytespool"
	v2rayNet "github.com/v2fly/v2ray-core/v4/common/net"
	"libcore/lwip/core"
	"libcore/tun"
	"net"
	"os"
	"sync"
)

var _ tun.Tun = (*LwIP)(nil)

type LwIP struct {
	pool *sync.Pool

	Dev     *os.File
	Stack   core.LWIPStack
	Handler tun.Handler
}

func New(dev *os.File, mtu int32, handler tun.Handler) (*LwIP, error) {
	t := &LwIP{
		pool: bytespool.GetPool(mtu),

		Dev:     dev,
		Stack:   core.NewLWIPStack(),
		Handler: handler,
	}
	core.RegisterOutputFn(dev.Write)
	core.RegisterTCPConnHandler(t)
	core.RegisterUDPConnHandler(t)
	core.SetMtu(mtu)

	go func() {
		for {
			err := t.processPacket()
			if err != nil {
				logrus.Warn(err.Error())
				return
			}
		}
	}()

	return t, nil
}

func (l *LwIP) processPacket() error {
	buffer := l.pool.Get().([]byte)
	defer l.pool.Put(buffer)

	length, err := l.Dev.Read(buffer)
	if err != nil {
		logrus.Warnf("failed to read packet from TUN: %v", err)
		return nil
	}
	if length == 0 {
		return errors.New("read EOF from TUN")
	}

	_, err = l.Stack.Write(buffer)
	if err != nil {
		return errors.WithMessage(err, "failed to write packet to LWIP")
	}
	return nil
}

func (l *LwIP) Handle(conn net.Conn) error {
	srcAddr := conn.LocalAddr().String()
	src, err := v2rayNet.ParseDestination(fmt.Sprint("tcp:", srcAddr))
	if err != nil {
		logrus.Warn("[TCP] parse source address ", srcAddr, " failed: ", err)
		return err
	}
	dstAddr := conn.RemoteAddr().String()
	dst, err := v2rayNet.ParseDestination(fmt.Sprint("tcp:", dstAddr))
	if err != nil {
		logrus.Warn("[TCP] parse destination address ", dstAddr, " failed: ", err)
		return err
	}
	go l.Handler.NewConnection(src, dst, conn)
	return nil
}

func (l *LwIP) ReceiveTo(conn core.UDPConn, data []byte, addr *net.UDPAddr) error {
	srcAddr := conn.LocalAddr().String()
	src, err := v2rayNet.ParseDestination(fmt.Sprint("udp:", srcAddr))
	if err != nil {
		logrus.Warn("[UDP] parse source address ", srcAddr, " failed: ", err)
		return err
	}
	dstAddr := addr.String()
	dst, err := v2rayNet.ParseDestination(fmt.Sprint("udp:", dstAddr))
	if err != nil {
		logrus.Warn("[UDP] parse destination address ", dstAddr, " failed: ", err)
		return err
	}
	go l.Handler.NewPacket(src, dst, data, func(bytes []byte, from *net.UDPAddr) (int, error) {
		if from == nil {
			from = addr
		}
		return conn.WriteFrom(bytes, from)
	}, conn)
	return nil
}

func (l *LwIP) Close() error {
	err := l.Stack.Close()
	core.RegisterOutputFn(nil)
	core.RegisterTCPConnHandler(nil)
	core.RegisterUDPConnHandler(nil)
	return err
}
