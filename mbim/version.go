package mbim

import (
	"context"
	"fmt"
)

func (c *Client) negotiateVersion(ctx context.Context) error {
	c.mbimExVersion = mbimExVersion10

	services := DeviceServicesRequest{TransactionID: c.nextTransactionID()}
	if err := c.transmit(ctx, services.Request()); err != nil {
		return fmt.Errorf("negotiating MBIM version: reading device services: %w", err)
	}
	if !services.Response.SupportsCID(ServiceMsBasicConnectExtensions, CIDVersion) {
		return nil
	}

	version := VersionRequest{
		TransactionID: c.nextTransactionID(),
		MBIMVersion:   mbimVersion10,
		MBIMExVersion: hostMBIMExVersion,
	}
	if err := c.transmit(ctx, version.Request()); err != nil {
		return fmt.Errorf("negotiating MBIM version: %w", err)
	}
	c.mbimExVersion = min(version.Response.MBIMExVersion, hostMBIMExVersion)
	return nil
}

func (c *Client) usesUiccSlotID() bool {
	return c.mbimExVersion >= mbimExVersion40
}

func (c *Client) subscriberReadySlotID() uint32 {
	if c.usesUiccSlotID() {
		return c.slot
	}
	return activeSubscriberSlot
}
