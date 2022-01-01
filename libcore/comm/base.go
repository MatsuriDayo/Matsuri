package comm

import (
	"io"

	"github.com/v2fly/v2ray-core/v5/common"
)

type closerWrapper struct {
	closer func()
}

func (c closerWrapper) Close() error {
	c.closer()
	return nil
}

func Closer(closer func()) io.Closer {
	return closerWrapper{closer}
}

func CloseIgnore(closer ...interface{}) {
	for _, c := range closer {
		if c == nil {
			continue
		}
		if ia, ok := c.(common.Interruptible); ok {
			ia.Interrupt()
		} else if ca, ok := c.(common.Closable); ok {
			_ = ca.Close()
		}
	}
}
