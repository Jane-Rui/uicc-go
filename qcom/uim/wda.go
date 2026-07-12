package uim

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"time"

	"github.com/damonto/uicc-go/qcom"
	"github.com/damonto/uicc-go/qcom/tlv"
)

const (
	wdaTLVQoSFormat               = 0x10
	wdaTLVLinkProtocol            = 0x11
	wdaTLVUplinkAggregation       = 0x12
	wdaTLVDownlinkAggregation     = 0x13
	wdaTLVNDPSignature            = 0x14
	wdaTLVDownlinkMaxDatagrams    = 0x15
	wdaTLVDownlinkMaxSize         = 0x16
	wdaTLVSetEndpoint             = 0x17
	wdaTLVSetQoSHeaderFormat      = 0x18
	wdaTLVSetDownlinkPadding      = 0x19
	wdaTLVSetFlowControl          = 0x1A
	wdaTLVUplinkMaxDatagrams      = 0x17
	wdaTLVUplinkMaxSize           = 0x18
	wdaTLVResponseQoSHeader       = 0x19
	wdaTLVResponseDownlinkPadding = 0x1A
	wdaTLVResponseFlowControl     = 0x1B
	wdaTLVGetEndpoint             = 0x10
)

// WDASetDataFormatRequest encodes QMI WDA Set Data Format.
type WDASetDataFormatRequest struct {
	ClientID      uint8
	TransactionID uint16
	Timeout       time.Duration
	Config        qcom.WDADataFormatConfig
}

// Request converts the requested fields into WDA TLVs.
func (r WDASetDataFormatRequest) Request() qcom.Request {
	tlvs := make(tlv.TLVs, 0, 11)
	if r.Config.QoSEnabled != nil {
		tlvs = append(tlvs, tlv.Uint(wdaTLVQoSFormat, boolByte(*r.Config.QoSEnabled)))
	}
	if r.Config.LinkLayerProtocol != nil {
		tlvs = append(tlvs, tlv.Uint(wdaTLVLinkProtocol, uint32(*r.Config.LinkLayerProtocol)))
	}
	if r.Config.UplinkAggregation != nil {
		tlvs = append(tlvs, tlv.Uint(wdaTLVUplinkAggregation, uint32(*r.Config.UplinkAggregation)))
	}
	if r.Config.DownlinkAggregation != nil {
		tlvs = append(tlvs, tlv.Uint(wdaTLVDownlinkAggregation, uint32(*r.Config.DownlinkAggregation)))
	}
	if r.Config.NDPSignature != nil {
		tlvs = append(tlvs, tlv.Uint(wdaTLVNDPSignature, *r.Config.NDPSignature))
	}
	if r.Config.DownlinkMaxDatagrams != nil {
		tlvs = append(tlvs, tlv.Uint(wdaTLVDownlinkMaxDatagrams, *r.Config.DownlinkMaxDatagrams))
	}
	if r.Config.DownlinkMaxSize != nil {
		tlvs = append(tlvs, tlv.Uint(wdaTLVDownlinkMaxSize, *r.Config.DownlinkMaxSize))
	}
	if r.Config.Endpoint != nil {
		endpoint, _ := r.Config.Endpoint.MarshalBinary() // Fixed-width endpoint encoding cannot fail.
		tlvs = append(tlvs, tlv.Bytes(wdaTLVSetEndpoint, endpoint))
	}
	if r.Config.QoSHeaderFormat != nil {
		tlvs = append(tlvs, tlv.Uint(wdaTLVSetQoSHeaderFormat, uint32(*r.Config.QoSHeaderFormat)))
	}
	if r.Config.DownlinkMinimumPadding != nil {
		tlvs = append(tlvs, tlv.Uint(wdaTLVSetDownlinkPadding, *r.Config.DownlinkMinimumPadding))
	}
	if r.Config.TerminalEquipmentFlowControl != nil {
		tlvs = append(tlvs, tlv.Uint(wdaTLVSetFlowControl, boolByte(*r.Config.TerminalEquipmentFlowControl)))
	}
	return qcom.Request{
		Service:       qcom.ServiceWDA,
		ClientID:      r.ClientID,
		TransactionID: r.TransactionID,
		MessageID:     qcom.MessageWDASetDataFormat,
		Timeout:       r.Timeout,
		TLVs:          tlvs,
	}
}

// WDAGetDataFormatRequest encodes QMI WDA Get Data Format.
type WDAGetDataFormatRequest struct {
	ClientID      uint8
	TransactionID uint16
	Timeout       time.Duration
	Endpoint      *qcom.DataEndpoint
}

// Request queries the default data port unless Endpoint is provided.
func (r WDAGetDataFormatRequest) Request() qcom.Request {
	var tlvs tlv.TLVs
	if r.Endpoint != nil {
		endpoint, _ := r.Endpoint.MarshalBinary() // Fixed-width endpoint encoding cannot fail.
		tlvs = tlv.TLVs{tlv.Bytes(wdaTLVGetEndpoint, endpoint)}
	}
	return qcom.Request{
		Service:       qcom.ServiceWDA,
		ClientID:      r.ClientID,
		TransactionID: r.TransactionID,
		MessageID:     qcom.MessageWDAGetDataFormat,
		Timeout:       r.Timeout,
		TLVs:          tlvs,
	}
}

// WDADataFormatResponse is the parsed Set/Get Data Format response.
type WDADataFormatResponse struct {
	Format qcom.WDADataFormat
}

// UnmarshalTLVs parses the optional data-format fields returned by WDA.
func (r *WDADataFormatResponse) UnmarshalTLVs(tlvs tlv.TLVs) error {
	*r = WDADataFormatResponse{}
	format := &r.Format

	if value, ok := tlv.Value(tlvs, wdaTLVQoSFormat); ok {
		if len(value) < 1 {
			return errors.New("parsing QMI WDA data format: QoS format TLV is truncated")
		}
		format.QoSEnabled = value[0] == 1
		format.QoSEnabledKnown = true
	}
	if value, ok := tlv.Value(tlvs, wdaTLVLinkProtocol); ok {
		parsed, err := wdaUint32(value, "link protocol")
		if err != nil {
			return err
		}
		format.LinkLayerProtocol = qcom.WDALinkLayerProtocol(parsed)
		format.LinkLayerProtocolKnown = true
	}
	if value, ok := tlv.Value(tlvs, wdaTLVUplinkAggregation); ok {
		parsed, err := wdaUint32(value, "uplink aggregation")
		if err != nil {
			return err
		}
		format.UplinkAggregation = qcom.WDAAggregationProtocol(parsed)
		format.UplinkAggregationKnown = true
	}
	if value, ok := tlv.Value(tlvs, wdaTLVDownlinkAggregation); ok {
		parsed, err := wdaUint32(value, "downlink aggregation")
		if err != nil {
			return err
		}
		format.DownlinkAggregation = qcom.WDAAggregationProtocol(parsed)
		format.DownlinkAggregationKnown = true
	}
	if err := parseWDAUint32(tlvs, wdaTLVNDPSignature, "NDP signature", &format.NDPSignature, &format.NDPSignatureKnown); err != nil {
		return err
	}
	if err := parseWDAUint32(tlvs, wdaTLVDownlinkMaxDatagrams, "downlink max datagrams", &format.DownlinkMaxDatagrams, &format.DownlinkMaxDatagramsKnown); err != nil {
		return err
	}
	if err := parseWDAUint32(tlvs, wdaTLVDownlinkMaxSize, "downlink max size", &format.DownlinkMaxSize, &format.DownlinkMaxSizeKnown); err != nil {
		return err
	}
	if err := parseWDAUint32(tlvs, wdaTLVUplinkMaxDatagrams, "uplink max datagrams", &format.UplinkMaxDatagrams, &format.UplinkMaxDatagramsKnown); err != nil {
		return err
	}
	if err := parseWDAUint32(tlvs, wdaTLVUplinkMaxSize, "uplink max size", &format.UplinkMaxSize, &format.UplinkMaxSizeKnown); err != nil {
		return err
	}
	if value, ok := tlv.Value(tlvs, wdaTLVResponseQoSHeader); ok {
		parsed, err := wdaUint32(value, "QoS header format")
		if err != nil {
			return err
		}
		format.QoSHeaderFormat = qcom.WDAQoSHeaderFormat(parsed)
		format.QoSHeaderFormatKnown = true
	}
	if err := parseWDAUint32(tlvs, wdaTLVResponseDownlinkPadding, "downlink minimum padding", &format.DownlinkMinimumPadding, &format.DownlinkMinimumPaddingKnown); err != nil {
		return err
	}
	if value, ok := tlv.Value(tlvs, wdaTLVResponseFlowControl); ok {
		if len(value) < 1 {
			return errors.New("parsing QMI WDA data format: flow control TLV is truncated")
		}
		format.TerminalEquipmentFlowControl = value[0] == 1
		format.TerminalEquipmentFlowControlKnown = true
	}
	return nil
}

// WDADataFormat reads the current format of the default modem data port.
func (r *Reader) WDADataFormat(ctx context.Context) (qcom.WDADataFormat, error) {
	return r.WDADataFormatForEndpoint(ctx, nil)
}

// WDADataFormatForEndpoint reads the current format of a modem data endpoint.
// A nil endpoint queries the default data port.
func (r *Reader) WDADataFormatForEndpoint(ctx context.Context, endpoint *qcom.DataEndpoint) (qcom.WDADataFormat, error) {
	var format qcom.WDADataFormat
	err := r.withServiceClient(ctx, qcom.ServiceWDA, func(clientID uint8) error {
		req := WDAGetDataFormatRequest{ClientID: clientID, Timeout: DefaultRequestTimeout, Endpoint: endpoint}.Request()
		resp, err := r.requestServiceWithTimeout(ctx, req.Service, req.ClientID, req.MessageID, req.TLVs, req.Timeout)
		if err != nil {
			return err
		}
		if err := resultOK(resp); err != nil {
			return err
		}
		var parsed WDADataFormatResponse
		if err := parsed.UnmarshalTLVs(resp.TLVs); err != nil {
			return err
		}
		format = parsed.Format
		return nil
	})
	if err != nil {
		return qcom.WDADataFormat{}, fmt.Errorf("querying QMI WDA data format: %w", err)
	}
	return format, nil
}

// SetWDADataFormat applies data-format fields to the modem's default data port.
func (r *Reader) SetWDADataFormat(ctx context.Context, config qcom.WDADataFormatConfig) (qcom.WDADataFormat, error) {
	var format qcom.WDADataFormat
	err := r.withServiceClient(ctx, qcom.ServiceWDA, func(clientID uint8) error {
		req := WDASetDataFormatRequest{ClientID: clientID, Timeout: DefaultRequestTimeout, Config: config}.Request()
		resp, err := r.requestServiceWithTimeout(ctx, req.Service, req.ClientID, req.MessageID, req.TLVs, req.Timeout)
		if err != nil {
			return err
		}
		if err := resultOK(resp); err != nil {
			return err
		}
		var parsed WDADataFormatResponse
		if err := parsed.UnmarshalTLVs(resp.TLVs); err != nil {
			return err
		}
		format = parsed.Format
		return nil
	})
	if err != nil {
		return qcom.WDADataFormat{}, fmt.Errorf("setting QMI WDA data format: %w", err)
	}
	return format, nil
}

// SetWDALinkLayerProtocol changes only the link-layer framing field.
func (r *Reader) SetWDALinkLayerProtocol(ctx context.Context, protocol qcom.WDALinkLayerProtocol) error {
	_, err := r.SetWDADataFormat(ctx, qcom.WDADataFormatConfig{LinkLayerProtocol: &protocol})
	return err
}

func wdaUint32(value []byte, name string) (uint32, error) {
	if len(value) < 4 {
		return 0, fmt.Errorf("parsing QMI WDA data format: %s TLV is truncated", name)
	}
	return binary.LittleEndian.Uint32(value[:4]), nil
}

func parseWDAUint32(tlvs tlv.TLVs, kind byte, name string, dst *uint32, known *bool) error {
	value, ok := tlv.Value(tlvs, kind)
	if !ok {
		return nil
	}
	parsed, err := wdaUint32(value, name)
	if err != nil {
		return err
	}
	*dst = parsed
	*known = true
	return nil
}
