package libcore

import (
	"context"
	"fmt"
	"github.com/pkg/errors"
	core "github.com/v2fly/v2ray-core/v4"
	v2rayNet "github.com/v2fly/v2ray-core/v4/common/net"
	"github.com/v2fly/v2ray-core/v4/common/session"
	"net"
	"net/http"
	"time"
)

func urlTest(dialContext func(ctx context.Context, network, addr string) (net.Conn, error), link string, timeout int32) (int32, error) {
	transport := &http.Transport{
		TLSHandshakeTimeout: time.Duration(timeout) * time.Millisecond,
		DisableKeepAlives:   true,
		DialContext:         dialContext,
	}
	req, err := http.NewRequestWithContext(context.Background(), "GET", link, nil)
	req.Header.Set("User-Agent", "curl/7.74.0")
	if err != nil {
		return 0, errors.WithMessage(err, "create get request")
	}
	start := time.Now()
	resp, err := (&http.Client{
		Transport: transport,
		Timeout:   time.Duration(timeout) * time.Millisecond,
	}).Do(req)
	if err == nil && resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK {
		err = fmt.Errorf("unexcpted response status: %d", resp.StatusCode)
	}
	if err != nil {
		return 0, err
	}
	return int32(time.Since(start).Milliseconds()), nil
}

func UrlTestV2ray(instance *V2RayInstance, inbound string, link string, timeout int32) (int32, error) {
	return urlTest(func(ctx context.Context, network, addr string) (net.Conn, error) {
		dest, err := v2rayNet.ParseDestination(fmt.Sprintf("%s:%s", network, addr))
		if err != nil {
			return nil, err
		}
		if inbound != "" {
			ctx = session.ContextWithInbound(ctx, &session.Inbound{Tag: inbound})
		}
		return core.Dial(ctx, instance.core, dest)
	}, link, timeout)
}

func UrlTestClashBased(instance *ClashBasedInstance, link string, timeout int32) (int32, error) {
	return urlTest(func(ctx context.Context, network, addr string) (net.Conn, error) {
		dest, err := addrToMetadata(addr)
		if err != nil {
			return nil, err
		}
		dest.NetWork = networkForClash(network)
		return instance.out.DialContext(ctx, dest)
	}, link, timeout)
}
