package libcore

import (
	"context"
	"libcore/comm"
	"libcore/tun"
	"sync"
	"sync/atomic"
	"time"

	core "github.com/v2fly/v2ray-core/v5"
	"github.com/v2fly/v2ray-core/v5/common"
	"github.com/v2fly/v2ray-core/v5/common/buf"
	"github.com/v2fly/v2ray-core/v5/common/net"
	"github.com/v2fly/v2ray-core/v5/common/signal"
	_ "github.com/v2fly/v2ray-core/v5/main/distro/all"
	"github.com/v2fly/v2ray-core/v5/transport"
)

// this file is for v2ray's udp

func (instance *V2RayInstance) newDispatcherConn(ctx context.Context, destinationConn net.Destination, destinationV2ray net.Destination,
	writeBack tun.WriteBack, timeout time.Duration, workerN int,
) (*dispatcherConn, error) {
	ctx, cancel := context.WithCancel(core.WithContext(ctx, instance.core))
	link, err := instance.dispatcher.Dispatch(ctx, destinationV2ray)
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

type dispatcherConn struct {
	access sync.Mutex
	dest   net.Destination
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

			var src net.Destination

			if buffer.Endpoint == nil {
				src = c.dest
			} else {
				src = *buffer.Endpoint
			}

			if src.Address.Family().IsDomain() {
				src.Address = net.AnyIP
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
	endpoint := net.DestinationFromAddr(addr)
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
