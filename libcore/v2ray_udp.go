package libcore

import (
	"context"
	"libcore/comm"
	"libcore/tun"
	"net"
	"sync"
	"sync/atomic"
	"time"

	core "github.com/v2fly/v2ray-core/v5"
	"github.com/v2fly/v2ray-core/v5/common"
	"github.com/v2fly/v2ray-core/v5/common/buf"
	net2 "github.com/v2fly/v2ray-core/v5/common/net"
	"github.com/v2fly/v2ray-core/v5/common/signal"
	_ "github.com/v2fly/v2ray-core/v5/main/distro/all"
	"github.com/v2fly/v2ray-core/v5/transport"
)

// this file is for v2ray's udp

// dispatcherConn (for zero-copy)

func (instance *V2RayInstance) newDispatcherConn(ctx context.Context, destinationConn net2.Destination, destinationV2ray net2.Destination,
	writeBack tun.WriteBack, timeout time.Duration, workerN int,
) (*dispatcherConn, error) {
	ctx, cancel := context.WithCancel(core.WithContext(ctx, instance.Core))
	link, err := instance.Dispatcher.Dispatch(ctx, destinationV2ray)
	if err != nil {
		cancel()
		return nil, err
	}
	c := &dispatcherConn{
		dest:      destinationConn,
		link:      link,
		ctx:       ctx,
		cancel:    cancel,
		writeBack: writeBack,
	}
	c.timer = signal.CancelAfterInactivity(ctx, func() {
		comm.CloseIgnore(c)
	}, timeout)

	for i := 0; i < workerN; i++ {
		go c.handleDownlink()
	}

	return c, nil
}

type myStats struct {
	uplink   *uint64
	downlink *uint64
}

type dispatcherConn struct {
	access sync.Mutex
	dest   net2.Destination
	link   *transport.Link
	timer  *signal.ActivityTimer

	ctx    context.Context
	cancel context.CancelFunc

	writeBack tun.WriteBack //downlink

	stats *myStats // traffic stats
}

func (c *dispatcherConn) handleDownlink() {
	defer comm.CloseIgnore(c)

	for {
		select {
		case <-c.ctx.Done():
			return
		default:
		}

		mb, err := c.link.Reader.ReadMultiBuffer()
		if err != nil {
			buf.ReleaseMulti(mb)
			return
		}
		c.timer.Update()

		for _, buffer := range mb {
			if buffer.Len() <= 0 {
				buffer.Release()
				continue
			}

			var src net2.Destination
			if buffer.Endpoint == nil {
				src = c.dest
			} else {
				src = *buffer.Endpoint
			}
			if src.Address.Family().IsDomain() {
				src.Address = net2.AnyIP
			}

			n, err := c.writeBack(buffer.Bytes(), &net.UDPAddr{
				IP:   src.Address.IP(),
				Port: int(src.Port),
			})

			buffer.Release()

			if err == nil {
				if c.stats != nil {
					atomic.AddUint64(c.stats.downlink, uint64(n))
				}
			} else {
				return
			}
		}
	}
}

// uplink
func (c *dispatcherConn) WriteTo(p []byte, addr net.Addr) (n int, err error) {
	if c.stats != nil {
		atomic.AddUint64(c.stats.uplink, uint64(len(p)))
	}

	buffer := buf.FromBytes(p)
	endpoint := net2.DestinationFromAddr(addr)
	buffer.Endpoint = &endpoint
	err = c.link.Writer.WriteMultiBuffer(buf.MultiBuffer{buffer})
	if err == nil {
		c.timer.Update()
	}
	return
}

func (c *dispatcherConn) Close() error {
	c.access.Lock()
	defer c.access.Unlock()

	select {
	case <-c.ctx.Done():
		return nil
	default:
	}

	c.cancel()
	_ = common.Interrupt(c.link.Reader)
	_ = common.Interrupt(c.link.Writer)

	return nil
}

// udpConn (for go net)

func (instance *V2RayInstance) DialUDP(raddr *net.UDPAddr) (*udpConn, error) {
	ctx := core.WithContext(context.Background(), instance.Core)
	destination := net2.DestinationFromAddr(raddr)
	r, err := instance.Dispatcher.Dispatch(ctx, destination)
	if err != nil {
		return nil, err
	}
	return &udpConn{
		r.Reader, r.Writer,
	}, nil
}

var _ net.PacketConn = (*udpConn)(nil)

type udpConn struct {
	reader buf.Reader
	writer buf.Writer
}

func (c *udpConn) ReadFrom(p []byte) (int, net.Addr, error) {
	mb, err := c.reader.ReadMultiBuffer()
	if err != nil {
		buf.ReleaseMulti(mb)
		return 0, nil, err
	}

	var n int
	var total int
	var endpoint net.Addr

	for _, buffer := range mb {
		if buffer.Len() <= 0 {
			buffer.Release()
			continue
		}

		var src net2.Destination
		if buffer.Endpoint == nil {
			src.Address = net2.AnyIP
		} else {
			src = *buffer.Endpoint
		}
		if src.Address.Family().IsDomain() {
			src.Address = net2.AnyIP
		}
		endpoint = &net.UDPAddr{
			IP:   src.Address.IP(),
			Port: int(src.Port.Value()),
		}

		n, err = buffer.Read(p)
		total += n
		buffer.Release()

		break
	}

	return total, endpoint, err
}

func (c *udpConn) WriteTo(p []byte, addr net.Addr) (int, error) {
	buffer := buf.FromBytes(p)
	endpoint := net2.DestinationFromAddr(addr)
	buffer.Endpoint = &endpoint
	err := c.writer.WriteMultiBuffer(buf.MultiBuffer{buffer})
	if err == nil {
		return len(p), nil
	}
	return 0, err
}

func (c *udpConn) Close() error {
	common.Close(c.writer)
	common.Close(c.reader)
	return nil
}

func (c *udpConn) LocalAddr() net.Addr                { return &net.UDPAddr{} }
func (c *udpConn) SetDeadline(t time.Time) error      { return nil }
func (c *udpConn) SetReadDeadline(t time.Time) error  { return nil }
func (c *udpConn) SetWriteDeadline(t time.Time) error { return nil }
