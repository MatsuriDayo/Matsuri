package gvisor

import (
	"golang.org/x/sys/unix"
	"gvisor.dev/gvisor/pkg/tcpip"
	"gvisor.dev/gvisor/pkg/tcpip/header"
	"gvisor.dev/gvisor/pkg/tcpip/link/rawfile"
	"gvisor.dev/gvisor/pkg/tcpip/stack"
	"sync"
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
	go e.dispatcher.DeliverNetworkPacket("", "", networkProtocol, pkt)
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

// WritePacket writes packet back into io.ReadWriter.
func (e *rwEndpoint) WritePacket(_ stack.RouteInfo, _ tcpip.NetworkProtocolNumber, pkt *stack.PacketBuffer) tcpip.Error {
	return e.writePacket(pkt)
}

func (e *rwEndpoint) writePacket(pkt *stack.PacketBuffer) tcpip.Error {
	views := pkt.Views()
	numIovecs := len(views)
	if numIovecs > rawfile.MaxIovs {
		numIovecs = rawfile.MaxIovs
	}
	// Allocate small iovec arrays on the stack.
	var iovecsArr [8]unix.Iovec
	iovecs := iovecsArr[:0]
	if numIovecs > len(iovecsArr) {
		iovecs = make([]unix.Iovec, 0, numIovecs)
	}
	for _, v := range views {
		iovecs = rawfile.AppendIovecFromBytes(iovecs, v, numIovecs)
	}
	return rawfile.NonBlockingWriteIovec(e.fd, iovecs)
}

func (e *rwEndpoint) sendBatch(batchFD int, pkts []*stack.PacketBuffer) (int, tcpip.Error) {
	// Send a batch of packets through batchFD.
	mmsgHdrsStorage := make([]rawfile.MMsgHdr, 0, len(pkts))
	packets := 0
	for packets < len(pkts) {
		mmsgHdrs := mmsgHdrsStorage
		batch := pkts[packets:]
		for _, pkt := range batch {
			views := pkt.Views()
			numIovecs := len(views)
			if numIovecs > rawfile.MaxIovs {
				numIovecs = rawfile.MaxIovs
			}

			// We can't easily allocate iovec arrays on the stack here since
			// they will escape this loop iteration via mmsgHdrs.
			iovecs := make([]unix.Iovec, 0, numIovecs)
			for _, v := range views {
				iovecs = rawfile.AppendIovecFromBytes(iovecs, v, numIovecs)
			}

			var mmsgHdr rawfile.MMsgHdr
			mmsgHdr.Msg.Iov = &iovecs[0]
			mmsgHdr.Msg.SetIovlen(len(iovecs))
			mmsgHdrs = append(mmsgHdrs, mmsgHdr)
		}

		if len(mmsgHdrs) == 0 {
			// We can't fit batch[0] into a mmsghdr while staying under
			// e.maxSyscallHeaderBytes. Use WritePacket, which will avoid the
			// mmsghdr (by using writev) and re-buffer iovecs more aggressively
			// if necessary (by using e.writevMaxIovs instead of
			// rawfile.MaxIovs).
			pkt := batch[0]
			if err := e.WritePacket(pkt.EgressRoute, pkt.NetworkProtocolNumber, pkt); err != nil {
				return packets, err
			}
			packets++
		} else {
			for len(mmsgHdrs) > 0 {
				sent, err := rawfile.NonBlockingSendMMsg(batchFD, mmsgHdrs)
				if err != nil {
					return packets, err
				}
				packets += sent
				mmsgHdrs = mmsgHdrs[sent:]
			}
		}
	}

	return packets, nil
}

// WritePackets writes packets back into io.ReadWriter.
func (e *rwEndpoint) WritePackets(_ stack.RouteInfo, pkts stack.PacketBufferList, _ tcpip.NetworkProtocolNumber) (int, tcpip.Error) {
	const batchSz = 47
	batch := make([]*stack.PacketBuffer, 0, batchSz)
	for pkt := pkts.Front(); pkt != nil; pkt = pkt.Next() {
		batch = append(batch, pkt)
	}
	return e.sendBatch(e.fd, batch)
}

func (e *rwEndpoint) WriteRawPacket(packetBuffer *stack.PacketBuffer) tcpip.Error {
	return e.writePacket(packetBuffer)
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

// AddHeader implements stack.LinkEndpoint.AddHeader.
func (e *rwEndpoint) AddHeader(tcpip.LinkAddress, tcpip.LinkAddress, tcpip.NetworkProtocolNumber, *stack.PacketBuffer) {
}

// Wait implements stack.LinkEndpoint.Wait.
func (e *rwEndpoint) Wait() {
	e.wg.Wait()
}
