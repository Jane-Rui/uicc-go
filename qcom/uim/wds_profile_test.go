package uim

import (
	"testing"

	"github.com/damonto/uicc-go/qcom"
	"github.com/damonto/uicc-go/qcom/tlv"
)

func TestWDSProfileRequests(t *testing.T) {
	tests := []struct {
		name string
		req  qcom.Request
		want qcom.MessageID
	}{
		{name: "list", req: WDSGetProfileListRequest{ClientID: 3, ProfileType: qcom.WDSProfileType3GPP}.Request(), want: qcom.MessageWDSGetProfileList},
		{name: "settings", req: WDSGetProfileSettingsRequest{ClientID: 4, Profile: qcom.WDSProfileID{Type: qcom.WDSProfileType3GPP, Index: 2}}.Request(), want: qcom.MessageWDSGetProfileSettings},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.req.Service != qcom.ServiceWDS || tt.req.MessageID != tt.want {
				t.Fatalf("request = service 0x%02X message 0x%04X", tt.req.Service, tt.req.MessageID)
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
			got := WDSGetProfileSettingsResponse{Settings: qcom.WDSProfileSettings{ID: qcom.WDSProfileID{Index: 2}}}
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
