package protect

import (
	"context"
	"net"
	"strings"

	v2rayNet "github.com/v2fly/v2ray-core/v5/common/net"
)

func (dialer ProtectedDialer) DialContext(ctx context.Context, network, addr string) (net.Conn, error) {
	var prefix = "tcp:"
	if strings.Contains(network, "udp") {
		prefix = "udp:"
	} else if strings.Contains(network, "unix") {
		prefix = "unix:"
	}
	dest, _ := v2rayNet.ParseDestination(prefix + addr)
	return dialer.Dial(ctx, nil, dest, nil)
}
