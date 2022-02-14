package libcore

import (
	"context"
	"errors"
	"fmt"
	"libcore/device"
	"libcore/doh"
	"libcore/protect"
	"libcore/tun"
	gonet "net"
	"strconv"
	"strings"
	"sync"
	"time"

	core "github.com/v2fly/v2ray-core/v5"
	"github.com/v2fly/v2ray-core/v5/app/dispatcher"
	"github.com/v2fly/v2ray-core/v5/app/observatory"
	"github.com/v2fly/v2ray-core/v5/common/buf"
	"github.com/v2fly/v2ray-core/v5/common/net"
	"github.com/v2fly/v2ray-core/v5/common/signal"
	"github.com/v2fly/v2ray-core/v5/features/dns"
	dns_feature "github.com/v2fly/v2ray-core/v5/features/dns"
	v2rayDns "github.com/v2fly/v2ray-core/v5/features/dns"
	"github.com/v2fly/v2ray-core/v5/features/dns/localdns"
	"github.com/v2fly/v2ray-core/v5/features/extension"
	"github.com/v2fly/v2ray-core/v5/features/routing"
	"github.com/v2fly/v2ray-core/v5/features/stats"
	"github.com/v2fly/v2ray-core/v5/infra/conf/serial"
	_ "github.com/v2fly/v2ray-core/v5/main/distro/all"
	"github.com/v2fly/v2ray-core/v5/nekoutils"
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
		} else if strings.HasSuffix(err.Error(), "geosite.dat: no such file or directory") {
			err = extractAssetName(geositeDat, true)
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

	instance.setupDialer()

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

// for udp
func (instance *V2RayInstance) newDispatcherConn(ctx context.Context, destinationConn net.Destination, destinationV2ray net.Destination, timeout time.Duration, writeBack tun.WriteBack) (*dispatcherConn, error) {
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
		closeIgnore(c)
	}, timeout)

	for i := 0; i < device.NumUDPWorkers(); i++ {
		go c.handleDownlink()
	}

	return c, nil
}

// Nekomura

var staticHosts = make(map[string][]net.IP)
var tryDomains = make([]string, 0)                                                    // server's domain, set when enhanced domain mode
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
func (v2ray *V2RayInstance) setupDialer() {
	setupResolvers()
	dc := v2ray.dnsClient

	// All lookup except dnsClient -> dc.LookupIP()
	// and also set protectedDialer
	if c, ok := dc.(v2rayDns.ClientWithIPOption); ok {
		internet.UseAlternativeSystemDialer(&protect.ProtectedDialer{
			Resolver: func(domain string) ([]net.IP, error) {
				if ips, ok := staticHosts[domain]; ok && ips != nil {
					return ips, nil
				}

				// server domain
				if nekoutils.In(tryDomains, domain) {
					// first try A
					ips, err := doh.LookupManyDoH(domain, 1)
					if err != nil {
						// then try AAAA
						ips, err = doh.LookupManyDoH(domain, 28)
						if err != nil {
							return nil, err
						}
					}
					staticHosts[domain] = ips
					return ips, nil
				}

				// other domain
				c.SetFakeDNSOption(false) // Skip FakeDNS
				return dc.LookupIP(domain)
			},
		})
	}
}

func setupResolvers() {
	// golang lookup -> androidResolver
	gonet.DefaultResolver = androidResolver

	// dnsClient lookup -> androidUnderlyingResolver.LookupIP()
	internet.UseAlternativeSystemDNSDialer(&protect.ProtectedDialer{
		Resolver: func(domain string) ([]net.IP, error) {
			return androidUnderlyingResolver.LookupIP("ip", domain)
		},
	})

	// "localhost" localDns lookup -> androidUnderlyingResolver.LookupIP()
	localdns.SetLookupFunc(androidUnderlyingResolver.LookupIP)
}

func SetTryDomains(a string) {
	tryDomains = strings.Split(a, ",")
}
