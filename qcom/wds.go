package qcom

import (
	"encoding/binary"
	"errors"
	"net"
	"slices"
	"time"

	"github.com/damonto/wwan-go/qcom/tlv"
)

// WDSLegacyBindMuxDataPortRequest encodes legacy QMI WDS Bind Data Port.
type WDSLegacyBindMuxDataPortRequest struct {
	ClientID      uint8
	TransactionID uint16
	Timeout       time.Duration
	DataPort      WDSSIOPort
}

// Request binds the WDS client to a legacy SIO data port.
func (r WDSLegacyBindMuxDataPortRequest) Request() Request {
	return Request{
		Service:       ServiceWDS,
		ClientID:      r.ClientID,
		TransactionID: r.TransactionID,
		MessageID:     MessageWDSLegacyBindMuxDataPort,
		Timeout:       r.Timeout,
		TLVs: tlv.TLVs{
			tlv.Uint(0x01, uint16(r.DataPort)),
		},
	}
}

// WDSBindMuxDataPortRequest encodes QMI WDS Bind Mux Data Port.
type WDSBindMuxDataPortRequest struct {
	ClientID      uint8
	TransactionID uint16
	Timeout       time.Duration
	DataPort      WDSMuxDataPort
}

// Request binds the WDS client to a logical data channel.
func (r WDSBindMuxDataPortRequest) Request() Request {
	tlvs := make(tlv.TLVs, 0, 4)
	if r.DataPort.Endpoint != nil {
		endpoint, _ := r.DataPort.Endpoint.MarshalBinary() // Fixed-width endpoint encoding cannot fail.
		tlvs = append(tlvs, tlv.Bytes(0x10, endpoint))
	}
	tlvs = append(tlvs, tlv.Uint(0x11, r.DataPort.MuxID))
	if r.DataPort.Reversed {
		tlvs = append(tlvs, tlv.Uint(0x12, uint8(1)))
	}
	if r.DataPort.ClientType != WDSClientTypeReserved {
		tlvs = append(tlvs, tlv.Uint(0x13, uint32(r.DataPort.ClientType)))
	}
	return Request{
		Service:       ServiceWDS,
		ClientID:      r.ClientID,
		TransactionID: r.TransactionID,
		MessageID:     MessageWDSBindMuxDataPort,
		Timeout:       r.Timeout,
		TLVs:          tlvs,
	}
}

// WDSStartNetworkInterfaceRequest encodes QMI WDS Start Network Interface.
type WDSStartNetworkInterfaceRequest struct {
	ClientID             uint8
	TransactionID        uint16
	Timeout              time.Duration
	APN                  string
	IPPreference         WDSIPPreference
	TechnologyPreference WDSTechnologyPreference
	ProfileIndex3GPP     uint8
	CallType             *WDSCallType
}

// Request converts the high-level request fields into a QMI WDS request.
func (r WDSStartNetworkInterfaceRequest) Request() Request {
	var tlvs tlv.TLVs
	if r.APN != "" {
		tlvs = append(tlvs, tlv.Bytes(0x14, []byte(r.APN)))
	}
	if r.IPPreference != WDSIPPreferenceDefault {
		tlvs = append(tlvs, tlv.Uint(0x19, uint8(r.IPPreference)))
	}
	if r.TechnologyPreference != 0 {
		tlvs = append(tlvs, tlv.Uint(0x30, uint8(r.TechnologyPreference)))
	}
	if r.ProfileIndex3GPP != 0 {
		tlvs = append(tlvs, tlv.Uint(0x31, r.ProfileIndex3GPP))
	}
	if r.CallType != nil {
		tlvs = append(tlvs, tlv.Uint(0x35, uint8(*r.CallType)))
	}
	return Request{
		Service:       ServiceWDS,
		ClientID:      r.ClientID,
		TransactionID: r.TransactionID,
		MessageID:     MessageWDSStartNetworkInterface,
		Timeout:       r.Timeout,
		TLVs:          tlvs,
	}
}

// WDSStartNetworkInterfaceResponse is the parsed WDS start network response.
type WDSStartNetworkInterfaceResponse struct {
	PacketDataHandle        uint32
	CallEndReason           WDSCallEndReason
	HasCallEndReason        bool
	VerboseCallEndReason    WDSVerboseCallEndReason
	HasVerboseCallEndReason bool
}

// UnmarshalTLVs reads the packet data handle returned by the modem.
func (r *WDSStartNetworkInterfaceResponse) UnmarshalTLVs(tlvs tlv.TLVs) error {
	*r = WDSStartNetworkInterfaceResponse{}

	if value, ok := tlv.Value(tlvs, 0x01); ok {
		if len(value) < 4 {
			return errors.New("parsing WDS start network response: packet data handle TLV is truncated")
		}
		r.PacketDataHandle = binary.LittleEndian.Uint32(value[:4])
	}
	if value, ok := tlv.Value(tlvs, 0x10); ok {
		if len(value) < 2 {
			return errors.New("parsing WDS start network response: call end reason TLV is truncated")
		}
		r.CallEndReason = WDSCallEndReason(binary.LittleEndian.Uint16(value[:2]))
		r.HasCallEndReason = true
	}
	if value, ok := tlv.Value(tlvs, 0x11); ok {
		if len(value) < 4 {
			return errors.New("parsing WDS start network response: verbose call end reason TLV is truncated")
		}
		r.VerboseCallEndReason = WDSVerboseCallEndReason{
			Type:   WDSVerboseCallEndReasonType(binary.LittleEndian.Uint16(value[:2])),
			Reason: int16(binary.LittleEndian.Uint16(value[2:4])),
		}
		r.HasVerboseCallEndReason = true
	}
	return nil
}

// WDSStopNetworkInterfaceRequest encodes QMI WDS Stop Network Interface.
type WDSStopNetworkInterfaceRequest struct {
	ClientID         uint8
	TransactionID    uint16
	Timeout          time.Duration
	PacketDataHandle uint32
}

// Request converts the stop request into a QMI WDS request.
func (r WDSStopNetworkInterfaceRequest) Request() Request {
	return Request{
		Service:       ServiceWDS,
		ClientID:      r.ClientID,
		TransactionID: r.TransactionID,
		MessageID:     MessageWDSStopNetworkInterface,
		Timeout:       r.Timeout,
		TLVs: tlv.TLVs{
			tlv.Uint(0x01, r.PacketDataHandle),
		},
	}
}

// WDSGetRuntimeSettingsRequest encodes QMI WDS Get Runtime Settings.
type WDSGetRuntimeSettingsRequest struct {
	ClientID          uint8
	TransactionID     uint16
	Timeout           time.Duration
	RequestedSettings WDSRuntimeSettingsMask
}

// Request converts the runtime-settings selector into a QMI WDS request.
func (r WDSGetRuntimeSettingsRequest) Request() Request {
	requested := r.RequestedSettings
	if requested == 0 {
		requested = WDSRuntimeRequestedIMSSettings
	}
	return Request{
		Service:       ServiceWDS,
		ClientID:      r.ClientID,
		TransactionID: r.TransactionID,
		MessageID:     MessageWDSGetRuntimeSettings,
		Timeout:       r.Timeout,
		TLVs: tlv.TLVs{
			tlv.Uint(0x10, uint32(requested)),
		},
	}
}

// WDSGetRuntimeSettingsResponse is the parsed WDS runtime settings response.
type WDSGetRuntimeSettingsResponse struct {
	Settings WDSRuntimeSettings
}

// UnmarshalTLVs parses IMS PDN addressing and P-CSCF data.
func (r *WDSGetRuntimeSettingsResponse) UnmarshalTLVs(tlvs tlv.TLVs) error {
	*r = WDSGetRuntimeSettingsResponse{}
	if value, ok := tlv.Value(tlvs, 0x1E); ok {
		if len(value) < 4 {
			return errors.New("parsing WDS runtime settings: IPv4 address TLV is truncated")
		}
		r.Settings.LocalIPv4 = qmiIPv4(value)
	}
	for _, kind := range []byte{0x15, 0x16} {
		if value, ok := tlv.Value(tlvs, kind); ok {
			if len(value) < 4 {
				return errors.New("parsing WDS runtime settings: IPv4 DNS TLV is truncated")
			}
			r.Settings.DNS = append(r.Settings.DNS, qmiIPv4(value))
		}
	}
	if value, ok := tlv.Value(tlvs, 0x20); ok {
		if len(value) < 4 {
			return errors.New("parsing WDS runtime settings: IPv4 gateway TLV is truncated")
		}
		r.Settings.IPv4Gateway = qmiIPv4(value)
	}
	if value, ok := tlv.Value(tlvs, 0x21); ok {
		if len(value) < 4 {
			return errors.New("parsing WDS runtime settings: IPv4 subnet mask TLV is truncated")
		}
		r.Settings.IPv4SubnetMask = qmiIPv4(value)
	}
	if value, ok := tlv.Value(tlvs, 0x25); ok {
		if len(value) < 17 {
			return errors.New("parsing WDS runtime settings: IPv6 address TLV is truncated")
		}
		r.Settings.LocalIPv6 = slices.Clone(value[:16])
		r.Settings.IPv6PrefixLength = value[16]
	}
	if value, ok := tlv.Value(tlvs, 0x26); ok {
		if len(value) < 17 {
			return errors.New("parsing WDS runtime settings: IPv6 gateway TLV is truncated")
		}
		r.Settings.IPv6Gateway = slices.Clone(value[:16])
	}
	for _, kind := range []byte{0x27, 0x28} {
		if value, ok := tlv.Value(tlvs, kind); ok {
			if len(value) < 16 {
				return errors.New("parsing WDS runtime settings: IPv6 DNS TLV is truncated")
			}
			r.Settings.DNS = append(r.Settings.DNS, slices.Clone(value[:16]))
		}
	}
	if value, ok := tlv.Value(tlvs, 0x29); ok {
		if len(value) < 4 {
			return errors.New("parsing WDS runtime settings: MTU TLV is truncated")
		}
		r.Settings.MTU = binary.LittleEndian.Uint32(value[:4])
	}
	if value, ok := tlv.Value(tlvs, 0x23); ok {
		ips, err := parseWDSIPv4List(value)
		if err != nil {
			return err
		}
		r.Settings.PCSCFIPs = append(r.Settings.PCSCFIPs, ips...)
	}
	if value, ok := tlv.Value(tlvs, 0x2E); ok {
		ips, err := parseWDSIPv6List(value)
		if err != nil {
			return err
		}
		r.Settings.PCSCFIPs = append(r.Settings.PCSCFIPs, ips...)
	}
	if value, ok := tlv.Value(tlvs, 0x2B); ok {
		if len(value) < 1 {
			return errors.New("parsing WDS runtime settings: IP family TLV is truncated")
		}
		family := WDSIPFamily(value[0])
		if family != WDSIPFamilyIPv4 && family != WDSIPFamilyIPv6 {
			return errors.New("parsing WDS runtime settings: IP family is invalid")
		}
		r.Settings.IPFamily = family
	}
	if value, ok := tlv.Value(tlvs, 0x2C); ok && len(value) > 0 {
		r.Settings.IMCN = value[0] == 1
	}
	r.Settings.PCSCFIPs = uniqueWDSIPs(r.Settings.PCSCFIPs)
	return nil
}

func parseWDSIPv4List(value []byte) ([]net.IP, error) {
	if len(value) == 0 {
		return nil, errors.New("parsing WDS runtime settings: P-CSCF IPv4 list TLV is truncated")
	}
	count := int(value[0])
	offset := 1
	ips := make([]net.IP, 0, count)
	for range count {
		if len(value) < offset+4 {
			return nil, errors.New("parsing WDS runtime settings: P-CSCF IPv4 list value is truncated")
		}
		ips = append(ips, qmiIPv4(value[offset:offset+4]))
		offset += 4
	}
	return ips, nil
}

func qmiIPv4(value []byte) net.IP {
	return net.IPv4(value[3], value[2], value[1], value[0])
}

func parseWDSIPv6List(value []byte) ([]net.IP, error) {
	if len(value) == 0 {
		return nil, errors.New("parsing WDS runtime settings: P-CSCF IPv6 list TLV is truncated")
	}
	count := int(value[0])
	offset := 1
	ips := make([]net.IP, 0, count)
	for range count {
		if len(value) < offset+16 {
			return nil, errors.New("parsing WDS runtime settings: P-CSCF IPv6 list value is truncated")
		}
		ips = append(ips, slices.Clone(value[offset:offset+16]))
		offset += 16
	}
	return ips, nil
}

func uniqueWDSIPs(ips []net.IP) []net.IP {
	unique := make([]net.IP, 0, len(ips))
	for _, ip := range ips {
		if len(ip) == 0 || slices.ContainsFunc(unique, ip.Equal) {
			continue
		}
		unique = append(unique, slices.Clone(ip))
	}
	return unique
}
