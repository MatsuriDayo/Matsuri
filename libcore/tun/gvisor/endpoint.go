package gvisor

import (
	"sync"

	"golang.org/x/sys/unix"
	"gvisor.dev/gvisor/pkg/tcpip"
	"gvisor.dev/gvisor/pkg/tcpip/header"
	"gvisor.dev/gvisor/pkg/tcpip/link/rawfile"
	"gvisor.dev/gvisor/pkg/tcpip/stack"
)

var _ stack.InjectableLinkEndpoint = (*rwEndpoint)(nil)

// rwEndpoint implements the interface of stack.LinkEndpoint from io.ReadWriter.
type rwEndpoint struct {
	fd int

	// mtu (maximum transmission unit) is the maximum size of a packet.
	mtu uint32
	wg  sync.WaitGroup

	inbound    *readVDispatcher
	dispatcher stack.NetworkDispatcher
}

func newRwEndpoint(dev int32, mtu int32) (*rwEndpoint, error) {
	e := &rwEndpoint{
		fd:  int(dev),
		mtu: uint32(mtu),
	}
	i, err := newReadVDispatcher(e.fd, e)
	if err != nil {
		return nil, err
	}
	e.inbound = i
	return e, nil
}

func (e *rwEndpoint) InjectInbound(networkProtocol tcpip.NetworkProtocolNumber, pkt *stack.PacketBuffer) {
	go e.dispatcher.DeliverNetworkPacket(networkProtocol, pkt)
}

func (e *rwEndpoint) InjectOutbound(dest tcpip.Address, packet []byte) tcpip.Error {
	return rawfile.NonBlockingWrite(e.fd, packet)
}

// Attach launches the goroutine that reads packets from io.ReadWriter and
// dispatches them via the provided dispatcher.
func (e *rwEndpoint) Attach(dispatcher stack.NetworkDispatcher) {
	if dispatcher == nil && e.dispatcher != nil {
		e.inbound.stop()
		e.Wait()
		e.dispatcher = nil
		return
	}
	if dispatcher != nil && e.dispatcher == nil {
		e.dispatcher = dispatcher
		e.wg.Add(1)
		go func() {
			e.dispatchLoop(e.inbound)
			e.wg.Done()
		}()
	}
}

// IsAttached implements stack.LinkEndpoint.IsAttached.
func (e *rwEndpoint) IsAttached() bool {
	return e.dispatcher != nil
}

// dispatchLoop reads packets from the file descriptor in a loop and dispatches
// them to the network stack.
func (e *rwEndpoint) dispatchLoop(inboundDispatcher *readVDispatcher) tcpip.Error {
	for {
		cont, err := inboundDispatcher.dispatch()
		if err != nil || !cont {
			return err
		}
	}
}

// WritePackets writes packets back into io.ReadWriter.
func (e *rwEndpoint) WritePackets(pkts stack.PacketBufferList) (int, tcpip.Error) {
	// Preallocate to avoid repeated reallocation as we append to batch.
	// batchSz is 47 because when SWGSO is in use then a single 65KB TCP
	// segment can get split into 46 segments of 1420 bytes and a single 216
	// byte segment.
	const batchSz = 47
	batch := make([]unix.Iovec, 0, batchSz)
	for pkt := pkts.Front(); pkt != nil; pkt = pkt.Next() {
		views := pkt.Views()
		for _, v := range views {
			batch = rawfile.AppendIovecFromBytes(batch, v, len(views))
		}
	}
	err := rawfile.NonBlockingWriteIovec(e.fd, batch)
	if err != nil {
		return 0, err
	}
	return pkts.Len(), nil
}

// MTU implements stack.LinkEndpoint.MTU.
func (e *rwEndpoint) MTU() uint32 {
	return e.mtu
}

// Capabilities implements stack.LinkEndpoint.Capabilities.
func (e *rwEndpoint) Capabilities() stack.LinkEndpointCapabilities {
	return stack.CapabilityNone
}

// MaxHeaderLength returns the maximum size of the link layer header. Given it
// doesn't have a header, it just returns 0.
func (*rwEndpoint) MaxHeaderLength() uint16 {
	return 0
}

// LinkAddress returns the link address of this endpoint.
func (*rwEndpoint) LinkAddress() tcpip.LinkAddress {
	return ""
}

// ARPHardwareType implements stack.LinkEndpoint.ARPHardwareType.
func (*rwEndpoint) ARPHardwareType() header.ARPHardwareType {
	return header.ARPHardwareNone
}

func (e *rwEndpoint) AddHeader(*stack.PacketBuffer) {
}

// Wait implements stack.LinkEndpoint.Wait.
func (e *rwEndpoint) Wait() {
	e.wg.Wait()
}
