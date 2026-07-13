package mbim

import (
	"context"
	"fmt"
	"slices"
)

func (c *Client) RadioState(ctx context.Context) (RadioStateInfo, error) {
	request := RadioStateRequest{TransactionID: c.nextTransactionID()}
	if err := c.transmit(ctx, request.Request()); err != nil {
		return RadioStateInfo{}, fmt.Errorf("reading MBIM radio state: %w", err)
	}
	return *request.Response, nil
}

func (c *Client) SetRadioState(ctx context.Context, state RadioSwitchState) (RadioStateInfo, error) {
	request := RadioStateSetRequest{
		TransactionID: c.nextTransactionID(),
		State:         state,
	}
	if err := c.transmit(ctx, request.Request()); err != nil {
		return RadioStateInfo{}, fmt.Errorf("setting MBIM radio state: %w", err)
	}
	return *request.Response, nil
}

func (c *Client) SubscriberReadyStatus(ctx context.Context) (SubscriberReadyStatusResponse, error) {
	request := SubscriberReadyStatusRequest{
		TransactionID: c.nextTransactionID(),
		MBIMExVersion: c.mbimExVersion,
		SlotID:        c.subscriberReadySlotID(),
	}
	if err := c.transmit(ctx, request.Request()); err != nil {
		return SubscriberReadyStatusResponse{}, fmt.Errorf("reading MBIM subscriber ready status: %w", err)
	}
	resp := *request.Response
	resp.TelephoneNumbers = slices.Clone(resp.TelephoneNumbers)
	return resp, nil
}
