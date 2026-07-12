package uim

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/damonto/uicc-go/qcom"
	"github.com/damonto/uicc-go/qcom/tlv"
)

const nasTLVServingSystem = 0x01

// NASGetServingSystemRequest encodes QMI NAS Get Serving System.
type NASGetServingSystemRequest struct {
	ClientID      uint8
	TransactionID uint16
	Timeout       time.Duration
}

// Request converts the request into a QMI NAS request.
func (r NASGetServingSystemRequest) Request() qcom.Request {
	return qcom.Request{
		Service:       qcom.ServiceNAS,
		ClientID:      r.ClientID,
		TransactionID: r.TransactionID,
		MessageID:     qcom.MessageNASGetServingSystem,
		Timeout:       r.Timeout,
	}
}

// NASServingSystem reads the current QMI NAS serving-system fields.
func (r *Reader) NASServingSystem(ctx context.Context) (qcom.NASServingSystem, error) {
	var serving qcom.NASServingSystem
	err := r.withServiceClient(ctx, qcom.ServiceNAS, func(clientID uint8) error {
		req := NASGetServingSystemRequest{ClientID: clientID, Timeout: DefaultRequestTimeout}.Request()
		resp, err := r.requestServiceWithTimeout(ctx, req.Service, req.ClientID, req.MessageID, req.TLVs, req.Timeout)
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
		return qcom.NASServingSystem{}, fmt.Errorf("querying QMI NAS serving system: %w", err)
	}
	return serving, nil
}

// NASGetServingSystemResponse is the parsed NAS Get Serving System response.
type NASGetServingSystemResponse struct {
	ServingSystem qcom.NASServingSystem
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
	r.ServingSystem = qcom.NASServingSystem{
		RegistrationState: qcom.NASRegistrationState(value[0]),
		CSAttachState:     qcom.NASAttachState(value[1]),
		PSAttachState:     qcom.NASAttachState(value[2]),
		SelectedNetwork:   qcom.NASSelectedNetwork(value[3]),
		RadioInterfaces:   make([]qcom.NASRadioInterface, count),
	}
	for i, radio := range value[5 : 5+count] {
		r.ServingSystem.RadioInterfaces[i] = qcom.NASRadioInterface(radio)
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
func (r NASGetSysInfoRequest) Request() qcom.Request {
	return qcom.Request{
		Service:       qcom.ServiceNAS,
		ClientID:      r.ClientID,
		TransactionID: r.TransactionID,
		MessageID:     qcom.MessageNASGetSysInfo,
		Timeout:       r.Timeout,
	}
}

// NASGetSysInfoResponse is the parsed NAS Get System Info response.
type NASGetSysInfoResponse struct {
	SysInfo qcom.NASSysInfo
}

// UnmarshalTLVs parses the NAS Get System Info response fields used by IMS.
func (r *NASGetSysInfoResponse) UnmarshalTLVs(tlvs tlv.TLVs) error {
	*r = NASGetSysInfoResponse{}
	value, ok := tlv.Value(tlvs, 0x29)
	if !ok || len(value) == 0 {
		return nil
	}
	r.SysInfo = qcom.NASSysInfo{
		VoPSKnown:     true,
		VoPSSupported: value[0] == 1,
	}
	return nil
}
