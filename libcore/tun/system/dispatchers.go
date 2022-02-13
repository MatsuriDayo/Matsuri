package system

import (
	"fmt"

	"golang.org/x/sys/unix"
	"gvisor.dev/gvisor/pkg/tcpip"
	"gvisor.dev/gvisor/pkg/tcpip/buffer"
	"gvisor.dev/gvisor/pkg/tcpip/link/rawfile"
	"gvisor.dev/gvisor/pkg/tcpip/stack"
)

// bufConfig defines the shape of the vectorised view used to read packets from the NIC.
var bufConfig = []int{128, 256, 256, 512, 1024, 2048, 4096, 8192, 16384, 32768}

type iovecBuffer struct {
	// views are the actual buffers that hold the packet contents.
	views []buffer.View

	// iovecs are initialized with base pointers/len of the corresponding
	// entries in the views defined above, except when GSO is enabled
	// (skipsVnetHdr) then the first iovec points to a buffer for the vnet header
	// which is stripped before the views are passed up the stack for further
	// processing.
	iovecs []unix.Iovec

	// sizes is an array of buffer sizes for the underlying views. sizes is
	// immutable.
	sizes []int
}

func newIovecBuffer(sizes []int) *iovecBuffer {
	b := &iovecBuffer{
		views: make([]buffer.View, len(sizes)),
		sizes: sizes,
	}
	b.iovecs = make([]unix.Iovec, len(b.views))
	return b
}

func (b *iovecBuffer) nextIovecs() []unix.Iovec {
	vnetHdrOff := 0
	for i := range b.views {
		if b.views[i] != nil {
			break
		}
		v := buffer.NewView(b.sizes[i])
		b.views[i] = v
		b.iovecs[i+vnetHdrOff] = unix.Iovec{Base: &v[0]}
		b.iovecs[i+vnetHdrOff].SetLen(len(v))
	}
	return b.iovecs
}

func (b *iovecBuffer) pullViews(n int) buffer.VectorisedView {
	var views []buffer.View
	c := 0
	for i, v := range b.views {
		c += len(v)
		if c >= n {
			b.views[i].CapLength(len(v) - (c - n))
			views = append([]buffer.View(nil), b.views[:i+1]...)
			break
		}
	}
	// Remove the first len(views) used views from the state.
	for i := range views {
		b.views[i] = nil
	}
	return buffer.NewVectorisedView(n, views)
}

// stopFd is an eventfd used to signal the stop of a dispatcher.
type stopFd struct {
	efd int
}

func newStopFd() (stopFd, error) {
	efd, err := unix.Eventfd(0, unix.EFD_NONBLOCK)
	if err != nil {
		return stopFd{efd: -1}, fmt.Errorf("failed to create eventfd: %w", err)
	}
	return stopFd{efd: efd}, nil
}

// stop writes to the eventfd and notifies the dispatcher to stop. It does not
// block.
func (s *stopFd) stop() {
	increment := []byte{1, 0, 0, 0, 0, 0, 0, 0}
	if n, err := unix.Write(s.efd, increment); n != len(increment) || err != nil {
		// There are two possible errors documented in eventfd(2) for writing:
		// 1. We are writing 8 bytes and not 0xffffffffffffff, thus no EINVAL.
		// 2. stop is only supposed to be called once, it can't reach the limit,
		// thus no EAGAIN.
		panic(fmt.Sprintf("write(efd) = (%d, %s), want (%d, nil)", n, err, len(increment)))
	}
}

// readVDispatcher uses readv() system call to read inbound packets and
// dispatches them.
type readVDispatcher struct {
	stopFd
	// fd is the file descriptor used to send and receive packets.
	fd int

	// e is the endpoint this dispatcher is attached to.
	e *SystemTun

	// buf is the iovec buffer that contains the packet contents.
	buf *iovecBuffer
}

func newReadVDispatcher(fd int, e *SystemTun) (*readVDispatcher, error) {
	stopFd, err := newStopFd()
	if err != nil {
		return nil, err
	}
	d := &readVDispatcher{
		stopFd: stopFd,
		fd:     fd,
		e:      e,
	}
	d.buf = newIovecBuffer(bufConfig)
	return d, nil
}

// dispatch reads one packet from the file descriptor and dispatches it.
func (d *readVDispatcher) dispatch() (bool, tcpip.Error) {
	n, err := rawfile.BlockingReadvUntilStopped(d.efd, d.fd, d.buf.nextIovecs())
	if n <= 0 || err != nil {
		return false, err
	}

	pkt := stack.NewPacketBuffer(stack.PacketBufferOptions{
		Data: d.buf.pullViews(n),
	})
	defer pkt.DecRef()
	d.e.deliverPacket(pkt)
	return true, nil
}

func (d *readVDispatcher) dispatchLoop() tcpip.Error {
	for {
		cont, err := d.dispatch()
		if err != nil || !cont {
			return err
		}
	}
}

func (d *readVDispatcher) writeRawPacket(vv buffer.VectorisedView) tcpip.Error {
	views := vv.Views()
	size := vv.Size()
	var iovecs []unix.Iovec
	for _, v := range views {
		iovecs = rawfile.AppendIovecFromBytes(iovecs, v, size)
	}
	return rawfile.NonBlockingWriteIovec(d.fd, iovecs)
}

func (d *readVDispatcher) writePacket(pkt *stack.PacketBuffer) tcpip.Error {
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
	return rawfile.NonBlockingWriteIovec(d.fd, iovecs)
}

func (d *readVDispatcher) writeBuffer(bytes []byte) tcpip.Error {
	return rawfile.NonBlockingWrite(d.fd, bytes)
}
