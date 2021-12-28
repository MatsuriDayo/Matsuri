package libcore

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/Dreamacro/clash/adapter/inbound"
	"github.com/Dreamacro/clash/adapter/outbound"
	"github.com/Dreamacro/clash/common/pool"
	"github.com/Dreamacro/clash/component/nat"
	"github.com/Dreamacro/clash/constant"
	clashC "github.com/Dreamacro/clash/constant"
	"github.com/Dreamacro/clash/listener/socks"
	"github.com/sirupsen/logrus"
	"github.com/v2fly/v2ray-core/v4/common/task"
	"io"
	"net"
	"sync"
	"time"
)

type ClashBasedInstance struct {
	access    sync.Mutex
	socksPort int32
	ctx       chan constant.ConnContext
	in        *socks.Listener
	udpIn     *socks.UDPListener
	udpCtx    chan *inbound.PacketAdapter
	out       clashC.ProxyAdapter
	nat       nat.Table
	started   bool
}

func (s *ClashBasedInstance) DialContext(ctx context.Context, network, address string) (net.Conn, error) {
	dest, err := addrToMetadata(address)
	if err != nil {
		return nil, err
	}
	dest.NetWork = networkForClash(network)
	return s.out.DialContext(ctx, dest)
}

func newClashBasedInstance(socksPort int32, out clashC.ProxyAdapter) *ClashBasedInstance {
	return &ClashBasedInstance{
		socksPort: socksPort,
		ctx:       make(chan constant.ConnContext, 64),
		udpCtx:    make(chan *inbound.PacketAdapter, 64),
		out:       out,
	}
}

func (s *ClashBasedInstance) Start() error {
	s.access.Lock()
	defer s.access.Unlock()

	if s.started {
		return newError("already started")
	}

	addr := fmt.Sprintf("127.0.0.1:%d", s.socksPort)

	in, err := socks.New(addr, s.ctx)
	if err != nil {
		return newError("create socks inbound").Base(err)
	}
	udpIn, err := socks.NewUDP(addr, s.udpCtx)
	if err != nil {
		return newError("create socks udp inbound").Base(err)
	}
	s.in = in
	s.udpIn = udpIn
	s.started = true
	go s.loop()
	go s.loopUdp()
	return nil
}

func (s *ClashBasedInstance) Close() error {
	s.access.Lock()
	defer s.access.Unlock()

	if !s.started {
		return newError("not started")
	}

	closeIgnore(s.in, s.udpIn, s.out, s.ctx, s.udpCtx)
	return nil
}

func (s *ClashBasedInstance) loop() {
	for conn := range s.ctx {
		conn := conn
		metadata := conn.Metadata()
		go func() {
			ctx := context.Background()
			remote, err := s.out.DialContext(ctx, metadata)
			if err != nil {
				fmt.Printf("Dial error: %s\n", err.Error())
				return
			}

			_ = task.Run(ctx, func() error {
				_, _ = io.Copy(remote, conn.Conn())
				return io.EOF
			}, func() error {
				_, _ = io.Copy(conn.Conn(), remote)
				return io.EOF
			})

			_ = remote.Close()
			_ = conn.Conn().Close()
		}()
	}
}

var udpTimeout = 60 * time.Second

func (s *ClashBasedInstance) loopUdp() {
	for packet := range s.udpCtx {
		metadata := packet.Metadata()
		if !metadata.Valid() {
			logrus.Warnln("[Metadata] not valid: ", metadata)
			continue
		}

		packet := packet
		go func() {
			key := packet.LocalAddr().String()

			handle := func() bool {
				pc := s.nat.Get(key)
				if pc != nil {
					_, err := pc.WriteTo(packet.Data(), metadata.UDPAddr())
					if err != nil {
						packet.Drop()
						_ = pc.Close()
					}
					return true
				}
				return false
			}

			if handle() {
				packet.Drop()
				return
			}

			lockKey := key + "-lock"
			cond, loaded := s.nat.GetOrCreateLock(lockKey)

			if loaded {
				cond.L.Lock()
				cond.Wait()
				handle()
				cond.L.Unlock()
				return
			}

			ctx := context.Background()
			remote, err := s.out.DialUDP(metadata)
			if err != nil {
				fmt.Printf("Dial UDP error: %s\n", err.Error())
				return
			}
			s.nat.Set(key, remote)
			go handle()
			buf := pool.Get(pool.RelayBufferSize)
			_ = task.Run(ctx, func() error {
				for {
					_ = remote.SetReadDeadline(time.Now().Add(udpTimeout))
					n, from, err := remote.ReadFrom(buf)
					if err == nil {
						_, err = packet.WriteBack(buf[:n], from)
					}
					if err != nil {
						return err
					}
				}
			})
			_ = pool.Put(buf)
			_ = remote.Close()
			packet.Drop()
			s.nat.Delete(lockKey)
			cond.Broadcast()
		}()
	}
}

func addrToMetadata(rawAddress string) (addr *clashC.Metadata, err error) {
	host, port, err := net.SplitHostPort(rawAddress)
	if err != nil {
		err = fmt.Errorf("addrToMetadata failed: %w", err)
		return
	}

	ip := net.ParseIP(host)
	if ip == nil {
		addr = &clashC.Metadata{
			AddrType: clashC.AtypDomainName,
			Host:     host,
			DstIP:    nil,
			DstPort:  port,
		}
		return
	} else if ip4 := ip.To4(); ip4 != nil {
		addr = &clashC.Metadata{
			AddrType: clashC.AtypIPv4,
			Host:     "",
			DstIP:    ip4,
			DstPort:  port,
		}
		return
	}

	addr = &clashC.Metadata{
		AddrType: clashC.AtypIPv6,
		Host:     "",
		DstIP:    ip,
		DstPort:  port,
	}
	return
}

func networkForClash(network string) clashC.NetWork {
	switch network {
	case "tcp", "tcp4", "tcp6":
		return clashC.TCP
	case "udp", "udp4", "udp6":
		return clashC.UDP
	}
	logrus.Fatalln("unexpected network name", network)
	return 0
}

func tcpKeepAlive(c net.Conn) {
	if tcp, ok := c.(*net.TCPConn); ok {
		_ = tcp.SetKeepAlive(true)
		_ = tcp.SetKeepAlivePeriod(30 * time.Second)
	}
}

func safeConnClose(c net.Conn, err error) {
	if err != nil {
		_ = c.Close()
	}
}

func NewShadowsocksInstance(socksPort int32, server string, port int32, password string, cipher string, plugin string, pluginOpts string) (*ClashBasedInstance, error) {
	if cipher == "none" {
		cipher = "dummy"
	}
	if plugin == "obfs-local" || plugin == "simple-obfs" {
		plugin = "obfs"
	}
	opts := map[string]interface{}{}
	err := json.Unmarshal([]byte(pluginOpts), &opts)
	if err != nil {
		return nil, err
	}
	out, err := outbound.NewShadowSocks(outbound.ShadowSocksOption{
		Server:     server,
		Port:       int(port),
		Password:   password,
		Cipher:     cipher,
		Plugin:     plugin,
		PluginOpts: opts,
	})
	if err != nil {
		return nil, err
	}
	return newClashBasedInstance(socksPort, out), nil
}

func NewShadowsocksRInstance(socksPort int32, server string, port int32, password string, cipher string, obfs string, obfsParam string, protocol string, protocolParam string) (*ClashBasedInstance, error) {
	if cipher == "none" {
		cipher = "dummy"
	}
	out, err := outbound.NewShadowSocksR(outbound.ShadowSocksROption{
		Server:        server,
		Port:          int(port),
		Password:      password,
		Cipher:        cipher,
		Obfs:          obfs,
		ObfsParam:     obfsParam,
		Protocol:      protocol,
		ProtocolParam: protocolParam,
		UDP:           true,
	})
	if err != nil {
		return nil, err
	}
	return newClashBasedInstance(socksPort, out), nil
}

func NewSnellInstance(socksPort int32, server string, port int32, psk string, obfsMode string, obfsHost string, version int32) (*ClashBasedInstance, error) {
	obfs := map[string]interface{}{}
	obfs["mode"] = obfsMode
	obfs["host"] = obfsHost
	out, err := outbound.NewSnell(outbound.SnellOption{
		Server:   server,
		Port:     int(port),
		Psk:      psk,
		Version:  int(version),
		ObfsOpts: obfs,
	})
	if err != nil {
		return nil, err
	}
	return newClashBasedInstance(socksPort, out), nil
}
