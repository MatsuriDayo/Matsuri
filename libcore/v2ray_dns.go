package libcore

import (
	"context"
	"fmt"
	"libcore/device"
	"libcore/doh"
	"libcore/protect"
	"net"
	"strconv"
	"strings"
	"sync/atomic"
	"time"
	"unsafe"

	dns_feature "github.com/v2fly/v2ray-core/v5/features/dns"
	"github.com/v2fly/v2ray-core/v5/features/dns/localdns"
	"github.com/v2fly/v2ray-core/v5/nekoutils"
	"github.com/v2fly/v2ray-core/v5/transport/internet"
)

// DNS & Protect

var staticHosts = make(map[string][]net.IP)
var tryDomains = make([]string, 0)                                           // server's domain, set when enhanced domain mode
var systemResolver = &net.Resolver{PreferGo: false}                          // Using System API, lookup from current network.
var underlyingResolver = &simpleSekaiWrapper{systemResolver: systemResolver} // Using System API, lookup from non-VPN network.
var v2rayDNSClient unsafe.Pointer
var underlyingDialer = &protect.ProtectedDialer{
	Resolver: func(domain string) ([]net.IP, error) {
		return underlyingResolver.LookupIP("ip", domain)
	},
}

// sekaiResolver
type LocalResolver interface {
	LookupIP(network string, domain string) (string, error)
}

type simpleSekaiWrapper struct {
	systemResolver *net.Resolver
	sekaiResolver  LocalResolver // Android: passed from java (only when VPNService)
}

func (p *simpleSekaiWrapper) LookupIP(network, host string) (ret []net.IP, err error) {
	// NOTE only Android
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
			ret, err = p.systemResolver.LookupIP(context.Background(), network, host)
		}
	}()

	select {
	case <-ctx.Done():
		return nil, newError(fmt.Sprintf("underlyingResolver: context cancelled! (sekai=%t)", isSekai))
	case <-ok:
		return
	}
}

func setupResolvers() {
	// golang lookup -> System
	net.DefaultResolver = systemResolver

	// dnsClient lookup -> Underlying
	internet.UseAlternativeSystemDNSDialer(underlyingDialer)

	// "localhost" localDns lookup -> Underlying
	if !device.IsNekoray {
		localdns.SetLookupFunc(underlyingResolver.LookupIP)
	}

	// doh package
	doh.SetDialContext(underlyingDialer.DialContext)

	// All lookup except dnsClient -> dc.LookupIP()
	// and also set protectedDialer for outbound connections
	internet.UseAlternativeSystemDialer(&protect.ProtectedDialer{
		Resolver: func(domain string) ([]net.IP, error) {
			if ips, ok := staticHosts[domain]; ok && ips != nil {
				return ips, nil
			}

			if nekoutils.In(tryDomains, domain) {
				// first try A
				_ips, err := doh.LookupManyDoH(domain, 1)
				if err != nil {
					// then try AAAA
					_ips, err = doh.LookupManyDoH(domain, 28)
					if err != nil {
						return nil, err
					}
				}
				ips := _ips.([]net.IP)
				staticHosts[domain] = ips
				return ips, nil
			}

			// Have running instance?
			ptr := (*dns_feature.Client)(atomic.LoadPointer(&v2rayDNSClient))
			if ptr != nil && *ptr != nil {
				return (*ptr).LookupIP(&dns_feature.MatsuriDomainStringEx{
					Domain:     domain,
					OptNetwork: "ip",
				})
			} else {
				return systemResolver.LookupIP(context.Background(), "ip", domain)
			}
		},
	})
}
