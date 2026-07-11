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
	wdsTLVProfileList    = 0x01
	wdsTLVProfileID      = 0x01
	wdsTLVProfileType    = 0x10
	wdsTLVProfileName    = 0x10
	wdsTLVProfileAPN     = 0x14
	wdsTLVPCSCFUsingPCO  = 0x1F
	wdsTLVPCSCFUsingDHCP = 0x21
	wdsTLVIMCNFlag       = 0x22
)

type WDSGetProfileListRequest struct {
	ClientID      uint8
	TransactionID uint16
	Timeout       time.Duration
	ProfileType   qcom.WDSProfileType
}

func (r WDSGetProfileListRequest) Request() qcom.Request {
	return qcom.Request{Service: qcom.ServiceWDS, ClientID: r.ClientID, TransactionID: r.TransactionID, MessageID: qcom.MessageWDSGetProfileList, Timeout: r.Timeout, TLVs: tlv.TLVs{tlv.Uint(wdsTLVProfileType, uint8(r.ProfileType))}}
}

type WDSGetProfileSettingsRequest struct {
	ClientID      uint8
	TransactionID uint16
	Timeout       time.Duration
	Profile       qcom.WDSProfileID
}

func (r WDSGetProfileSettingsRequest) Request() qcom.Request {
	return qcom.Request{Service: qcom.ServiceWDS, ClientID: r.ClientID, TransactionID: r.TransactionID, MessageID: qcom.MessageWDSGetProfileSettings, Timeout: r.Timeout, TLVs: tlv.TLVs{tlv.Bytes(wdsTLVProfileID, []byte{byte(r.Profile.Type), r.Profile.Index})}}
}

func (r *Reader) WDSProfiles(ctx context.Context, profileType qcom.WDSProfileType) ([]qcom.WDSProfile, error) {
	var profiles []qcom.WDSProfile
	err := r.withServiceClient(ctx, qcom.ServiceWDS, func(clientID uint8) error {
		req := WDSGetProfileListRequest{ClientID: clientID, Timeout: DefaultRequestTimeout, ProfileType: profileType}.Request()
		resp, err := r.requestServiceWithTimeout(ctx, req.Service, req.ClientID, req.MessageID, req.TLVs, req.Timeout)
		if err != nil {
			return err
		}
		if err := resultOK(resp); err != nil {
			return err
		}
		var parsed WDSGetProfileListResponse
		if err := parsed.UnmarshalTLVs(resp.TLVs); err != nil {
			return err
		}
		profiles = parsed.Profiles
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("querying QMI WDS profiles: %w", err)
	}
	return profiles, nil
}

func (r *Reader) WDSProfileSettings(ctx context.Context, id qcom.WDSProfileID) (qcom.WDSProfileSettings, error) {
	settings := qcom.WDSProfileSettings{ID: id}
	err := r.withServiceClient(ctx, qcom.ServiceWDS, func(clientID uint8) error {
		req := WDSGetProfileSettingsRequest{ClientID: clientID, Timeout: DefaultRequestTimeout, Profile: id}.Request()
		resp, err := r.requestServiceWithTimeout(ctx, req.Service, req.ClientID, req.MessageID, req.TLVs, req.Timeout)
		if err != nil {
			return err
		}
		if err := resultOK(resp); err != nil {
			return err
		}
		var parsed WDSGetProfileSettingsResponse
		parsed.Settings.ID = id
		if err := parsed.UnmarshalTLVs(resp.TLVs); err != nil {
			return err
		}
		parsed.Settings.ID = id
		settings = parsed.Settings
		return nil
	})
	if err != nil {
		return qcom.WDSProfileSettings{}, fmt.Errorf("querying QMI WDS profile %d settings: %w", id.Index, err)
	}
	return settings, nil
}

type WDSGetProfileListResponse struct{ Profiles []qcom.WDSProfile }

func (r *WDSGetProfileListResponse) UnmarshalTLVs(tlvs tlv.TLVs) error {
	*r = WDSGetProfileListResponse{}
	value, ok := tlv.Value(tlvs, wdsTLVProfileList)
	if !ok {
		return errors.New("parsing QMI WDS profile list: profile list TLV missing")
	}
	if len(value) < 1 {
		return errors.New("parsing QMI WDS profile list: profile list TLV is truncated")
	}
	count, rest := int(value[0]), value[1:]
	r.Profiles = make([]qcom.WDSProfile, 0, count)
	for range count {
		if len(rest) < 3 {
			return errors.New("parsing QMI WDS profile list: profile entry is truncated")
		}
		nameLen := int(rest[2])
		if len(rest) < 3+nameLen {
			return errors.New("parsing QMI WDS profile list: profile name is truncated")
		}
		r.Profiles = append(r.Profiles, qcom.WDSProfile{ID: qcom.WDSProfileID{Type: qcom.WDSProfileType(rest[0]), Index: rest[1]}, Name: string(rest[3 : 3+nameLen])})
		rest = rest[3+nameLen:]
	}
	return nil
}

type WDSGetProfileSettingsResponse struct{ Settings qcom.WDSProfileSettings }

func (r *WDSGetProfileSettingsResponse) UnmarshalTLVs(tlvs tlv.TLVs) error {
	id := r.Settings.ID
	*r = WDSGetProfileSettingsResponse{Settings: qcom.WDSProfileSettings{ID: id}}
	if value, ok := tlv.Value(tlvs, wdsTLVProfileName); ok {
		r.Settings.Name, r.Settings.NameKnown = string(value), true
	}
	if value, ok := tlv.Value(tlvs, wdsTLVProfileAPN); ok {
		r.Settings.APN, r.Settings.APNKnown = string(value), true
	}
	readBool := func(kind byte, known *bool, target *bool) error {
		value, ok := tlv.Value(tlvs, kind)
		if !ok {
			return nil
		}
		if len(value) < 1 {
			return fmt.Errorf("parsing QMI WDS profile settings: TLV 0x%02X is truncated", kind)
		}
		*known, *target = true, value[0] != 0
		return nil
	}
	if err := readBool(wdsTLVPCSCFUsingPCO, &r.Settings.PCSCFUsingPCOKnown, &r.Settings.PCSCFUsingPCO); err != nil {
		return err
	}
	if err := readBool(wdsTLVPCSCFUsingDHCP, &r.Settings.PCSCFUsingDHCPKnown, &r.Settings.PCSCFUsingDHCP); err != nil {
		return err
	}
	return readBool(wdsTLVIMCNFlag, &r.Settings.IMCNKnown, &r.Settings.IMCN)
}
