//go:build linux

package app

import (
	"context"
	"syscall"
	"time"

	"vpn/app/ohos"

	"github.com/xtls/xray-core/common/errors"
	"github.com/xtls/xray-core/common/net"
	"github.com/xtls/xray-core/transport/internet"
)

var _ internet.SystemDialer = (*OHSystemDialer)(nil)

type OHSystemDialer struct{}

func (d *OHSystemDialer) Dial(ctx context.Context, src net.Address, dest net.Destination, _ *internet.SocketConfig) (net.Conn, error) {
	errors.LogDebug(ctx, "dialing to "+dest.String())
	switch dest.Network {
	case net.Network_TCP:
		goStdKeepAlive := time.Duration(0)
		dialer := &net.Dialer{
			Timeout:   time.Second * 16,
			LocalAddr: d.ResolveSrcAddr(dest.Network, src),
			KeepAlive: goStdKeepAlive,
		}
		dialer.Control = func(network, address string, c syscall.RawConn) error {
			return d.BindToDefaultDevice(c)
		}
		return dialer.DialContext(ctx, dest.Network.SystemString(), dest.NetAddr())
	case net.Network_UDP:
		srcAddr := d.ResolveSrcAddr(net.Network_UDP, src)
		if srcAddr == nil {
			srcAddr = &net.UDPAddr{
				IP:   []byte{0, 0, 0, 0},
				Port: 0,
			}
		}
		var lc net.ListenConfig
		lc.Control = func(network, address string, c syscall.RawConn) error {
			return d.BindToDefaultDevice(c)
		}
		packetConn, err := lc.ListenPacket(ctx, srcAddr.Network(), srcAddr.String())
		if err != nil {
			return nil, err
		}
		destAddr, err := net.ResolveUDPAddr("udp", dest.NetAddr())
		if err != nil {
			return nil, err
		}
		return &internet.PacketConnWrapper{
			Conn: packetConn,
			Dest: destAddr,
		}, nil
	default:
		return nil, errors.New("unknown network")
	}
}

func (d *OHSystemDialer) DestIpAddress() net.IP {
	return nil
}

func (d *OHSystemDialer) ResolveSrcAddr(network net.Network, src net.Address) net.Addr {
	if src == nil || src == net.AnyIP {
		return nil
	}
	if network == net.Network_TCP {
		return &net.TCPAddr{
			IP:   src.IP(),
			Port: 0,
		}
	}
	return &net.UDPAddr{
		IP:   src.IP(),
		Port: 0,
	}
}

func (d *OHSystemDialer) BindToDefaultDevice(conn syscall.RawConn) error {
	var innerErr error
	err := conn.Control(func(fd uintptr) {
		if device, err := ohos.MustGetPlatformSupport().GetDefaultNetInterfaceName(); err != nil {
			innerErr = err
		} else {
			innerErr = syscall.BindToDevice(int(fd), device)
		}
	})
	if err == nil {
		return innerErr
	}
	return err
}

func init() {
	internet.UseAlternativeSystemDialer(&OHSystemDialer{})
}
