package qcom

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/damonto/wwan-go/qcom/tlv"
)

const (
	wdsTLVProfileList    = 0x01
	wdsTLVProfileID      = 0x01
	wdsTLVProfileType    = 0x10
	wdsTLVProfileName    = 0x10
	wdsTLVProfilePDPType = 0x11
	wdsTLVProfileAPN     = 0x14
	wdsTLVPCSCFUsingPCO  = 0x1F
	wdsTLVPCSCFUsingDHCP = 0x21
	wdsTLVIMCNFlag       = 0x22
)

var ErrWDSProfileNotFound = errors.New("WDS profile not found")

type WDSCreateProfileRequest struct {
	ClientID      uint8
	TransactionID uint16
	Timeout       time.Duration
	APN           string
	PDPType       WDSPDPType
}

func (r WDSCreateProfileRequest) Request() Request {
	return Request{
		Service:       ServiceWDS,
		ClientID:      r.ClientID,
		TransactionID: r.TransactionID,
		MessageID:     MessageWDSCreateProfile,
		Timeout:       r.Timeout,
		TLVs: tlv.TLVs{
			tlv.Uint(wdsTLVProfileID, uint8(WDSProfileType3GPP)),
			tlv.Bytes(wdsTLVProfileName, fmt.Appendf(nil, "wwan-go-%d", r.PDPType)),
			tlv.Uint(wdsTLVProfilePDPType, uint8(r.PDPType)),
			tlv.Bytes(wdsTLVProfileAPN, []byte(r.APN)),
		},
	}
}

type WDSDeleteProfileRequest struct {
	ClientID      uint8
	TransactionID uint16
	Timeout       time.Duration
	Profile       WDSProfileID
}

func (r WDSDeleteProfileRequest) Request() Request {
	return Request{
		Service:       ServiceWDS,
		ClientID:      r.ClientID,
		TransactionID: r.TransactionID,
		MessageID:     MessageWDSDeleteProfile,
		Timeout:       r.Timeout,
		TLVs:          tlv.TLVs{tlv.Bytes(wdsTLVProfileID, []byte{byte(r.Profile.Type), r.Profile.Index})},
	}
}

type WDSGetProfileListRequest struct {
	ClientID      uint8
	TransactionID uint16
	Timeout       time.Duration
	ProfileType   WDSProfileType
}

func (r WDSGetProfileListRequest) Request() Request {
	return Request{Service: ServiceWDS, ClientID: r.ClientID, TransactionID: r.TransactionID, MessageID: MessageWDSGetProfileList, Timeout: r.Timeout, TLVs: tlv.TLVs{tlv.Uint(wdsTLVProfileType, uint8(r.ProfileType))}}
}

type WDSGetProfileSettingsRequest struct {
	ClientID      uint8
	TransactionID uint16
	Timeout       time.Duration
	Profile       WDSProfileID
}

func (r WDSGetProfileSettingsRequest) Request() Request {
	return Request{Service: ServiceWDS, ClientID: r.ClientID, TransactionID: r.TransactionID, MessageID: MessageWDSGetProfileSettings, Timeout: r.Timeout, TLVs: tlv.TLVs{tlv.Bytes(wdsTLVProfileID, []byte{byte(r.Profile.Type), r.Profile.Index})}}
}

func (c *Client) WDSProfiles(ctx context.Context, profileType WDSProfileType) ([]WDSProfile, error) {
	var profiles []WDSProfile
	err := c.withServiceClient(ctx, ServiceWDS, func(clientID uint8) error {
		req := WDSGetProfileListRequest{ClientID: clientID, Timeout: DefaultRequestTimeout, ProfileType: profileType}.Request()
		resp, err := c.requestServiceWithTimeout(ctx, req.Service, req.ClientID, req.MessageID, req.TLVs, req.Timeout)
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

func (c *Client) WDSProfileSettings(ctx context.Context, id WDSProfileID) (WDSProfileSettings, error) {
	settings := WDSProfileSettings{ID: id}
	err := c.withServiceClient(ctx, ServiceWDS, func(clientID uint8) error {
		req := WDSGetProfileSettingsRequest{ClientID: clientID, Timeout: DefaultRequestTimeout, Profile: id}.Request()
		resp, err := c.requestServiceWithTimeout(ctx, req.Service, req.ClientID, req.MessageID, req.TLVs, req.Timeout)
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
		return WDSProfileSettings{}, fmt.Errorf("querying QMI WDS profile %d settings: %w", id.Index, err)
	}
	return settings, nil
}

// WDSCreateProfile creates a persistent 3GPP profile. The caller owns the
// returned profile and should delete it when it is no longer needed.
func (c *Client) WDSCreateProfile(ctx context.Context, apn string, pdpType WDSPDPType) (uint8, error) {
	apn = strings.TrimSpace(apn)
	if apn == "" {
		return 0, errors.New("creating QMI WDS profile: APN is required")
	}
	if pdpType > WDSPDPTypeIPv4v6 {
		return 0, fmt.Errorf("creating QMI WDS profile: unsupported PDP type %d", pdpType)
	}

	var index uint8
	err := c.withServiceClient(ctx, ServiceWDS, func(clientID uint8) error {
		req := WDSCreateProfileRequest{ClientID: clientID, Timeout: DefaultRequestTimeout, APN: apn, PDPType: pdpType}.Request()
		resp, err := c.requestServiceWithTimeout(ctx, req.Service, req.ClientID, req.MessageID, req.TLVs, req.Timeout)
		if err != nil {
			return err
		}
		if err := resultOK(resp); err != nil {
			return err
		}
		value, ok := tlv.Value(resp.TLVs, wdsTLVProfileID)
		if !ok || len(value) < 2 {
			return errors.New("parsing QMI WDS create profile: profile identifier is missing")
		}
		if WDSProfileType(value[0]) != WDSProfileType3GPP {
			return fmt.Errorf("parsing QMI WDS create profile: profile type %d is not 3GPP", value[0])
		}
		index = value[1]
		return nil
	})
	if err != nil {
		return 0, fmt.Errorf("creating QMI WDS profile: %w", err)
	}
	return index, nil
}

// WDSDeleteProfile removes a persistent 3GPP profile.
func (c *Client) WDSDeleteProfile(ctx context.Context, index uint8) error {
	err := c.withServiceClient(ctx, ServiceWDS, func(clientID uint8) error {
		req := WDSDeleteProfileRequest{ClientID: clientID, Timeout: DefaultRequestTimeout, Profile: WDSProfileID{Type: WDSProfileType3GPP, Index: index}}.Request()
		resp, err := c.requestServiceWithTimeout(ctx, req.Service, req.ClientID, req.MessageID, req.TLVs, req.Timeout)
		if err != nil {
			return err
		}
		return resultOK(resp)
	})
	if err != nil {
		return fmt.Errorf("deleting QMI WDS profile %d: %w", index, err)
	}
	return nil
}

// WDSProfileIndex returns the 3GPP profile whose APN matches apn.
func (c *Client) WDSProfileIndex(ctx context.Context, apn string) (uint8, error) {
	apn = strings.TrimSpace(apn)
	if apn == "" {
		return 0, errors.New("querying QMI WDS profile: APN is required")
	}
	profiles, err := c.WDSProfiles(ctx, WDSProfileType3GPP)
	if err != nil {
		return 0, err
	}
	for _, profile := range profiles {
		settings, err := c.WDSProfileSettings(ctx, profile.ID)
		if err != nil {
			return 0, err
		}
		if settings.APNKnown && strings.EqualFold(strings.TrimSpace(settings.APN), apn) {
			return profile.ID.Index, nil
		}
	}
	return 0, fmt.Errorf("%w: APN %q", ErrWDSProfileNotFound, apn)
}

type WDSGetProfileListResponse struct{ Profiles []WDSProfile }

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
	r.Profiles = make([]WDSProfile, 0, count)
	for range count {
		if len(rest) < 3 {
			return errors.New("parsing QMI WDS profile list: profile entry is truncated")
		}
		nameLen := int(rest[2])
		if len(rest) < 3+nameLen {
			return errors.New("parsing QMI WDS profile list: profile name is truncated")
		}
		r.Profiles = append(r.Profiles, WDSProfile{ID: WDSProfileID{Type: WDSProfileType(rest[0]), Index: rest[1]}, Name: string(rest[3 : 3+nameLen])})
		rest = rest[3+nameLen:]
	}
	return nil
}

type WDSGetProfileSettingsResponse struct{ Settings WDSProfileSettings }

func (r *WDSGetProfileSettingsResponse) UnmarshalTLVs(tlvs tlv.TLVs) error {
	id := r.Settings.ID
	*r = WDSGetProfileSettingsResponse{Settings: WDSProfileSettings{ID: id}}
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
