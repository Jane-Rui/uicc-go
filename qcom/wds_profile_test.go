package qcom

import (
	"context"
	"testing"

	"github.com/damonto/wwan-go/qcom/tlv"
)

func TestWDSProfileRequests(t *testing.T) {
	tests := []struct {
		name string
		req  Request
		want MessageID
	}{
		{name: "list", req: WDSGetProfileListRequest{ClientID: 3, ProfileType: WDSProfileType3GPP}.Request(), want: MessageWDSGetProfileList},
		{name: "settings", req: WDSGetProfileSettingsRequest{ClientID: 4, Profile: WDSProfileID{Type: WDSProfileType3GPP, Index: 2}}.Request(), want: MessageWDSGetProfileSettings},
		{name: "create", req: WDSCreateProfileRequest{ClientID: 5, APN: "ims", PDPType: WDSPDPTypeIPv4v6}.Request(), want: MessageWDSCreateProfile},
		{name: "delete", req: WDSDeleteProfileRequest{ClientID: 6, Profile: WDSProfileID{Type: WDSProfileType3GPP, Index: 3}}.Request(), want: MessageWDSDeleteProfile},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.req.Service != ServiceWDS || tt.req.MessageID != tt.want {
				t.Fatalf("request = service 0x%02X message 0x%04X", tt.req.Service, tt.req.MessageID)
			}
		})
	}
}

func TestClientWDSProfileLifecycleReusesClient(t *testing.T) {
	tests := []struct {
		name    string
		apn     string
		pdpType WDSPDPType
	}{
		{name: "IPv4v6 profile", apn: " ims ", pdpType: WDSPDPTypeIPv4v6},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			transport := &fakeTransport{t: t, calls: []transportCall{
				{resp: allocatedClientResponse(ServiceWDS, 5)},
				{
					check: func(req Request) {
						if req.ClientID != 5 || req.MessageID != MessageWDSCreateProfile {
							t.Fatalf("create request = client %d message 0x%04X", req.ClientID, req.MessageID)
						}
						assertTLV(t, req.TLVs, wdsTLVProfileID, []byte{byte(WDSProfileType3GPP)})
						assertTLV(t, req.TLVs, wdsTLVProfilePDPType, []byte{byte(tt.pdpType)})
						assertTLV(t, req.TLVs, wdsTLVProfileAPN, []byte("ims"))
					},
					resp: successResponse(MessageWDSCreateProfile, tlv.Bytes(wdsTLVProfileID, []byte{byte(WDSProfileType3GPP), 7})),
				},
				{
					check: func(req Request) {
						if req.ClientID != 5 || req.MessageID != MessageWDSDeleteProfile {
							t.Fatalf("delete request = client %d message 0x%04X", req.ClientID, req.MessageID)
						}
						assertTLV(t, req.TLVs, wdsTLVProfileID, []byte{byte(WDSProfileType3GPP), 7})
					},
					resp: successResponse(MessageWDSDeleteProfile),
				},
				{
					check: func(req Request) {
						assertTLV(t, req.TLVs, 0x01, []byte{byte(ServiceWDS), 5})
					},
					resp: successResponse(MessageReleaseClientID),
				},
			}}
			reader := &Client{transport: transport, slot: 1}

			index, err := reader.WDSCreateProfile(context.Background(), tt.apn, tt.pdpType)
			if err != nil {
				t.Fatalf("WDSCreateProfile() error = %v", err)
			}
			if index != 7 {
				t.Fatalf("WDSCreateProfile() = %d, want 7", index)
			}
			if err := reader.WDSDeleteProfile(context.Background(), index); err != nil {
				t.Fatalf("WDSDeleteProfile() error = %v", err)
			}
			if err := reader.Close(); err != nil {
				t.Fatalf("Close() error = %v", err)
			}
			if got := transport.callCount(); got != len(transport.calls) {
				t.Fatalf("Do() calls = %d, want %d", got, len(transport.calls))
			}
		})
	}
}

func TestClientWDSProfileIndex(t *testing.T) {
	tests := []struct {
		name string
		apn  string
		want uint8
	}{
		{name: "case-insensitive APN", apn: " IMS ", want: 2},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			profileList := []byte{2, 0, 1, 3, 'n', 'e', 't', 0, 2, 3, 'i', 'm', 's'}
			transport := &fakeTransport{t: t, calls: []transportCall{
				{resp: allocatedClientResponse(ServiceWDS, 5)},
				{resp: successResponse(MessageWDSGetProfileList, tlv.Bytes(wdsTLVProfileList, profileList))},
				{resp: successResponse(MessageWDSGetProfileSettings, tlv.Bytes(wdsTLVProfileAPN, []byte("internet")))},
				{resp: successResponse(MessageWDSGetProfileSettings, tlv.Bytes(wdsTLVProfileAPN, []byte("ims")))},
				{resp: successResponse(MessageReleaseClientID)},
			}}
			reader := &Client{transport: transport, slot: 1}

			got, err := reader.WDSProfileIndex(context.Background(), tt.apn)
			if err != nil {
				t.Fatalf("WDSProfileIndex() error = %v", err)
			}
			if got != tt.want {
				t.Fatalf("WDSProfileIndex() = %d, want %d", got, tt.want)
			}
			if err := reader.Close(); err != nil {
				t.Fatalf("Close() error = %v", err)
			}
			if got := transport.callCount(); got != len(transport.calls) {
				t.Fatalf("Do() calls = %d, want %d", got, len(transport.calls))
			}
		})
	}
}

func TestWDSGetProfileListResponseUnmarshalTLVs(t *testing.T) {
	tests := []struct {
		name    string
		tlvs    tlv.TLVs
		wantErr bool
	}{
		{name: "missing", wantErr: true},
		{name: "truncated", tlvs: tlv.TLVs{tlv.Bytes(wdsTLVProfileList, []byte{1, 0})}, wantErr: true},
		{name: "profiles", tlvs: tlv.TLVs{tlv.Bytes(wdsTLVProfileList, []byte{2, 0, 1, 3, 'n', 'e', 't', 0, 2, 3, 'i', 'm', 's'})}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got WDSGetProfileListResponse
			err := got.UnmarshalTLVs(tt.tlvs)
			if tt.wantErr {
				if err == nil {
					t.Fatal("UnmarshalTLVs() error = nil, want non-nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("UnmarshalTLVs() error = %v", err)
			}
			if len(got.Profiles) != 2 || got.Profiles[1].ID.Index != 2 || got.Profiles[1].Name != "ims" {
				t.Fatalf("Profiles = %+v", got.Profiles)
			}
		})
	}
}

func TestWDSGetProfileSettingsResponseUnmarshalTLVs(t *testing.T) {
	tests := []struct {
		name    string
		tlvs    tlv.TLVs
		wantErr bool
	}{
		{name: "empty"},
		{name: "IMS", tlvs: tlv.TLVs{tlv.Bytes(wdsTLVProfileAPN, []byte("ims")), tlv.Bytes(wdsTLVPCSCFUsingPCO, []byte{1}), tlv.Bytes(wdsTLVIMCNFlag, []byte{1})}},
		{name: "truncated bool", tlvs: tlv.TLVs{tlv.Bytes(wdsTLVPCSCFUsingPCO, nil)}, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := WDSGetProfileSettingsResponse{Settings: WDSProfileSettings{ID: WDSProfileID{Index: 2}}}
			err := got.UnmarshalTLVs(tt.tlvs)
			if tt.wantErr {
				if err == nil {
					t.Fatal("UnmarshalTLVs() error = nil, want non-nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("UnmarshalTLVs() error = %v", err)
			}
			if tt.name == "IMS" && (!got.Settings.APNKnown || got.Settings.APN != "ims" || !got.Settings.PCSCFUsingPCO || !got.Settings.IMCN) {
				t.Fatalf("Settings = %+v", got.Settings)
			}
		})
	}
}
