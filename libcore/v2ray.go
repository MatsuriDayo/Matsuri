package libcore

import (
	"context"
	"errors"
	"fmt"
	"io"
	gonet "net"
	"strconv"
	"strings"
	"sync"
	"time"

	_ "github.com/v2fly/v2ray-core/v5/main/distro/all"

	core "github.com/v2fly/v2ray-core/v5"
	"github.com/v2fly/v2ray-core/v5/app/dispatcher"
	"github.com/v2fly/v2ray-core/v5/app/observatory"
	"github.com/v2fly/v2ray-core/v5/common"
	"github.com/v2fly/v2ray-core/v5/common/buf"
	"github.com/v2fly/v2ray-core/v5/common/net"
	"github.com/v2fly/v2ray-core/v5/common/protocol/udp"
	"github.com/v2fly/v2ray-core/v5/common/signal"
	"github.com/v2fly/v2ray-core/v5/features/dns"
	dns_feature "github.com/v2fly/v2ray-core/v5/features/dns"
	v2rayDns "github.com/v2fly/v2ray-core/v5/features/dns"
	"github.com/v2fly/v2ray-core/v5/features/dns/localdns"
	"github.com/v2fly/v2ray-core/v5/features/extension"
	"github.com/v2fly/v2ray-core/v5/features/routing"
	"github.com/v2fly/v2ray-core/v5/features/stats"
	"github.com/v2fly/v2ray-core/v5/infra/conf/serial"
	"github.com/v2fly/v2ray-core/v5/transport"
	"github.com/v2fly/v2ray-core/v5/transport/internet"
)

func GetV2RayVersion() string {
	return core.Version() + "-喵"
}

type V2RayInstance struct {
	access       sync.Mutex
	started      bool
	core         *core.Instance
	statsManager stats.Manager
	observatory  *observatory.Observer
	dispatcher   *dispatcher.DefaultDispatcher
	dnsClient    dns.Client
}

func NewV2rayInstance() *V2RayInstance {
	return &V2RayInstance{}
}

func (instance *V2RayInstance) LoadConfig(content string) error {
	if outdated != "" {
		return errors.New(outdated)
	}

	instance.access.Lock()
	defer instance.access.Unlock()

	config, err := serial.LoadJSONConfig(strings.NewReader(content))
	if err != nil {
		if strings.HasSuffix(err.Error(), "geoip.dat: no such file or directory") {
			err = extractAssetName(geoipDat, true)
		} else if strings.HasSuffix(err.Error(), "not found in geoip.dat") {
			err = extractAssetName(geoipDat, false)
		} else if strings.HasSuffix(err.Error(), "geosite.dat: no such file or directory") {
			err = extractAssetName(geositeDat, true)
		} else if strings.HasSuffix(err.Error(), "not found in geosite.dat") {
			err = extractAssetName(geositeDat, false)
		}
		if err == nil {
			config, err = serial.LoadJSONConfig(strings.NewReader(content))
		}
	}
	if err != nil {
		return err
	}

	c, err := core.New(config)
	if err != nil {
		return err
	}
	instance.core = c
	instance.statsManager = c.GetFeature(stats.ManagerType()).(stats.Manager)
	instance.dispatcher = c.GetFeature(routing.DispatcherType()).(routing.Dispatcher).(*dispatcher.DefaultDispatcher)
	instance.dnsClient = c.GetFeature(dns.ClientType()).(dns.Client)
	instance.setupDialer(false)

	o := c.GetFeature(extension.ObservatoryType())
	if o != nil {
		instance.observatory = o.(*observatory.Observer)
	}
	return nil
}

func (instance *V2RayInstance) Start() error {
	instance.access.Lock()
	defer instance.access.Unlock()
	if instance.started {
		return newError("already started")
	}
	if instance.core == nil {
		return newError("not initialized")
	}
	err := instance.core.Start()
	if err != nil {
		return err
	}
	instance.started = true
	return nil
}

func (instance *V2RayInstance) QueryStats(tag string, direct string) int64 {
	if instance.statsManager == nil {
		return 0
	}
	counter := instance.statsManager.GetCounter(fmt.Sprintf("outbound>>>%s>>>traffic>>>%s", tag, direct))
	if counter == nil {
		return 0
	}
	return counter.Set(0)
}

func (instance *V2RayInstance) Close() error {
	instance.access.Lock()
	defer instance.access.Unlock()
	if instance.started {
		return instance.core.Close()
	}
	return nil
}

func (instance *V2RayInstance) dialContext(ctx context.Context, destination net.Destination) (net.Conn, error) {
	ctx = core.WithContext(ctx, instance.core)
	r, err := instance.dispatcher.Dispatch(ctx, destination)
	if err != nil {
		return nil, err
	}
	var readerOpt buf.ConnectionOption
	if destination.Network == net.Network_TCP {
		readerOpt = buf.ConnectionOutputMulti(r.Reader)
	} else {
		readerOpt = buf.ConnectionOutputMultiUDP(r.Reader)
	}
	return buf.NewConnection(buf.ConnectionInputMulti(r.Writer), readerOpt), nil
}

func (instance *V2RayInstance) dialUDP(ctx context.Context, destinationConn net.Destination, destinationV2ray net.Destination, timeout time.Duration) (packetConn, error) {
	ctx, cancel := context.WithCancel(core.WithContext(ctx, instance.core))
	link, err := instance.dispatcher.Dispatch(ctx, destinationV2ray)
	if err != nil {
		cancel()
		return nil, err
	}
	c := &dispatcherConn{
		dest:   destinationConn,
		link:   link,
		ctx:    ctx,
		cancel: cancel,
		cache:  make(chan *udp.Packet, 16),
	}
	c.timer = signal.CancelAfterInactivity(ctx, func() {
		closeIgnore(c)
	}, timeout)
	go c.handleInput()
	return c, nil
}

var _ packetConn = (*dispatcherConn)(nil)

type dispatcherConn struct {
	access sync.Mutex
	dest   net.Destination
	link   *transport.Link
	timer  *signal.ActivityTimer

	ctx    context.Context
	cancel context.CancelFunc

	cache chan *udp.Packet
}

func (c *dispatcherConn) handleInput() {
	defer closeIgnore(c)
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
				continue
			}
			packet := udp.Packet{
				Payload: buffer,
			}
			if buffer.Endpoint == nil {
				packet.Source = c.dest
			} else {
				packet.Source = *buffer.Endpoint
			}
			select {
			case c.cache <- &packet:
				continue
			case <-c.ctx.Done():
			default:
			}
			buffer.Release()
		}
	}
}

func (c *dispatcherConn) ReadFrom(p []byte) (n int, addr net.Addr, err error) {
	select {
	case <-c.ctx.Done():
		return 0, nil, io.EOF
	case packet := <-c.cache:
		n := copy(p, packet.Payload.Bytes())
		return n, &net.UDPAddr{
			IP:   packet.Source.Address.IP(),
			Port: int(packet.Source.Port),
		}, nil
	}
}

func (c *dispatcherConn) readFrom() (p []byte, addr net.Addr, err error) {
	select {
	case <-c.ctx.Done():
		return nil, nil, io.EOF
	case packet := <-c.cache:
		return packet.Payload.Bytes(), &net.UDPAddr{
			IP:   packet.Source.Address.IP(),
			Port: int(packet.Source.Port),
		}, nil
	}
}

func (c *dispatcherConn) WriteTo(p []byte, addr net.Addr) (n int, err error) {
	buffer := buf.FromBytes(p)
	endpoint := net.DestinationFromAddr(addr)
	buffer.Endpoint = &endpoint
	err = c.link.Writer.WriteMultiBuffer(buf.MultiBuffer{buffer})
	if err == nil {
		c.timer.Update()
	}
	return
}

func (c *dispatcherConn) LocalAddr() net.Addr {
	return &net.UDPAddr{
		IP:   []byte{0, 0, 0, 0},
		Port: 0,
	}
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
	close(c.cache)

	return nil
}

func (c *dispatcherConn) SetDeadline(t time.Time) error {
	return nil
}

func (c *dispatcherConn) SetReadDeadline(t time.Time) error {
	return nil
}

func (c *dispatcherConn) SetWriteDeadline(t time.Time) error {
	return nil
}

// Nekomura

var androidResolver = &net.Resolver{PreferGo: false}                                  // Using Android API, lookup from current network.
var androidUnderlyingResolver = &simpleSekaiWrapper{androidResolver: androidResolver} // Using Android API, lookup from non-VPN network.

type simpleSekaiWrapper struct {
	androidResolver *net.Resolver
	sekaiResolver   LocalResolver // passed from java (only when VPNService)
}

func (p *simpleSekaiWrapper) LookupIP(network, host string) (ret []net.IP, err error) {
	isSekai := p.sekaiResolver != nil

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	ok := make(chan interface{})
	defer cancel()

	go func() {
		defer func() {
			select {
			case <-ctx.Done():
			default:
				ok <- nil
			}
			close(ok)
		}()

		if isSekai {
			var str string
			str, err = p.sekaiResolver.LookupIP(network, host)
			// java -> go
			if err != nil {
				rcode, err2 := strconv.Atoi(err.Error())
				if err2 == nil {
					err = dns_feature.RCodeError(rcode)
				}
				return
			} else if str == "" {
				err = dns_feature.ErrEmptyResponse
				return
			}
			ret = make([]net.IP, 0)
			for _, ip := range strings.Split(str, ",") {
				ret = append(ret, net.ParseIP(ip))
			}
		} else {
			ret, err = p.androidResolver.LookupIP(context.Background(), network, host)
		}
	}()

	select {
	case <-ctx.Done():
		return nil, newError(fmt.Sprintf("androidUnderlyingResolver: context cancelled! (sekai=%t)", isSekai))
	case <-ok:
		return
	}
}

// setup dialer and resolver for v2ray (v2ray options)
func (v2ray *V2RayInstance) setupDialer(fakedns bool) {
	setupResolvers()
	dc := v2ray.dnsClient

	// All lookup except dnsClient -> dc.LookupIP()
	// and also set protectedDialer
	if c, ok := dc.(v2rayDns.ClientWithIPOption); ok {
		internet.UseAlternativeSystemDialer(&protectedDialer{
			resolver: func(domain string) ([]net.IP, error) {
				c.SetFakeDNSOption(false) // Skip FakeDNS
				return dc.LookupIP(domain)
			},
		})
	} else {
		internet.UseAlternativeSystemDialer(&protectedDialer{
			resolver: func(domain string) ([]net.IP, error) {
				return dc.LookupIP(domain)
			},
		})
	}
}

func setupResolvers() {
	// golang lookup -> androidResolver
	gonet.DefaultResolver = androidResolver

	// dnsClient lookup -> androidUnderlyingResolver.LookupIP()
	internet.UseAlternativeSystemDNSDialer(&protectedDialer{
		resolver: func(domain string) ([]net.IP, error) {
			return androidUnderlyingResolver.LookupIP("ip", domain)
		},
	})

	// "localhost" localDns lookup -> androidUnderlyingResolver.LookupIP()
	localdns.SetLookupFunc(androidUnderlyingResolver.LookupIP)
}
