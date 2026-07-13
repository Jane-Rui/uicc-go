package qcom

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/damonto/wwan-go/qcom/tlv"
)

const nasTLVServingSystem = 0x01

// NASGetServingSystemRequest encodes QMI NAS Get Serving System.
type NASGetServingSystemRequest struct {
	ClientID      uint8
	TransactionID uint16
	Timeout       time.Duration
}

// Request converts the request into a QMI NAS request.
func (r NASGetServingSystemRequest) Request() Request {
	return Request{
		Service:       ServiceNAS,
		ClientID:      r.ClientID,
		TransactionID: r.TransactionID,
		MessageID:     MessageNASGetServingSystem,
		Timeout:       r.Timeout,
	}
}

// NASServingSystem reads the current QMI NAS serving-system fields.
func (c *Client) NASServingSystem(ctx context.Context) (NASServingSystem, error) {
	var serving NASServingSystem
	err := c.withServiceClient(ctx, ServiceNAS, func(clientID uint8) error {
		req := NASGetServingSystemRequest{ClientID: clientID, Timeout: DefaultRequestTimeout}.Request()
		resp, err := c.requestServiceWithTimeout(ctx, req.Service, req.ClientID, req.MessageID, req.TLVs, req.Timeout)
		if err != nil {
			return err
		}
		if err := resultOK(resp); err != nil {
			return err
		}
		var parsed NASGetServingSystemResponse
		if err := parsed.UnmarshalTLVs(resp.TLVs); err != nil {
			return err
		}
		serving = parsed.ServingSystem
		return nil
	})
	if err != nil {
		return NASServingSystem{}, fmt.Errorf("querying QMI NAS serving system: %w", err)
	}
	return serving, nil
}

// NASGetServingSystemResponse is the parsed NAS Get Serving System response.
type NASGetServingSystemResponse struct {
	ServingSystem NASServingSystem
}

// UnmarshalTLVs parses the mandatory serving-system aggregate.
func (r *NASGetServingSystemResponse) UnmarshalTLVs(tlvs tlv.TLVs) error {
	*r = NASGetServingSystemResponse{}
	value, ok := tlv.Value(tlvs, nasTLVServingSystem)
	if !ok {
		return errors.New("parsing QMI NAS serving system: serving system TLV missing")
	}
	if len(value) < 5 {
		return errors.New("parsing QMI NAS serving system: serving system TLV is truncated")
	}
	count := int(value[4])
	if len(value) < 5+count {
		return errors.New("parsing QMI NAS serving system: radio interface list is truncated")
	}
	r.ServingSystem = NASServingSystem{
		RegistrationState: NASRegistrationState(value[0]),
		CSAttachState:     NASAttachState(value[1]),
		PSAttachState:     NASAttachState(value[2]),
		SelectedNetwork:   NASSelectedNetwork(value[3]),
		RadioInterfaces:   make([]NASRadioInterface, count),
	}
	for i, radio := range value[5 : 5+count] {
		r.ServingSystem.RadioInterfaces[i] = NASRadioInterface(radio)
	}
	return nil
}

// NASGetSysInfoRequest encodes QMI NAS Get System Info.
type NASGetSysInfoRequest struct {
	ClientID      uint8
	TransactionID uint16
	Timeout       time.Duration
}

// Request converts the request into a QMI NAS request.
func (r NASGetSysInfoRequest) Request() Request {
	return Request{
		Service:       ServiceNAS,
		ClientID:      r.ClientID,
		TransactionID: r.TransactionID,
		MessageID:     MessageNASGetSysInfo,
		Timeout:       r.Timeout,
	}
}

// NASGetSysInfoResponse is the parsed NAS Get System Info response.
type NASGetSysInfoResponse struct {
	SysInfo NASSysInfo
}

// UnmarshalTLVs parses the NAS Get System Info response fields used by IMS.
func (r *NASGetSysInfoResponse) UnmarshalTLVs(tlvs tlv.TLVs) error {
	*r = NASGetSysInfoResponse{}
	value, ok := tlv.Value(tlvs, 0x29)
	if !ok || len(value) == 0 {
		return nil
	}
	r.SysInfo = NASSysInfo{
		VoPSKnown:     true,
		VoPSSupported: value[0] == 1,
	}
	return nil
}
