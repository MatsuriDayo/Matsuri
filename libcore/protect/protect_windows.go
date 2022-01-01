package protect

import (
	"context"
	"net"

	v2rayNet "github.com/v2fly/v2ray-core/v5/common/net"
	"github.com/v2fly/v2ray-core/v5/transport/internet"
)

func (dialer ProtectedDialer) dial(ctx context.Context, source v2rayNet.Address, destination v2rayNet.Destination, sockopt *internet.SocketConfig) (conn net.Conn, err error) {
	return v2rayDefaultDialer.Dial(ctx, source, destination, sockopt)
}
