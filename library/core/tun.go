package libcore

import (
	"context"
	"io"
	"libcore/gvisor"
	"libcore/nat"
	"libcore/tun"
	"math"
	"net"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"

	"github.com/miekg/dns"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	core "github.com/v2fly/v2ray-core/v4"
	"github.com/v2fly/v2ray-core/v4/common"
	"github.com/v2fly/v2ray-core/v4/common/buf"
	v2rayNet "github.com/v2fly/v2ray-core/v4/common/net"
	"github.com/v2fly/v2ray-core/v4/common/session"
	"github.com/v2fly/v2ray-core/v4/common/task"
	"github.com/v2fly/v2ray-core/v4/transport"
	"github.com/v2fly/v2ray-core/v4/transport/pipe"
)

var _ tun.Handler = (*Tun2ray)(nil)

type Tun2ray struct {
	access              sync.Mutex
	dev                 tun.Tun
	router              string
	hijackDns           bool
	v2ray               *V2RayInstance
	udpTable            *natTable
	fakedns             bool
	sniffing            bool
	overrideDestination bool
	debug               bool

	dumpUid      bool
	trafficStats bool
	appStats     map[uint16]*appStats
	pcap         bool
}

type ErrorHandler interface {
	HandleError(err string)
}

func NewTun2ray(fd int32, mtu int32, v2ray *V2RayInstance,
	router string, tunImpl int32, hijackDns bool, sniffing bool,
	overrideDestination bool, fakedns bool, debug bool,
	dumpUid bool, trafficStats bool, pcap bool, errorHandler ErrorHandler) (*Tun2ray, error) {
	/*	if fd < 0 {
			return nil, errors.New("must provide a valid TUN file descriptor")
		}
		// Make a copy of `fd` so that os.File's finalizer doesn't close `fd`.
		newFd, err := unix.Dup(int(fd))
		if err != nil {
			return nil, err
		}*/

	if debug {
		logrus.SetLevel(logrus.DebugLevel)
	} else {
		logrus.SetLevel(logrus.WarnLevel)
	}
	t := &Tun2ray{
		router:              router,
		hijackDns:           hijackDns,
		v2ray:               v2ray,
		udpTable:            &natTable{},
		sniffing:            sniffing,
		overrideDestination: overrideDestination,
		fakedns:             fakedns,
		debug:               debug,
		dumpUid:             dumpUid,
		trafficStats:        trafficStats,
	}

	if trafficStats {
		t.appStats = map[uint16]*appStats{}
	}
	var err error
	if tunImpl == 0 { // gvisor
		var pcapFile *os.File
		if pcap {
			path := time.Now().UTC().String()
			path = externalAssetsPath + "/pcap/" + path + ".pcap"
			err = os.MkdirAll(filepath.Dir(path), 0755)
			if err != nil {
				return nil, errors.WithMessage(err, "unable to create pcap dir")
			}
			pcapFile, err = os.Create(path)
			if err != nil {
				return nil, errors.WithMessage(err, "unable to create pcap file")
			}
		}

		t.dev, err = gvisor.New(fd, mtu, t, gvisor.DefaultNIC, pcap, pcapFile, math.MaxUint32, ipv6Mode)
	} else if tunImpl == 1 { // SYSTEM
		t.dev, err = nat.New(fd, mtu, t, ipv6Mode, errorHandler.HandleError)
	} else {
		err = errors.New("Not supported")
	}
	if err != nil {
		return nil, err
	}

	v2ray.setupDialer(fakedns)
	net.DefaultResolver.Dial = t.dialDNS

	return t, nil
}

func (t *Tun2ray) Close() {
	t.access.Lock()
	defer t.access.Unlock()

	net.DefaultResolver.Dial = nil
	closeIgnore(t.dev)
}

//TCP
func (t *Tun2ray) NewConnection(source v2rayNet.Destination, destination v2rayNet.Destination, conn net.Conn) {
	inbound := &session.Inbound{
		Source: source,
		Tag:    "socks",
	}

	isDns := destination.Address.String() == t.router
	if isDns {
		inbound.Tag = "dns-in"
	}

	var uid uint16
	var self bool

	if t.dumpUid || t.trafficStats {
		u, err := uidDumper.DumpUid(destination.Address.Family().IsIPv6(), false, source.Address.IP().String(), int32(source.Port), destination.Address.IP().String(), int32(destination.Port))
		if err == nil {
			uid = uint16(u)
			var info *UidInfo
			self = uid > 0 && int(uid) == os.Getuid()
			if t.debug && !self && uid >= 10000 {
				if err == nil {
					info, _ = uidDumper.GetUidInfo(int32(uid))
				}
				if info == nil {
					logrus.Infof("[TCP] %s ==> %s", source.NetAddr(), destination.NetAddr())
				} else {
					logrus.Infof("[TCP][%s (%d/%s)] %s ==> %s", info.Label, uid, info.PackageName, source.NetAddr(), destination.NetAddr())
				}
			}

			if uid < 10000 {
				uid = 1000
			}

			inbound.Uid = uint32(uid)
		}
	}

	ctx := core.WithContext(context.Background(), t.v2ray.core)
	ctx = session.ContextWithInbound(ctx, inbound)

	if !isDns && (t.sniffing || t.fakedns) {
		req := session.SniffingRequest{
			Enabled:      true,
			MetadataOnly: t.fakedns && !t.sniffing,
			RouteOnly:    !t.overrideDestination,
		}
		if t.sniffing && t.fakedns {
			req.OverrideDestinationForProtocol = []string{"fakedns", "http", "tls"}
		}
		if t.sniffing && !t.fakedns {
			req.OverrideDestinationForProtocol = []string{"http", "tls"}
		}
		if !t.sniffing && t.fakedns {
			req.OverrideDestinationForProtocol = []string{"fakedns"}
		}
		ctx = session.ContextWithContent(ctx, &session.Content{
			SniffingRequest: req,
		})
	}

	if t.trafficStats && !self && !isDns {
		stats := t.appStats[uid]
		if stats == nil {
			t.access.Lock()
			stats = t.appStats[uid]
			if stats == nil {
				stats = &appStats{}
				t.appStats[uid] = stats
			}
			t.access.Unlock()
		}
		atomic.AddInt32(&stats.tcpConn, 1)
		atomic.AddUint32(&stats.tcpConnTotal, 1)
		atomic.StoreInt64(&stats.deactivateAt, 0)
		defer func() {
			if atomic.AddInt32(&stats.tcpConn, -1)+atomic.LoadInt32(&stats.udpConn) == 0 {
				atomic.StoreInt64(&stats.deactivateAt, time.Now().Unix())
			}
		}()
		conn = &statsConn{conn, &stats.uplink, &stats.downlink}
	}

	defer closeIgnore(conn)

	upLinkReader, upLinkWriter := pipe.New()
	link := &transport.Link{Reader: upLinkReader, Writer: connWriter{conn, buf.NewWriter(conn)}}
	err := t.v2ray.dispatcher.DispatchLink(ctx, destination, link)
	if err != nil {
		logrus.Errorf("[TCP] dispatchLink failed: %s", err.Error())
		closeIgnore(link.Reader, link.Writer)
		return
	}

	err = task.Run(ctx, func() error {
		// copy uplink traffic
		return buf.Copy(buf.NewReader(conn), upLinkWriter)
	})

	// connection ends
	if err == nil {
		err = common.Close(link.Writer)
		common.Close(link.Reader)
	}

	// Close/Interrupt link.Reader breaks mux?
	if err != nil {
		logrus.Warnf("[TCP] Error transport / closing: %s", err.Error())
		common.Interrupt(link.Reader)
		common.Interrupt(link.Writer)
	}
}

type connWriter struct {
	net.Conn
	buf.Writer
}

//UDP
func (t *Tun2ray) NewPacket(source v2rayNet.Destination, destination v2rayNet.Destination, data []byte, writeBack func([]byte, *net.UDPAddr) (int, error), closer io.Closer) {
	natKey := source.NetAddr()

	sendTo := func() bool {
		conn := t.udpTable.Get(natKey)
		if conn == nil {
			return false
		}
		_, err := conn.WriteTo(data, &net.UDPAddr{
			IP:   destination.Address.IP(),
			Port: int(destination.Port),
		})
		if err != nil {
			_ = conn.Close()
		}
		return true
	}

	if sendTo() {
		return
	}

	lockKey := natKey + "-lock"
	cond, loaded := t.udpTable.GetOrCreateLock(lockKey)
	if loaded {
		cond.L.Lock()
		cond.Wait()
		sendTo()
		cond.L.Unlock()
		return
	}

	t.udpTable.Delete(lockKey)
	cond.Broadcast()

	inbound := &session.Inbound{
		Source: source,
		Tag:    "socks",
	}
	isDns := destination.Address.String() == t.router

	if !isDns && t.hijackDns {
		dnsMsg := dns.Msg{}
		err := dnsMsg.Unpack(data)
		if err == nil && !dnsMsg.Response && len(dnsMsg.Question) > 0 {
			isDns = true
		}
	}

	if isDns {
		inbound.Tag = "dns-in"
	}

	var uid uint16
	var self bool

	if t.dumpUid || t.trafficStats {

		u, err := uidDumper.DumpUid(source.Address.Family().IsIPv6(), true, source.Address.String(), int32(source.Port), destination.Address.String(), int32(destination.Port))
		if err == nil {
			uid = uint16(u)
			var info *UidInfo
			self = uid > 0 && int(uid) == os.Getuid()

			if t.debug && !self && uid >= 1000 {
				if err == nil {
					info, _ = uidDumper.GetUidInfo(int32(uid))
				}
				var tag string
				if !isDns {
					tag = "UDP"
				} else {
					tag = "DNS"
				}

				if info == nil {
					logrus.Infof("[%s] %s ==> %s", tag, source.NetAddr(), destination.NetAddr())
				} else {
					logrus.Infof("[%s][%s (%d/%s)] %s ==> %s", tag, info.Label, uid, info.PackageName, source.NetAddr(), destination.NetAddr())
				}
			}

			if uid < 10000 {
				uid = 1000
			}

			inbound.Uid = uint32(uid)
		}

	}

	ctx := session.ContextWithInbound(context.Background(), inbound)

	if !isDns && t.fakedns {
		ctx = session.ContextWithContent(ctx, &session.Content{
			SniffingRequest: session.SniffingRequest{
				Enabled:                        true,
				MetadataOnly:                   t.fakedns && !t.sniffing,
				OverrideDestinationForProtocol: []string{"fakedns"},
				RouteOnly:                      !t.overrideDestination,
			},
		})
	}

	conn, err := t.v2ray.dialUDP(ctx, destination, time.Minute*5)

	if err != nil {
		logrus.Errorf("[UDP] dial failed: %s", err.Error())
		return
	}

	if t.trafficStats && !self && !isDns {
		stats := t.appStats[uid]
		if stats == nil {
			t.access.Lock()
			stats = t.appStats[uid]
			if stats == nil {
				stats = &appStats{}
				t.appStats[uid] = stats
			}
			t.access.Unlock()
		}
		atomic.AddInt32(&stats.udpConn, 1)
		atomic.AddUint32(&stats.udpConnTotal, 1)
		atomic.StoreInt64(&stats.deactivateAt, 0)
		defer func() {
			if atomic.AddInt32(&stats.udpConn, -1)+atomic.LoadInt32(&stats.tcpConn) == 0 {
				atomic.StoreInt64(&stats.deactivateAt, time.Now().Unix())
			}
		}()
		conn = &statsPacketConn{conn, &stats.uplink, &stats.downlink}
	}

	t.udpTable.Set(natKey, conn)

	go sendTo()

	for {
		buffer, addr, err := conn.readFrom()
		if err != nil {
			break
		}
		if isDns {
			addr = nil
		}
		if addr, ok := addr.(*net.UDPAddr); ok {
			_, err = writeBack(buffer, addr)
		} else {
			_, err = writeBack(buffer, nil)
		}
		if err != nil {
			break
		}
	}
	// close
	closeIgnore(conn, closer)
	t.udpTable.Delete(natKey)
}

func (t *Tun2ray) dialDNS(ctx context.Context, _, _ string) (conn net.Conn, err error) {
	conn, err = t.v2ray.dialContext(session.ContextWithInbound(ctx, &session.Inbound{
		Tag:         "dns-in",
		SkipFakeDNS: true,
	}), v2rayNet.Destination{
		Network: v2rayNet.Network_UDP,
		Address: v2rayNet.ParseAddress("1.0.0.1"),
		Port:    53,
	})
	if err == nil {
		conn = wrappedConn{conn}
	}
	return
}

type wrappedConn struct {
	net.Conn
}

func (c wrappedConn) ReadFrom(p []byte) (n int, addr net.Addr, err error) {
	n, err = c.Conn.Read(p)
	if err == nil {
		addr = c.Conn.RemoteAddr()
	}
	return
}

func (c wrappedConn) WriteTo(p []byte, _ net.Addr) (n int, err error) {
	return c.Conn.Write(p)
}

type natTable struct {
	mapping sync.Map
}

func (t *natTable) Set(key string, pc net.PacketConn) {
	t.mapping.Store(key, pc)
}

func (t *natTable) Get(key string) net.PacketConn {
	item, exist := t.mapping.Load(key)
	if !exist {
		return nil
	}
	return item.(net.PacketConn)
}

func (t *natTable) GetOrCreateLock(key string) (*sync.Cond, bool) {
	item, loaded := t.mapping.LoadOrStore(key, sync.NewCond(&sync.Mutex{}))
	return item.(*sync.Cond), loaded
}

func (t *natTable) Delete(key string) {
	t.mapping.Delete(key)
}

var ipv6Mode int32

func SetIPv6Mode(mode int32) {
	ipv6Mode = mode
}
