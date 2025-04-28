package option

import (
	"context"
	"fmt"
	"time"

	"github.com/xtls/xray-core/common/errors"
	"github.com/xtls/xray-core/common/net"
	"golang.org/x/time/rate"
	"gvisor.dev/gvisor/pkg/tcpip"
	"gvisor.dev/gvisor/pkg/tcpip/adapters/gonet"
	"gvisor.dev/gvisor/pkg/tcpip/header"
	"gvisor.dev/gvisor/pkg/tcpip/network/ipv4"
	"gvisor.dev/gvisor/pkg/tcpip/network/ipv6"
	"gvisor.dev/gvisor/pkg/tcpip/stack"
	"gvisor.dev/gvisor/pkg/tcpip/transport/tcp"
	"gvisor.dev/gvisor/pkg/tcpip/transport/udp"
	"gvisor.dev/gvisor/pkg/waiter"
)

type Option func(*stack.Stack) error

func WithDefaultTTL(ttl uint8) Option {
	return func(s *stack.Stack) error {
		opt := tcpip.DefaultTTLOption(ttl)
		if err := s.SetNetworkProtocolOption(ipv4.ProtocolNumber, &opt); err != nil {
			return fmt.Errorf("set ipv4 default TTL: %s", err)
		}
		if err := s.SetNetworkProtocolOption(ipv6.ProtocolNumber, &opt); err != nil {
			return fmt.Errorf("set ipv6 default TTL: %s", err)
		}
		return nil
	}
}

func WithForwarding(v bool) Option {
	return func(s *stack.Stack) error {
		if err := s.SetForwardingDefaultAndAllNICs(ipv4.ProtocolNumber, v); err != nil {
			return fmt.Errorf("set ipv4 forwarding: %s", err)
		}
		if err := s.SetForwardingDefaultAndAllNICs(ipv6.ProtocolNumber, v); err != nil {
			return fmt.Errorf("set ipv6 forwarding: %s", err)
		}
		return nil
	}
}

func WithICMPBurst(burst int) Option {
	return func(s *stack.Stack) error {
		s.SetICMPBurst(burst)
		return nil
	}
}

func WithICMPLimit(limit rate.Limit) Option {
	return func(s *stack.Stack) error {
		s.SetICMPLimit(limit)
		return nil
	}
}

func WithTCPSendBufferSize(size int) Option {
	return func(s *stack.Stack) error {
		sndOpt := tcpip.TCPSendBufferSizeRangeOption{Min: tcp.MinBufferSize, Default: size, Max: tcp.MaxBufferSize}
		if err := s.SetTransportProtocolOption(tcp.ProtocolNumber, &sndOpt); err != nil {
			return fmt.Errorf("set TCP send buffer size range: %s", err)
		}
		return nil
	}
}

func WithTCPSendBufferSizeRange(a, b, c int) Option {
	return func(s *stack.Stack) error {
		sndOpt := tcpip.TCPSendBufferSizeRangeOption{Min: a, Default: b, Max: c}
		if err := s.SetTransportProtocolOption(tcp.ProtocolNumber, &sndOpt); err != nil {
			return fmt.Errorf("set TCP send buffer size range: %s", err)
		}
		return nil
	}
}

func WithTCPReceiveBufferSize(size int) Option {
	return func(s *stack.Stack) error {
		rcvOpt := tcpip.TCPReceiveBufferSizeRangeOption{Min: tcp.MinBufferSize, Default: size, Max: tcp.MaxBufferSize}
		if err := s.SetTransportProtocolOption(tcp.ProtocolNumber, &rcvOpt); err != nil {
			return fmt.Errorf("set TCP receive buffer size range: %s", err)
		}
		return nil
	}
}

func WithTCPReceiveBufferSizeRange(a, b, c int) Option {
	return func(s *stack.Stack) error {
		rcvOpt := tcpip.TCPReceiveBufferSizeRangeOption{Min: a, Default: b, Max: c}
		if err := s.SetTransportProtocolOption(tcp.ProtocolNumber, &rcvOpt); err != nil {
			return fmt.Errorf("set TCP receive buffer size range: %s", err)
		}
		return nil
	}
}

func WithTCPCongestionControl(cc string) Option {
	return func(s *stack.Stack) error {
		opt := tcpip.CongestionControlOption(cc)
		if err := s.SetTransportProtocolOption(tcp.ProtocolNumber, &opt); err != nil {
			return fmt.Errorf("set TCP congestion control algorithm: %s", err)
		}
		return nil
	}
}

func WithTCPDelay(v bool) Option {
	return func(s *stack.Stack) error {
		opt := tcpip.TCPDelayEnabled(v)
		if err := s.SetTransportProtocolOption(tcp.ProtocolNumber, &opt); err != nil {
			return fmt.Errorf("set TCP delay: %s", err)
		}
		return nil
	}
}

func WithTCPModerateReceiveBuffer(v bool) Option {
	return func(s *stack.Stack) error {
		opt := tcpip.TCPModerateReceiveBufferOption(v)
		if err := s.SetTransportProtocolOption(tcp.ProtocolNumber, &opt); err != nil {
			return fmt.Errorf("set TCP moderate receive buffer: %s", err)
		}
		return nil
	}
}

func WithTCPSACKEnabled(v bool) Option {
	return func(s *stack.Stack) error {
		opt := tcpip.TCPSACKEnabled(v)
		if err := s.SetTransportProtocolOption(tcp.ProtocolNumber, &opt); err != nil {
			return fmt.Errorf("set TCP SACK: %s", err)
		}
		return nil
	}
}

func WithTCPRecovery(v tcpip.TCPRecovery) Option {
	return func(s *stack.Stack) error {
		if err := s.SetTransportProtocolOption(tcp.ProtocolNumber, &v); err != nil {
			return fmt.Errorf("set TCP Recovery: %s", err)
		}
		return nil
	}
}

func WithCreatingNIC(nicID tcpip.NICID, ep stack.LinkEndpoint) Option {
	return func(s *stack.Stack) error {
		if err := s.CreateNICWithOptions(nicID, ep,
			stack.NICOptions{
				Disabled: false,
				QDisc:    nil,
			}); err != nil {
			return fmt.Errorf("create NIC: %s", err)
		}
		return nil
	}
}

func WithPromiscuousMode(nicID tcpip.NICID, v bool) Option {
	return func(s *stack.Stack) error {
		if err := s.SetPromiscuousMode(nicID, v); err != nil {
			return fmt.Errorf("set promiscuous mode: %s", err)
		}
		return nil
	}
}

func WithSpoofing(nicID tcpip.NICID, v bool) Option {
	return func(s *stack.Stack) error {
		if err := s.SetSpoofing(nicID, v); err != nil {
			return fmt.Errorf("set spoofing: %s", err)
		}
		return nil
	}
}

func WithRouteTable(nicID tcpip.NICID) Option {
	return func(s *stack.Stack) error {
		s.SetRouteTable([]tcpip.Route{
			{
				Destination: header.IPv4EmptySubnet,
				NIC:         nicID,
			},
			{
				Destination: header.IPv6EmptySubnet,
				NIC:         nicID,
			},
		})
		return nil
	}
}

func WithTransportHandler(handle func(src, dst net.Destination, conn net.Conn)) Option {
	return func(s *stack.Stack) error {
		tcpForwarder := tcp.NewForwarder(s, 0, 65535, func(r *tcp.ForwarderRequest) {
			go func(r *tcp.ForwarderRequest) {
				var (
					wq waiter.Queue
					id = r.ID()
				)
				ep, err := r.CreateEndpoint(&wq)
				if err != nil {
					errors.LogError(context.Background(), err.String())
					r.Complete(true)
					return
				}
				r.Complete(false)
				defer ep.Close()
				ep.SocketOptions().SetKeepAlive(true)
				handle(
					net.TCPDestination(net.IPAddress(id.RemoteAddress.AsSlice()), net.Port(id.RemotePort)),
					net.TCPDestination(net.IPAddress(id.LocalAddress.AsSlice()), net.Port(id.LocalPort)),
					gonet.NewTCPConn(&wq, ep),
				)
			}(r)
		})
		s.SetTransportProtocolHandler(tcp.ProtocolNumber, tcpForwarder.HandlePacket)

		udpForwarder := udp.NewForwarder(s, func(r *udp.ForwarderRequest) {
			go func(r *udp.ForwarderRequest) {
				var (
					wq waiter.Queue
					id = r.ID()
				)
				ep, err := r.CreateEndpoint(&wq)
				if err != nil {
					errors.LogError(context.Background(), err.String())
					return
				}
				defer ep.Close()
				ep.SocketOptions().SetLinger(tcpip.LingerOption{
					Enabled: true,
					Timeout: 15 * time.Second,
				})
				handle(
					net.UDPDestination(net.IPAddress(id.RemoteAddress.AsSlice()), net.Port(id.RemotePort)),
					net.UDPDestination(net.IPAddress(id.LocalAddress.AsSlice()), net.Port(id.LocalPort)),
					gonet.NewUDPConn(&wq, ep),
				)
			}(r)
		})
		s.SetTransportProtocolHandler(udp.ProtocolNumber, udpForwarder.HandlePacket)
		return nil
	}
}
