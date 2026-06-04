package qrtr

import (
	"context"
	"errors"

	"github.com/damonto/uicc-go/qualcomm"
)

type Dialer interface {
	Dial(ctx context.Context, service qualcomm.ServiceType) (packetConn, error)
}

type DirectDialer struct{}

type Option func(*config)

type config struct {
	service qualcomm.ServiceType
	dialer  Dialer
}

func WithDialer(d Dialer) Option {
	return func(c *config) {
		c.dialer = d
	}
}

func WithService(service qualcomm.ServiceType) Option {
	return func(c *config) {
		c.service = service
	}
}

func Open(ctx context.Context, opts ...Option) (*Transport, error) {
	cfg := config{
		service: qualcomm.ServiceUIM,
		dialer:  DirectDialer{},
	}
	for _, opt := range opts {
		opt(&cfg)
	}
	if cfg.dialer == nil {
		return nil, errors.New("opening QRTR transport: dialer is nil")
	}

	conn, err := cfg.dialer.Dial(ctx, cfg.service)
	if err != nil {
		return nil, err
	}
	return New(conn), nil
}

func (d DirectDialer) Dial(ctx context.Context, service qualcomm.ServiceType) (packetConn, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return openService(service)
}

var _ qualcomm.Transport = (*Transport)(nil)
