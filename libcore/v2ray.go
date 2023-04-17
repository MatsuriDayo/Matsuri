package libcore

import (
	"context"
	"fmt"
	"libcore/device"
	"log"
	"strings"
	"sync"
	"sync/atomic"
	"unsafe"

	core "github.com/v2fly/v2ray-core/v5"
	"github.com/v2fly/v2ray-core/v5/app/dispatcher"
	"github.com/v2fly/v2ray-core/v5/common/buf"
	"github.com/v2fly/v2ray-core/v5/common/net"
	dns_feature "github.com/v2fly/v2ray-core/v5/features/dns"
	"github.com/v2fly/v2ray-core/v5/features/routing"
	"github.com/v2fly/v2ray-core/v5/features/stats"
	"github.com/v2fly/v2ray-core/v5/infra/conf/serial"
	"github.com/v2fly/v2ray-core/v5/nekoutils"
)

func GetV2RayVersion() string {
	return core.Version() + "-喵"
}

type V2RayInstance struct {
	access       sync.Mutex
	started      bool
	Core         *core.Instance
	StatsManager stats.Manager
	Dispatcher   *dispatcher.DefaultDispatcher
	DnsClient    dns_feature.Client
	ForTest      bool
}

func NewV2rayInstance() *V2RayInstance {
	return &V2RayInstance{}
}

func (instance *V2RayInstance) LoadConfig(content string) error {
	defer device.DeferPanicToError("LoadConfig", func(err error) { log.Println(err) })

	instance.access.Lock()
	defer instance.access.Unlock()

	// load v4 or v5 config
	var config *core.Config
	var err error
	if content2 := strings.TrimPrefix(content, "matsuri-v2ray-v5"); content2 != content {
		config, err = core.LoadConfig("jsonv5", strings.NewReader(content2))
		if err != nil {
			log.Println(content, err.Error())
			return err
		}
	} else {
		config, err = serial.LoadJSONConfig(strings.NewReader(content))
		if err != nil {
			log.Println(content, err.Error())
			return err
		}
	}

	c, err := core.New(config)
	if err != nil {
		return err
	}

	instance.Core = c
	instance.StatsManager = c.GetFeature(stats.ManagerType()).(stats.Manager)
	instance.Dispatcher = c.GetFeature(routing.DispatcherType()).(routing.Dispatcher).(*dispatcher.DefaultDispatcher)
	instance.DnsClient = c.GetFeature(dns_feature.ClientType()).(dns_feature.Client)

	if !instance.ForTest {
		atomic.StorePointer(&v2rayDNSClient, unsafe.Pointer(&instance.DnsClient))
	}

	return nil
}

func (instance *V2RayInstance) Start() error {
	defer device.DeferPanicToError("Start", func(err error) { log.Println(err) })

	instance.access.Lock()
	defer instance.access.Unlock()
	if instance.started {
		return newError("already started")
	}
	if instance.Core == nil {
		return newError("not initialized")
	}
	err := instance.Core.Start()
	if err != nil {
		return err
	}
	instance.started = true
	return nil
}

func (instance *V2RayInstance) QueryStats(tag string, direct string) int64 {
	if instance.StatsManager == nil {
		return 0
	}
	counter := instance.StatsManager.GetCounter(fmt.Sprintf("outbound>>>%s>>>traffic>>>%s", tag, direct))
	if counter == nil {
		return 0
	}
	return counter.Set(0)
}

func (instance *V2RayInstance) Close() error {
	defer device.DeferPanicToError("Close", func(err error) { log.Println(err) })

	instance.access.Lock()
	defer instance.access.Unlock()
	if instance.started {
		if !instance.ForTest {
			atomic.StorePointer(&v2rayDNSClient, unsafe.Pointer(nil))
		}
		nekoutils.ResetConnections(uintptr(unsafe.Pointer(instance.Core)))
		return instance.Core.Close()
	}
	return nil
}

func (instance *V2RayInstance) DialContext(ctx context.Context, destination net.Destination) (net.Conn, error) {
	ctx = core.WithContext(ctx, instance.Core)
	r, err := instance.Dispatcher.Dispatch(ctx, destination)
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

// Neko connections

func (instance *V2RayInstance) SetConnectionPoolEnabled(enable bool) {
	nekoutils.SetConnectionPoolV2RayEnabled(uintptr(unsafe.Pointer(instance.Core)), enable)
	nekoutils.ResetAllConnections(false)
}

func ResetAllConnections(system bool) {
	nekoutils.ResetAllConnections(system)
}

func ListV2rayConnections() string {
	return nekoutils.ListConnections(0)
}

func CloseV2rayConnection(id uint32) {
	m := &nekoutils.ConnectionPool_V2Ray.Map

	m.Range(func(key interface{}, value interface{}) bool {
		if c, ok := key.(*nekoutils.ManagedV2rayConn); ok && c.ID() == id {
			c.Close()
			return false
		}
		return true
	})
}
