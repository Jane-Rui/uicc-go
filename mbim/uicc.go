package mbim

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"slices"

	"github.com/damonto/wwan-go/apdu"
)

func (c *Client) ListApplications(ctx context.Context) ([]Application, error) {
	request := ApplicationListRequest{TransactionID: c.nextTransactionID()}
	if err := c.transmit(ctx, request.Request()); err != nil {
		return nil, fmt.Errorf("listing MBIM applications: %w", err)
	}

	apps := make([]Application, 0, len(request.Response.Applications))
	for _, app := range request.Response.Applications {
		if len(app.AID) == 0 {
			continue
		}
		apps = append(apps, Application{
			AID:   slices.Clone(app.AID),
			Label: app.Label,
		})
	}
	return apps, nil
}

func (c *Client) AuthenticateAKA(ctx context.Context, rand, autn []byte) (*AuthAKAResponse, error) {
	if len(rand) != 16 {
		return nil, fmt.Errorf("authenticating MBIM AKA: RAND length %d, want 16", len(rand))
	}
	if len(autn) != 16 {
		return nil, fmt.Errorf("authenticating MBIM AKA: AUTN length %d, want 16", len(autn))
	}

	request := AuthAKARequest{
		TransactionID: c.nextTransactionID(),
		Rand:          slices.Clone(rand),
		AUTN:          slices.Clone(autn),
	}
	if err := c.transmit(ctx, request.Request()); err != nil {
		if errors.Is(err, StatusAuthSyncFailure) {
			return request.Response, fmt.Errorf("authenticating MBIM AKA: %w", err)
		}
		return nil, fmt.Errorf("authenticating MBIM AKA: %w", err)
	}
	return request.Response, nil
}

func (c *Client) QueryUiccATR(ctx context.Context) ([]byte, error) {
	request := UiccATRQueryRequest{
		TransactionID: c.nextTransactionID(),
		MBIMExVersion: c.mbimExVersion,
		SlotID:        c.slot,
	}
	if err := c.transmit(ctx, request.Request()); err != nil {
		return nil, fmt.Errorf("querying MBIM UICC ATR: %w", err)
	}
	return slices.Clone(request.Response.ATR), nil
}

func (c *Client) OpenChannel(ctx context.Context, aid []byte) (uint32, error) {
	request := OpenChannelRequest{
		TransactionID: c.nextTransactionID(),
		MBIMExVersion: c.mbimExVersion,
		SlotID:        c.slot,
		ApplicationID: slices.Clone(aid),
		ChannelGroup:  uiccChannelGroupDefault,
	}
	if err := c.transmit(ctx, request.Request()); err != nil {
		return 0, fmt.Errorf("opening MBIM UICC channel: %w", err)
	}
	if err := uiccStatusError(request.Response.Status); err != nil {
		return 0, fmt.Errorf("opening MBIM UICC channel: %w", err)
	}
	return request.Response.Channel, nil
}

func (c *Client) TransmitAPDU(ctx context.Context, channel uint32, command []byte) ([]byte, uint32, error) {
	request := APDURequest{
		TransactionID:   c.nextTransactionID(),
		MBIMExVersion:   c.mbimExVersion,
		SlotID:          c.slot,
		Channel:         channel,
		SecureMessaging: UiccSecureMessagingNone,
		ClassByteType:   UiccClassByteTypeInterIndustry,
		Command:         slices.Clone(command),
	}
	if err := c.transmit(ctx, request.Request()); err != nil {
		return nil, 0, fmt.Errorf("transmitting MBIM UICC APDU: %w", err)
	}
	return slices.Clone(request.Response.Response), request.Response.Status, nil
}

func (c *Client) SetUiccReset(ctx context.Context, action UiccPassThroughAction) (UiccPassThroughStatus, error) {
	request := UiccResetSetRequest{
		TransactionID: c.nextTransactionID(),
		MBIMExVersion: c.mbimExVersion,
		SlotID:        c.slot,
		Action:        action,
	}
	if err := c.transmit(ctx, request.Request()); err != nil {
		return 0, fmt.Errorf("setting MBIM UICC reset: %w", err)
	}
	c.clearEnvelopeSupport()
	return request.Response.PassThroughStatus, nil
}

func (c *Client) QueryUiccReset(ctx context.Context) (UiccPassThroughStatus, error) {
	request := UiccResetQueryRequest{
		TransactionID: c.nextTransactionID(),
		MBIMExVersion: c.mbimExVersion,
		SlotID:        c.slot,
	}
	if err := c.transmit(ctx, request.Request()); err != nil {
		return 0, fmt.Errorf("querying MBIM UICC reset: %w", err)
	}
	return request.Response.PassThroughStatus, nil
}

func (c *Client) SetUiccTerminalCapability(ctx context.Context, capabilities [][]byte) error {
	request := UiccTerminalCapabilitySetRequest{
		TransactionID: c.nextTransactionID(),
		MBIMExVersion: c.mbimExVersion,
		SlotID:        c.slot,
		Capabilities:  cloneByteSlices(capabilities),
	}
	if err := c.transmit(ctx, request.Request()); err != nil {
		return fmt.Errorf("setting MBIM UICC terminal capability: %w", err)
	}
	return nil
}

func (c *Client) QueryUiccTerminalCapability(ctx context.Context) ([][]byte, error) {
	request := UiccTerminalCapabilityQueryRequest{
		TransactionID: c.nextTransactionID(),
		MBIMExVersion: c.mbimExVersion,
		SlotID:        c.slot,
	}
	if err := c.transmit(ctx, request.Request()); err != nil {
		return nil, fmt.Errorf("querying MBIM UICC terminal capability: %w", err)
	}
	return cloneByteSlices(request.Response.Capabilities), nil
}

func (c *Client) CloseChannel(ctx context.Context, channel uint32) error {
	request := CloseChannelRequest{
		TransactionID: c.nextTransactionID(),
		MBIMExVersion: c.mbimExVersion,
		SlotID:        c.slot,
		Channel:       channel,
		ChannelGroup:  uiccChannelGroupDefault,
	}
	if err := c.transmit(ctx, request.Request()); err != nil {
		return fmt.Errorf("closing MBIM UICC channel: %w", err)
	}
	if err := uiccStatusError(request.Response.Status); err != nil {
		return fmt.Errorf("closing MBIM UICC channel: %w", err)
	}
	return nil
}

func (c *Client) QuerySTKPAC(ctx context.Context) (STKPACInfo, error) {
	request := STKPACQueryRequest{TransactionID: c.nextTransactionID()}
	if err := c.transmit(ctx, request.Request()); err != nil {
		return STKPACInfo{}, fmt.Errorf("querying MBIM STK PAC: %w", err)
	}
	return *request.Response, nil
}

func (c *Client) SetSTKPAC(ctx context.Context, pacHostControl []byte) (STKPACInfo, error) {
	if len(pacHostControl) != stkPACHostControlLength {
		return STKPACInfo{}, fmt.Errorf("setting MBIM STK PAC: host control length %d, want %d", len(pacHostControl), stkPACHostControlLength)
	}

	request := STKPACSetRequest{
		TransactionID:  c.nextTransactionID(),
		PacHostControl: slices.Clone(pacHostControl),
	}
	if err := c.transmit(ctx, request.Request()); err != nil {
		return STKPACInfo{}, fmt.Errorf("setting MBIM STK PAC: %w", err)
	}
	return *request.Response, nil
}

func (c *Client) ReadSTKPAC(ctx context.Context) (STKPAC, error) {
	indication, err := c.nextIndication(ctx, indicationKey{serviceID: ServiceSTK, commandID: CIDSTKPAC})
	if err != nil {
		return STKPAC{}, fmt.Errorf("reading MBIM STK PAC: %w", err)
	}
	var pac STKPAC
	if err := pac.UnmarshalBinary(indication.InformationBuffer); err != nil {
		return STKPAC{}, fmt.Errorf("reading MBIM STK PAC: %w", err)
	}
	return pac, nil
}

// WatchSTKPAC streams STK proactive command notifications until ctx is done.
func (c *Client) WatchSTKPAC(ctx context.Context) (<-chan STKPAC, error) {
	indications, unsubscribe, err := c.subscribeIndication(indicationKey{serviceID: ServiceSTK, commandID: CIDSTKPAC})
	if err != nil {
		return nil, fmt.Errorf("watching MBIM STK PAC: %w", err)
	}

	out := make(chan STKPAC, maxQueuedIndications)
	go func() {
		defer close(out)
		defer unsubscribe()

		for {
			select {
			case <-ctx.Done():
				return
			case indication, ok := <-indications:
				if !ok {
					return
				}
				var pac STKPAC
				if err := pac.UnmarshalBinary(indication.InformationBuffer); err != nil {
					continue
				}
				select {
				case out <- pac:
				case <-ctx.Done():
					return
				}
			}
		}
	}()
	return out, nil
}

func (c *Client) STKTerminalResponse(ctx context.Context, data []byte) (STKTerminalResponseInfo, error) {
	if len(data) == 0 {
		return STKTerminalResponseInfo{}, errors.New("sending MBIM STK terminal response: response is empty")
	}

	request := STKTerminalResponseRequest{
		TransactionID: c.nextTransactionID(),
		Data:          slices.Clone(data),
	}
	if err := c.transmit(ctx, request.Request()); err != nil {
		return STKTerminalResponseInfo{}, fmt.Errorf("sending MBIM STK terminal response: %w", err)
	}
	return *request.Response, nil
}

func (c *Client) QuerySTKEnvelopeSupport(ctx context.Context) (STKEnvelopeInfo, error) {
	info, err := c.querySTKEnvelopeSupport(ctx)
	if err != nil {
		return STKEnvelopeInfo{}, err
	}
	c.setEnvelopeSupport(info)
	return info, nil
}

func (c *Client) STKEnvelope(ctx context.Context, data []byte) error {
	if len(data) == 0 {
		return errors.New("running MBIM STK envelope: envelope is empty")
	}

	info, err := c.envelopeSupportInfo(ctx)
	if err != nil {
		return fmt.Errorf("running MBIM STK envelope: %w", err)
	}
	if !info.Supports(data[0]) {
		return fmt.Errorf("running MBIM STK envelope: envelope tag 0x%02X is not expected by function", data[0])
	}

	request := STKEnvelopeRequest{
		TransactionID: c.nextTransactionID(),
		Data:          slices.Clone(data),
	}
	if err := c.transmit(ctx, request.Request()); err != nil {
		return fmt.Errorf("running MBIM STK envelope: %w", err)
	}
	return nil
}

func (c *Client) querySTKEnvelopeSupport(ctx context.Context) (STKEnvelopeInfo, error) {
	request := STKEnvelopeQueryRequest{TransactionID: c.nextTransactionID()}
	if err := c.transmit(ctx, request.Request()); err != nil {
		return STKEnvelopeInfo{}, fmt.Errorf("querying MBIM STK envelope support: %w", err)
	}
	return *request.Response, nil
}

func (c *Client) envelopeSupportInfo(ctx context.Context) (STKEnvelopeInfo, error) {
	c.mu.Lock()
	if c.envelopeSupport != nil {
		info := *c.envelopeSupport
		c.mu.Unlock()
		return info, nil
	}
	c.mu.Unlock()

	info, err := c.querySTKEnvelopeSupport(ctx)
	if err != nil {
		return STKEnvelopeInfo{}, err
	}
	c.setEnvelopeSupport(info)
	return info, nil
}

func (c *Client) setEnvelopeSupport(info STKEnvelopeInfo) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.envelopeSupport = new(STKEnvelopeInfo)
	*c.envelopeSupport = info
}

func (c *Client) clearEnvelopeSupport() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.envelopeSupport = nil
}

func cloneByteSlices(values [][]byte) [][]byte {
	if values == nil {
		return nil
	}
	clones := make([][]byte, len(values))
	for i, value := range values {
		clones[i] = slices.Clone(value)
	}
	return clones
}

func uiccStatusError(status uint32) error {
	if uiccStatusOK(status) {
		return nil
	}
	return apdu.StatusError{SW: uiccStatusCode(status)}
}

func uiccStatusOK(status uint32) bool {
	return status == 0 || uiccStatusCode(status) == 0x9000
}

func uiccStatusCode(status uint32) uint16 {
	var sw [2]byte
	binary.LittleEndian.PutUint16(sw[:], uint16(status&0xffff))
	return binary.BigEndian.Uint16(sw[:])
}

func cardStatusError(sw1, sw2 uint32) error {
	if sw1 == 0x90 && sw2 == 0x00 {
		return nil
	}
	return fmt.Errorf("unexpected status word 0x%02X%02X", sw1, sw2)
}
