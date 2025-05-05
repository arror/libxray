package tun

import (
	"context"

	"vpn/app/tun/endpoint"
	"vpn/app/tun/option"

	"github.com/xtls/xray-core/common"
	"github.com/xtls/xray-core/common/buf"
	"github.com/xtls/xray-core/common/errors"
	"github.com/xtls/xray-core/common/log"
	"github.com/xtls/xray-core/common/net"
	"github.com/xtls/xray-core/common/session"
	"github.com/xtls/xray-core/common/signal"
	"github.com/xtls/xray-core/common/task"
	"github.com/xtls/xray-core/core"
	"github.com/xtls/xray-core/features"
	"github.com/xtls/xray-core/features/policy"
	"github.com/xtls/xray-core/features/routing"

	"gvisor.dev/gvisor/pkg/tcpip"
	"gvisor.dev/gvisor/pkg/tcpip/network/ipv4"
	"gvisor.dev/gvisor/pkg/tcpip/network/ipv6"
	"gvisor.dev/gvisor/pkg/tcpip/stack"
	"gvisor.dev/gvisor/pkg/tcpip/transport/icmp"
	"gvisor.dev/gvisor/pkg/tcpip/transport/tcp"
	"gvisor.dev/gvisor/pkg/tcpip/transport/udp"
)

type SniffingConfig struct {
	Enabled                        bool     `json:"enabled"`
	OverrideDestinationForProtocol []string `json:"overrideDestinationForProtocol"`
	MetadataOnly                   bool     `json:"metadataOnly"`
	RouteOnly                      bool     `json:"routeOnly"`
}

type Config struct {
	Tag      string         `json:"tag"`
	Fd       int            `json:"fd"`
	MTU      int            `json:"mtu"`
	Sniffing SniffingConfig `json:"sniffing"`
}

func init() {
	common.Must(common.RegisterConfig((*Config)(nil), func(ctx context.Context, cfg any) (any, error) {
		return New(ctx, cfg.(*Config))
	}))
}

type Tun struct {
	ctx           context.Context
	stack         *stack.Stack
	fd            int
	mtu           int
	dispatcher    routing.Dispatcher
	policyManager policy.Manager
}

var _ features.Feature = (*Tun)(nil)

func New(ctx context.Context, cfg *Config) (*Tun, error) {
	v := core.MustFromContext(ctx)
	ctx = session.ContextWithInbound(ctx, &session.Inbound{
		Tag: cfg.Tag,
	})
	ctx = session.ContextWithContent(ctx, &session.Content{
		SniffingRequest: session.SniffingRequest{
			Enabled:                        cfg.Sniffing.Enabled,
			OverrideDestinationForProtocol: cfg.Sniffing.OverrideDestinationForProtocol,
			MetadataOnly:                   cfg.Sniffing.MetadataOnly,
			RouteOnly:                      cfg.Sniffing.RouteOnly,
		},
	})
	return &Tun{
		ctx:           ctx,
		fd:            cfg.Fd,
		mtu:           cfg.MTU,
		dispatcher:    v.GetFeature(routing.DispatcherType()).(routing.Dispatcher),
		policyManager: v.GetFeature(policy.ManagerType()).(policy.Manager),
	}, nil
}

func (t *Tun) Type() any {
	return (*Tun)(nil)
}

func (t *Tun) Start() error {
	ep := endpoint.New(t.fd, t.mtu)
	t.stack = stack.New(stack.Options{
		NetworkProtocols: []stack.NetworkProtocolFactory{
			ipv4.NewProtocol,
			ipv6.NewProtocol,
		},
		TransportProtocols: []stack.TransportProtocolFactory{
			tcp.NewProtocol,
			udp.NewProtocol,
			icmp.NewProtocol4,
			icmp.NewProtocol6,
		},
	})
	var nicID tcpip.NICID = 1
	opts := []option.Option{
		option.WithDefaultTTL(64),
		option.WithForwarding(true),
		option.WithICMPBurst(50),
		option.WithICMPLimit(1000),
		option.WithTCPSendBufferSizeRange(tcp.MinBufferSize, tcp.DefaultSendBufferSize, tcp.MaxBufferSize),
		option.WithTCPReceiveBufferSizeRange(tcp.MinBufferSize, tcp.DefaultReceiveBufferSize, tcp.MaxBufferSize),
		option.WithTCPCongestionControl("cubic"),
		option.WithTCPDelay(false),
		option.WithTCPModerateReceiveBuffer(false),
		option.WithTCPSACKEnabled(true),
		option.WithTCPRecovery(tcpip.TCPRACKLossDetection),
		option.WithCreatingNIC(nicID, ep),
		option.WithPromiscuousMode(nicID, true),
		option.WithSpoofing(nicID, true),
		option.WithRouteTable(nicID),
		option.WithTransportHandler(t.handle),
	}
	for _, opt := range opts {
		if err := opt(t.stack); err != nil {
			return err
		}
	}
	return nil
}

func (t *Tun) Close() error {
	if t.stack != nil {
		t.stack.Close()
	}
	return nil
}

func (t *Tun) handle(src, dst net.Destination, conn net.Conn) {
	defer conn.Close()
	ctx, cancel := context.WithCancel(t.ctx)
	plcy := t.policyManager.ForLevel(0)
	timer := signal.CancelAfterInactivity(ctx, cancel, plcy.Timeouts.ConnectionIdle)
	ctx = log.ContextWithAccessMessage(ctx, &log.AccessMessage{
		From:   src,
		To:     dst,
		Status: log.AccessAccepted,
		Reason: "",
	})
	link, err := t.dispatcher.Dispatch(ctx, dst)
	if err != nil {
		errors.LogErrorInner(t.ctx, err, "dispatch connection")
	}
	defer cancel()
	reqDone := func() error {
		defer timer.SetTimeout(plcy.Timeouts.DownlinkOnly)
		if err := buf.Copy(buf.NewReader(conn), link.Writer, buf.UpdateActivity(timer)); err != nil {
			return errors.New("failed to transport all request").Base(err)
		}
		return nil
	}
	rspDone := func() error {
		defer timer.SetTimeout(plcy.Timeouts.UplinkOnly)
		if err := buf.Copy(link.Reader, buf.NewWriter(conn), buf.UpdateActivity(timer)); err != nil {
			return errors.New("failed to transport all response").Base(err)
		}
		return nil
	}
	donePost := task.OnSuccess(reqDone, task.Close(link.Writer))
	if err := task.Run(ctx, donePost, rspDone); err != nil {
		common.Interrupt(link.Reader)
		common.Interrupt(link.Writer)
		errors.LogDebugInner(t.ctx, err, "connection ends")
		return
	}
}
