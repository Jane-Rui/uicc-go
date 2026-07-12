package uim

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/damonto/uicc-go/qcom"
	"github.com/damonto/uicc-go/qcom/tlv"
)

const (
	dmsTLVOperatingMode       = 0x01
	dmsTLVReportOperatingMode = 0x14
	dmsTLVVoiceNumber         = 0x01
	dmsTLVMobileIDNumber      = 0x10
	dmsTLVIMSI                = 0x11
)

// DMSGetMSISDNRequest encodes QMI DMS Get MSISDN.
type DMSGetMSISDNRequest struct {
	ClientID      uint8
	TransactionID uint16
	Timeout       time.Duration
}

// Request converts the request into a QMI DMS request.
func (r DMSGetMSISDNRequest) Request() qcom.Request {
	return qcom.Request{
		Service:       qcom.ServiceDMS,
		ClientID:      r.ClientID,
		TransactionID: r.TransactionID,
		MessageID:     qcom.MessageDMSGetMSISDN,
		Timeout:       r.Timeout,
	}
}

// DMSGetOperatingModeRequest encodes QMI DMS Get Operating Mode.
type DMSGetOperatingModeRequest struct {
	ClientID      uint8
	TransactionID uint16
	Timeout       time.Duration
}

// Request converts the request into a QMI DMS request.
func (r DMSGetOperatingModeRequest) Request() qcom.Request {
	return qcom.Request{
		Service:       qcom.ServiceDMS,
		ClientID:      r.ClientID,
		TransactionID: r.TransactionID,
		MessageID:     qcom.MessageDMSGetOperatingMode,
		Timeout:       r.Timeout,
	}
}

// DMSSetOperatingModeRequest encodes QMI DMS Set Operating Mode.
type DMSSetOperatingModeRequest struct {
	ClientID      uint8
	TransactionID uint16
	Timeout       time.Duration
	Mode          qcom.DMSOperatingMode
}

// Request converts the request into a QMI DMS request.
func (r DMSSetOperatingModeRequest) Request() qcom.Request {
	return qcom.Request{
		Service:       qcom.ServiceDMS,
		ClientID:      r.ClientID,
		TransactionID: r.TransactionID,
		MessageID:     qcom.MessageDMSSetOperatingMode,
		Timeout:       r.Timeout,
		TLVs: tlv.TLVs{
			tlv.Uint(dmsTLVOperatingMode, uint8(r.Mode)),
		},
	}
}

// DMSSetEventReportRequest encodes QMI DMS Set Event Report for operating mode.
type DMSSetEventReportRequest struct {
	ClientID            uint8
	TransactionID       uint16
	Timeout             time.Duration
	ReportOperatingMode bool
}

// Request converts the request into a QMI DMS request.
func (r DMSSetEventReportRequest) Request() qcom.Request {
	report := uint8(0)
	if r.ReportOperatingMode {
		report = 1
	}

	return qcom.Request{
		Service:       qcom.ServiceDMS,
		ClientID:      r.ClientID,
		TransactionID: r.TransactionID,
		MessageID:     qcom.MessageDMSSetEventReport,
		Timeout:       r.Timeout,
		TLVs: tlv.TLVs{
			tlv.Uint(dmsTLVReportOperatingMode, report),
		},
	}
}

// OperatingMode reads the current QMI DMS modem operating mode.
func (r *Reader) OperatingMode(ctx context.Context) (qcom.DMSOperatingMode, error) {
	var mode qcom.DMSOperatingMode
	err := r.withServiceClient(ctx, qcom.ServiceDMS, func(clientID uint8) error {
		req := DMSGetOperatingModeRequest{
			ClientID: clientID,
			Timeout:  DefaultRequestTimeout,
		}.Request()
		resp, err := r.requestServiceWithTimeout(ctx, req.Service, req.ClientID, req.MessageID, req.TLVs, req.Timeout)
		if err != nil {
			return err
		}
		if err := resultOK(resp); err != nil {
			return err
		}

		var parsed DMSGetOperatingModeResponse
		if err := parsed.UnmarshalTLVs(resp.TLVs); err != nil {
			return err
		}
		mode = parsed.Mode
		return nil
	})
	if err != nil {
		return 0, fmt.Errorf("querying QMI DMS operating mode: %w", err)
	}
	return mode, nil
}

// SetOperatingMode sets the QMI DMS modem operating mode.
func (r *Reader) SetOperatingMode(ctx context.Context, mode qcom.DMSOperatingMode) error {
	err := r.withServiceClient(ctx, qcom.ServiceDMS, func(clientID uint8) error {
		req := DMSSetOperatingModeRequest{
			ClientID: clientID,
			Timeout:  DefaultRequestTimeout,
			Mode:     mode,
		}.Request()
		resp, err := r.requestServiceWithTimeout(ctx, req.Service, req.ClientID, req.MessageID, req.TLVs, req.Timeout)
		if err != nil {
			return err
		}
		return resultOK(resp)
	})
	if err != nil {
		return fmt.Errorf("setting QMI DMS operating mode: %w", err)
	}
	return nil
}

// MSISDN returns the voice number and related subscriber identifiers reported by QMI DMS.
func (r *Reader) MSISDN(ctx context.Context) (DMSGetMSISDNResponse, error) {
	var result DMSGetMSISDNResponse
	err := r.withServiceClient(ctx, qcom.ServiceDMS, func(clientID uint8) error {
		req := DMSGetMSISDNRequest{
			ClientID: clientID,
			Timeout:  DefaultRequestTimeout,
		}.Request()
		resp, err := r.requestServiceWithTimeout(ctx, req.Service, req.ClientID, req.MessageID, req.TLVs, req.Timeout)
		if err != nil {
			return err
		}
		if err := resultOK(resp); err != nil {
			return err
		}
		return result.UnmarshalTLVs(resp.TLVs)
	})
	if err != nil {
		return DMSGetMSISDNResponse{}, fmt.Errorf("querying QMI DMS MSISDN: %w", err)
	}
	return result, nil
}

// DMSGetMSISDNResponse is the parsed QMI DMS Get MSISDN response.
type DMSGetMSISDNResponse struct {
	VoiceNumber    string
	MobileIDNumber string
	IMSI           string
}

// UnmarshalTLVs parses QMI DMS Get MSISDN response TLVs.
func (r *DMSGetMSISDNResponse) UnmarshalTLVs(tlvs tlv.TLVs) error {
	*r = DMSGetMSISDNResponse{}
	voiceNumber, ok := tlv.Value(tlvs, dmsTLVVoiceNumber)
	if !ok {
		return errors.New("parsing QMI DMS MSISDN: voice number TLV missing")
	}
	r.VoiceNumber = string(voiceNumber)
	if mobileIDNumber, ok := tlv.Value(tlvs, dmsTLVMobileIDNumber); ok {
		r.MobileIDNumber = string(mobileIDNumber)
	}
	if imsi, ok := tlv.Value(tlvs, dmsTLVIMSI); ok {
		r.IMSI = string(imsi)
	}
	return nil
}

// DMSGetOperatingModeResponse is the parsed QMI DMS Get Operating Mode response.
type DMSGetOperatingModeResponse struct {
	Mode qcom.DMSOperatingMode
}

// UnmarshalTLVs parses QMI DMS Get Operating Mode response TLVs.
func (r *DMSGetOperatingModeResponse) UnmarshalTLVs(tlvs tlv.TLVs) error {
	*r = DMSGetOperatingModeResponse{}
	value, ok := tlv.Value(tlvs, dmsTLVOperatingMode)
	if !ok {
		return errors.New("parsing QMI DMS operating mode: operating mode TLV missing")
	}
	if len(value) < 1 {
		return errors.New("parsing QMI DMS operating mode: operating mode TLV is truncated")
	}
	r.Mode = qcom.DMSOperatingMode(value[0])
	return nil
}
