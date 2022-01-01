package tun2socket

import (
	"io"

	"libcore/tun/tun2socket/nat"
)

type Tun2Socket struct {
	tcp *nat.TCP
	udp *nat.UDP
}

//noinspection GoUnusedExportedFunction
func StartTun2Socket(device io.ReadWriteCloser) (*Tun2Socket, error) {
	tcp, udp, err := nat.Start(device)
	if err != nil {
		return nil, err
	}

	return &Tun2Socket{
		tcp: tcp,
		udp: udp,
	}, nil
}

func (t *Tun2Socket) Stop() {
	_ = t.tcp.Close()
	_ = t.udp.Close()

	// Note: SagerNet close tun fd at VPNService.kt
}

func (t *Tun2Socket) TCP() *nat.TCP {
	return t.tcp
}

func (t *Tun2Socket) UDP() *nat.UDP {
	return t.udp
}
