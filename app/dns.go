//go:build linux

package app

import (
	"context"
	"net"
	"syscall"
	"time"

	"vpn/app/ohos"
)

func init() {
	net.DefaultResolver = &net.Resolver{
		PreferGo: true,
		Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
			dialer := &net.Dialer{
				Timeout: time.Second * 16,
			}
			dialer.Control = func(network, address string, c syscall.RawConn) error {
				var innerErr error
				err := c.Control(func(fd uintptr) {
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
			return dialer.DialContext(ctx, network, address)
		},
	}
}
