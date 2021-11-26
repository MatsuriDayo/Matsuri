package libcore

import (
	"errors"
	"github.com/sagernet/sagerconnect/api"
	"github.com/sirupsen/logrus"
	"net"
	"strings"
	"sync"
)

type ApiInstance struct {
	access     sync.Mutex
	deviceName string
	socksPort  int32
	dnsPort    int32
	debug      bool
	bypassLan  bool

	conn    *net.UDPConn
	started bool
}

func NewApiInstance(deviceName string, socksPort int32, dnsPort int32, debug bool, bypassLan bool) *ApiInstance {
	return &ApiInstance{
		deviceName: deviceName,
		socksPort:  socksPort,
		dnsPort:    dnsPort,
		debug:      debug,
		bypassLan:  bypassLan,
	}
}

func (i *ApiInstance) Start() (err error) {
	i.access.Lock()
	defer i.access.Unlock()

	if i.started {
		return errors.New("already started")
	}

	i.conn, err = net.ListenUDP("udp4", &net.UDPAddr{
		IP:   net.IPv4zero,
		Port: 11451,
	})
	if err != nil {
		return err
	}

	i.started = true
	go i.loop()

	return nil
}

func (i *ApiInstance) Close() {
	i.access.Lock()
	defer i.access.Unlock()

	if i.started {
		i.started = false
		closeIgnore(i.conn)
	}
}

func (i *ApiInstance) loop() {
	buffer := make([]byte, 2048)
	for i.started {
		length, addr, err := i.conn.ReadFrom(buffer)
		if err != nil {
			continue
		}
		query, err := api.ParseQuery(buffer[:length])
		if err != nil {
			if err != nil && strings.Contains(err.Error(), "upgrade") {
				message, err := api.MakeResponse(&api.Response{Version: api.Version, DeviceName: "", SocksPort: 0, DnsPort: 0, Debug: false, BypassLan: false})
				if err != nil {
					logrus.Warnf("api: make response error: %v", err)
					continue
				}

				_, err = i.conn.WriteTo(message, addr)
				if err != nil {
					logrus.Warnf("api: send response error: %v", err)
					continue
				}

			}
			logrus.Warnf("api: parse error: %v", err)
			continue
		}

		logrus.Infof("api: new query from %s (%s)", query.DeviceName, addr.String())

		response := api.Response{Version: api.Version, DeviceName: i.deviceName, SocksPort: uint16(i.socksPort), DnsPort: uint16(i.dnsPort), Debug: i.debug, BypassLan: i.bypassLan}
		message, err := api.MakeResponse(&response)
		if err != nil {
			logrus.Warnf("api: make response error: %v", err)
			continue
		}

		_, err = i.conn.WriteTo(message, addr)
		if err != nil {
			logrus.Warnf("api: send response error: %v", err)
			continue
		}
	}
}
