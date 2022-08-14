package libcore

import (
	"context"
	"fmt"
	"math/rand"
	"net/http"
	"strings"
	"time"

	"github.com/v2fly/v2ray-core/v5/common/net"
	"github.com/v2fly/v2ray-core/v5/common/session"
)

func UrlTestV2ray(instance *V2RayInstance, inbound string, link string, timeout int32) (int32, error) {
	transport := &http.Transport{
		TLSHandshakeTimeout: time.Duration(timeout) * time.Millisecond,
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			// I believe the server...
			dest, err := net.ParseDestination(fmt.Sprintf("%s:%s", network, addr))
			if err != nil {
				return nil, err
			}
			if inbound != "" {
				ctx = session.ContextWithInbound(ctx, &session.Inbound{Tag: inbound})
			}
			return instance.DialContext(ctx, dest)
		},
	}

	// Keep alive, use one connection
	client := &http.Client{
		Transport: transport,
		Timeout:   time.Duration(timeout) * time.Millisecond,
	}

	// Test handshake time
	var time_start time.Time
	var times = 1
	var rtt_times = 1

	// Test RTT "true delay"
	if link2 := strings.TrimLeft(link, "true"); link != link2 {
		link = link2
		times = 3
		rtt_times = 2
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeout)*time.Millisecond)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, "GET", link, nil)
	req.Header.Set("User-Agent", fmt.Sprintf("curl/7.%d.%d", rand.Int()%84, rand.Int()%2))
	if err != nil {
		return 0, err
	}

	for i := 0; i < times; i++ {
		if i == 1 || times == 1 {
			time_start = time.Now()
		}

		resp, err := client.Do(req)
		if err != nil {
			return 0, err
		}
		resp.Body.Close()
	}

	return int32(time.Since(time_start).Milliseconds() / int64(rtt_times)), nil
}
