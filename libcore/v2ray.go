package libcore

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"sort"
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
	if outdated != "" {
		return errors.New(outdated)
	}

	instance.access.Lock()
	defer instance.access.Unlock()

	config, err := serial.LoadJSONConfig(strings.NewReader(content))
	if err != nil {
		log.Println(content, err.Error())
		return err
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
	instance.access.Lock()
	defer instance.access.Unlock()
	if instance.started {
		if !instance.ForTest {
			atomic.StorePointer(&v2rayDNSClient, unsafe.Pointer(nil))
		}
		nekoutils.ConnectionLog_V2Ray.ResetConnections(uintptr(unsafe.Pointer(instance.Core)))
		nekoutils.ConnectionPool_V2Ray.ResetConnections(uintptr(unsafe.Pointer(instance.Core)))
		nekoutils.ConnectionPool_System.ResetConnections(uintptr(unsafe.Pointer(instance.Core)))
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

func ResetAllConnections(system bool) {
	if system {
		nekoutils.ConnectionPool_System.ResetConnections(0)
	} else {
		nekoutils.ConnectionPool_V2Ray.ResetConnections(0)
		nekoutils.ConnectionLog_V2Ray.ResetConnections(0)
	}
}

func ListV2rayConnections() string {
	list2 := make([]interface{}, 0)

	rangeMap := func(m *sync.Map) []interface{} {
		vs := make(map[uint32]interface{}, 0)
		ks := make([]uint32, 0)

		m.Range(func(key interface{}, value interface{}) bool {
			if k, ok := key.(uint32); ok {
				vs[k] = value
				ks = append(ks, k)
			}
			return true
		})

		sort.Slice(ks, func(i, j int) bool { return ks[i] > ks[j] })

		ret := make([]interface{}, 0)
		for _, id := range ks {
			ret = append(ret, vs[id])
		}
		return ret
	}

	addToList := func(list interface{}) {
		for i, c := range list.([]interface{}) {
			if i >= 100 { // too much
				return
			}
			if c2, ok := c.(*nekoutils.ManagedV2rayConn); ok {
				if c2.Tag == "dns-out" || c2.Tag == "direct" {
					continue
				}
				item := &struct {
					ID    uint32
					Dest  string
					RDest string
					Uid   uint32
					Start int64
					End   int64
					Tag   string
				}{
					c2.ID(),
					c2.Dest,
					c2.RouteDest,
					c2.InboundUid,
					c2.StartTime,
					c2.EndTime,
					c2.Tag,
				}
				list2 = append(list2, item)
			}
		}
	}

	addToList(rangeMap(&nekoutils.ConnectionPool_V2Ray.Map))
	addToList(rangeMap(&nekoutils.ConnectionLog_V2Ray.Map))

	b, _ := json.Marshal(&list2)
	return string(b)
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
