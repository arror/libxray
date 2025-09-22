package endpoint

import (
	"context"
	"sync"
	"syscall"
	"unsafe"

	"gvisor.dev/gvisor/pkg/buffer"
	"gvisor.dev/gvisor/pkg/tcpip"
	"gvisor.dev/gvisor/pkg/tcpip/header"
	"gvisor.dev/gvisor/pkg/tcpip/link/channel"
	"gvisor.dev/gvisor/pkg/tcpip/stack"
)

type Endpoint struct {
	*channel.Endpoint
	fd   int
	mtu  int
	once sync.Once
	wg   sync.WaitGroup
	pool sync.Pool
}

func New(fd, mtu int) *Endpoint {
	return &Endpoint{
		Endpoint: channel.New(1<<10, uint32(mtu), ""),
		fd:       fd,
		mtu:      mtu,
		pool: sync.Pool{
			New: func() any {
				return make([]syscall.Iovec, 0, 64)
			},
		},
	}
}

func (e *Endpoint) Attach(dispatcher stack.NetworkDispatcher) {
	e.Endpoint.Attach(dispatcher)
	e.once.Do(func() {
		ctx, cancel := context.WithCancel(context.Background())
		e.wg.Add(2)
		go func() {
			e.outboundLoop(ctx)
			e.wg.Done()
		}()
		go func() {
			e.dispatchLoop(cancel)
			e.wg.Done()
		}()
	})
}

func (e *Endpoint) Wait() {
	e.wg.Wait()
}

func (e *Endpoint) dispatchLoop(cancel context.CancelFunc) {
	defer cancel()
	for {
		data := make([]byte, e.mtu)
		n, err := e.read(e.fd, data)
		if err != nil {
			break
		}
		if n == 0 || n > e.mtu {
			continue
		}
		if !e.IsAttached() {
			continue
		}
		pkt := stack.NewPacketBuffer(stack.PacketBufferOptions{
			Payload: buffer.MakeWithData(data[:n]),
		})
		switch header.IPVersion(data) {
		case header.IPv4Version:
			e.InjectInbound(header.IPv4ProtocolNumber, pkt)
		case header.IPv6Version:
			e.InjectInbound(header.IPv6ProtocolNumber, pkt)
		}
		pkt.DecRef()
	}
}

func (e *Endpoint) outboundLoop(ctx context.Context) {
	for {
		pkt := e.ReadContext(ctx)
		if pkt == nil {
			break
		}
		e.writePacket(pkt)
	}
}

func (e *Endpoint) writePacket(pkt *stack.PacketBuffer) tcpip.Error {
	defer pkt.DecRef()
	if _, err := e.write(e.fd, pkt.AsSlices()); err != nil {
		return &tcpip.ErrInvalidEndpointState{}
	}
	return nil
}

func (e *Endpoint) read(fd int, p []byte) (n int, err error) {
	return syscall.Read(fd, p)
}

func (e *Endpoint) write(fd int, ps [][]byte) (n int, err error) {
	count := len(ps)
	if count == 0 {
		return 0, nil
	}
	iovs := e.pool.Get().([]syscall.Iovec)
	iovs = iovs[:count]
	for i, p := range ps {
		if len(p) > 0 {
			iovs[i].Base = &p[0]
			iovs[i].Len = uint64(len(p))
		}
	}
	r, _, err := syscall.Syscall(syscall.SYS_WRITEV,
		uintptr(fd),
		uintptr(unsafe.Pointer(&iovs[0])),
		uintptr(count))

	if err != syscall.Errno(0) {
		return int(r), err
	}
	return int(r), nil
}
