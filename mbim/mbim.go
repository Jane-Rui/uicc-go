package mbim

import (
	"context"
	"errors"
	"fmt"
	"io"
	"sync"
	"sync/atomic"
	"time"
)

const defaultCloseTimeout = 5 * time.Second

type Client struct {
	conn               Conn
	slot               uint32
	mbimExVersion      uint16
	txn                atomic.Uint32
	proxy              bool
	maxControlTransfer int

	mu              sync.Mutex
	writeMu         sync.Mutex
	closed          bool
	closing         bool
	receiverStarted bool
	receiverErr     error
	pending         map[uint32]*responseWaiter
	subs            map[indicationKey]map[chan Indication]struct{}
	waiters         map[indicationKey][]chan Indication
	indications     map[indicationKey][]Indication
	envelopeSupport *STKEnvelopeInfo
}

type Option func(*config)

type config struct {
	dialer Dialer
	slot   int
}

func WithDialer(d Dialer) Option {
	return func(c *config) {
		c.dialer = d
	}
}

func WithProxy(device string) Option {
	return func(c *config) {
		c.dialer = ProxyDialer{Device: device}
	}
}

func WithDirect(device string) Option {
	return func(c *config) {
		c.dialer = DirectDialer{Device: device}
	}
}

func WithSlot(slot int) Option {
	return func(c *config) {
		c.slot = slot
	}
}

func Open(ctx context.Context, opts ...Option) (*Client, error) {
	cfg := config{slot: 1}
	for _, opt := range opts {
		opt(&cfg)
	}
	if cfg.slot < 1 {
		return nil, fmt.Errorf("opening MBIM client: slot %d is out of range", cfg.slot)
	}
	if cfg.dialer == nil {
		return nil, errors.New("opening MBIM client: dialer is nil")
	}

	device := dialerDevice(cfg.dialer)
	if dialerUsesProxy(cfg.dialer) && device == "" {
		return nil, errors.New("opening MBIM proxy: device is empty")
	}

	conn, err := cfg.dialer.Dial(ctx)
	if err != nil {
		return nil, err
	}

	client := &Client{
		conn:               conn,
		slot:               uint32(cfg.slot - 1),
		proxy:              dialerUsesProxy(cfg.dialer),
		maxControlTransfer: connMaxControlTransfer(conn),
	}
	if err := client.connect(ctx, device); err != nil {
		_ = conn.Close()
		return nil, err
	}
	return client, nil
}

func dialerDevice(d Dialer) string {
	device, ok := d.(deviceDialer)
	if ok {
		return device.device()
	}
	return ""
}

func (c *Client) connect(ctx context.Context, device string) error {
	if c.proxy {
		if err := c.configureProxy(ctx, device); err != nil {
			return err
		}
	}
	if err := c.openDevice(ctx); err != nil {
		return err
	}
	if err := c.startReceiver(); err != nil {
		return err
	}
	if err := c.negotiateVersion(ctx); err != nil {
		return err
	}
	if !c.usesUiccSlotID() {
		if err := c.ensureSlotActivated(ctx); err != nil {
			return err
		}
	}
	return nil
}

func (c *Client) configureProxy(ctx context.Context, device string) error {
	request := ProxyConfigRequest{
		TransactionID: c.nextTransactionID(),
		DevicePath:    device,
		Timeout:       30,
	}
	if err := request.Request().Transmit(ctx, c.conn); err != nil {
		if errors.Is(err, io.EOF) {
			return fmt.Errorf("opening MBIM client: device %s is not connected", device)
		}
		return fmt.Errorf("configuring MBIM proxy for %s: %w", device, err)
	}
	return nil
}

func (c *Client) openDevice(ctx context.Context) error {
	request := OpenDeviceRequest{
		TransactionID:      c.nextTransactionID(),
		MaxControlTransfer: uint32(c.maxControlTransfer),
	}
	if err := request.Request().Transmit(ctx, c.conn); err != nil {
		return fmt.Errorf("opening MBIM device: %w", err)
	}
	return nil
}

func (c *Client) Close() error {
	ctx, cancel := context.WithTimeout(context.Background(), defaultCloseTimeout)
	defer cancel()

	if !c.beginClose() {
		return nil
	}

	request := CloseRequest{TransactionID: c.nextTransactionID()}
	err := c.transmitClosing(ctx, request.Request())
	closeErr := c.conn.Close()
	c.finishClose()
	return errors.Join(err, closeErr)
}

func (c *Client) nextTransactionID() uint32 {
	return c.txn.Add(1)
}
