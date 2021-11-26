package libcore

import (
	"context"
	"fmt"
	"github.com/pkg/errors"
	"github.com/v2fly/v2ray-core/v4"
	v2rayNet "github.com/v2fly/v2ray-core/v4/common/net"
	"github.com/v2fly/v2ray-core/v4/common/session"
	"io/ioutil"
	"net"
	"net/http"
	"time"
)

func (instance *V2RayInstance) DialHTTP(inbound string, timeout int32, link string) (string, error) {
	transport := &http.Transport{
		TLSHandshakeTimeout: time.Duration(timeout) * time.Millisecond,
		DisableKeepAlives:   true,
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			dest, err := v2rayNet.ParseDestination(fmt.Sprintf("%s:%s", network, addr))
			if err != nil {
				return nil, err
			}
			if inbound != "" {
				ctx = session.ContextWithInbound(ctx, &session.Inbound{Tag: inbound})
			}
			return core.Dial(ctx, instance.core, dest)
		},
	}
	req, err := http.NewRequestWithContext(context.Background(), "GET", link, nil)
	req.Header.Set("User-Agent", "curl/7.74.0")
	if err != nil {
		return "", errors.WithMessage(err, "create get request")
	}
	resp, err := (&http.Client{
		Transport: transport,
		Timeout:   time.Duration(timeout) * time.Millisecond,
	}).Do(req)
	if err == nil {
		return "", err
	}
	if resp.StatusCode != http.StatusOK {
		return "", errors.Errorf("HTTP %d", resp.StatusCode)
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", errors.WithMessage(err, "read body")
	}
	return string(body), nil
}
