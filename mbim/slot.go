package mbim

import (
	"context"
	"errors"
	"fmt"
	"time"
)

const (
	slotPollInterval = 500 * time.Millisecond
	slotReadyTimeout = 5 * time.Second
)

func (c *Client) ensureSlotActivated(ctx context.Context) error {
	slot, err := c.currentActivatedSlot(ctx)
	if err != nil {
		if errors.Is(err, StatusNoDeviceSupport) {
			return nil
		}
		return fmt.Errorf("activating MBIM slot %d: %w", c.slot+1, err)
	}
	if slot == c.slot {
		return nil
	}
	if err := c.activateSlot(ctx, c.slot); err != nil {
		return fmt.Errorf("activating MBIM slot %d: %w", c.slot+1, err)
	}
	if err := c.waitForSlotReady(ctx); err != nil {
		return fmt.Errorf("activating MBIM slot %d: %w", c.slot+1, err)
	}
	return nil
}

func (c *Client) currentActivatedSlot(ctx context.Context) (uint32, error) {
	request := DeviceSlotMappingsRequest{TransactionID: c.nextTransactionID()}
	if err := c.transmit(ctx, request.Request()); err != nil {
		return 0, err
	}
	if len(request.Response.SlotMappings) == 0 {
		return 0, errors.New("reading MBIM slot mappings: mapping is empty")
	}
	return request.Response.SlotMappings[0].Slot, nil
}

func (c *Client) activateSlot(ctx context.Context, slot uint32) error {
	request := DeviceSlotMappingsRequest{
		TransactionID: c.nextTransactionID(),
		SlotMappings:  []SlotMapping{{Slot: slot}},
	}
	return c.transmit(ctx, request.Request())
}

func (c *Client) waitForSlotReady(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, slotReadyTimeout)
	defer cancel()

	var lastReadyState SubscriberReadyState
	var sawReadyState bool
	for {
		request := SubscriberReadyStatusRequest{
			TransactionID: c.nextTransactionID(),
			MBIMExVersion: c.mbimExVersion,
			SlotID:        c.subscriberReadySlotID(),
		}
		err := c.transmit(ctx, request.Request())
		if err == nil {
			sawReadyState = true
			lastReadyState = request.Response.ReadyState
			if lastReadyState == SubscriberReadyStateInitialized || lastReadyState == SubscriberReadyStateNoESIMProfile {
				return nil
			}
		}
		if ctx.Err() != nil {
			if err != nil {
				return fmt.Errorf("waiting for MBIM SIM readiness: %w", err)
			}
			if sawReadyState {
				return fmt.Errorf("waiting for MBIM SIM readiness: last ready state %#x", lastReadyState)
			}
			return errors.New("waiting for MBIM SIM readiness: timeout")
		}

		timer := time.NewTimer(slotPollInterval)
		select {
		case <-ctx.Done():
			timer.Stop()
		case <-timer.C:
		}
	}
}
