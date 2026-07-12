package uim

import (
	"context"
	"testing"
	"time"

	"github.com/damonto/uicc-go/qcom"
	"github.com/damonto/uicc-go/qcom/tlv"
)

func TestNASGetServingSystemRequest(t *testing.T) {
	req := NASGetServingSystemRequest{ClientID: 7}.Request()
	if req.Service != qcom.ServiceNAS || req.MessageID != qcom.MessageNASGetServingSystem {
		t.Fatalf("request = service 0x%02X message 0x%04X", req.Service, req.MessageID)
	}
	if len(req.TLVs) != 0 {
		t.Fatalf("TLVs len = %d, want 0", len(req.TLVs))
	}
}

func TestNASGetServingSystemResponseUnmarshalTLVs(t *testing.T) {
	tests := []struct {
		name       string
		tlvs       tlv.TLVs
		wantErr    bool
		wantPS     qcom.NASAttachState
		wantRadios []qcom.NASRadioInterface
	}{
		{name: "missing", wantErr: true},
		{name: "truncated aggregate", tlvs: tlv.TLVs{tlv.Bytes(nasTLVServingSystem, []byte{1, 1})}, wantErr: true},
		{name: "truncated radio list", tlvs: tlv.TLVs{tlv.Bytes(nasTLVServingSystem, []byte{1, 1, 1, 2, 2, 8})}, wantErr: true},
		{
			name:       "registered LTE",
			tlvs:       tlv.TLVs{tlv.Bytes(nasTLVServingSystem, []byte{1, 1, 1, 2, 1, 8})},
			wantPS:     qcom.NASAttachAttached,
			wantRadios: []qcom.NASRadioInterface{qcom.NASRadioInterfaceLTE},
		},
		{
			name:       "multiple radios",
			tlvs:       tlv.TLVs{tlv.Bytes(nasTLVServingSystem, []byte{1, 2, 1, 2, 2, 5, 8})},
			wantPS:     qcom.NASAttachAttached,
			wantRadios: []qcom.NASRadioInterface{qcom.NASRadioInterfaceUMTS, qcom.NASRadioInterfaceLTE},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got NASGetServingSystemResponse
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
			if got.ServingSystem.PSAttachState != tt.wantPS {
				t.Fatalf("PSAttachState = %d, want %d", got.ServingSystem.PSAttachState, tt.wantPS)
			}
			if len(got.ServingSystem.RadioInterfaces) != len(tt.wantRadios) {
				t.Fatalf("RadioInterfaces len = %d, want %d", len(got.ServingSystem.RadioInterfaces), len(tt.wantRadios))
			}
			for i := range tt.wantRadios {
				if got.ServingSystem.RadioInterfaces[i] != tt.wantRadios[i] {
					t.Fatalf("RadioInterfaces[%d] = %d, want %d", i, got.ServingSystem.RadioInterfaces[i], tt.wantRadios[i])
				}
			}
		})
	}
}

func TestReaderNASServingSystem(t *testing.T) {
	transport := &fakeTransport{t: t, calls: []transportCall{
		{resp: successResponse(qcom.MessageAllocateClientID, tlv.Bytes(0x01, []byte{byte(qcom.ServiceNAS), 5}))},
		{
			check: func(req qcom.Request) {
				if req.Service != qcom.ServiceNAS || req.ClientID != 5 || req.MessageID != qcom.MessageNASGetServingSystem {
					t.Fatalf("unexpected NAS request: %+v", req)
				}
			},
			resp: successResponse(qcom.MessageNASGetServingSystem, tlv.Bytes(nasTLVServingSystem, []byte{1, 1, 1, 2, 1, 8})),
		},
		{resp: successResponse(qcom.MessageReleaseClientID)},
	}}
	reader := &Reader{transport: transport, slot: 1}

	got, err := reader.NASServingSystem(context.Background())
	if err != nil {
		t.Fatalf("NASServingSystem() error = %v", err)
	}
	if got.RegistrationState != qcom.NASRegistrationRegistered || got.PSAttachState != qcom.NASAttachAttached {
		t.Fatalf("NASServingSystem() = %+v", got)
	}
}

func TestNASGetSysInfoRequest(t *testing.T) {
	tests := []struct {
		name string
		req  NASGetSysInfoRequest
	}{
		{
			name: "request fields",
			req: NASGetSysInfoRequest{
				ClientID:      7,
				TransactionID: 9,
				Timeout:       3 * time.Second,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.req.Request()
			if got.Service != qcom.ServiceNAS {
				t.Fatalf("Service = 0x%02X, want 0x%02X", got.Service, qcom.ServiceNAS)
			}
			if got.ClientID != tt.req.ClientID {
				t.Fatalf("ClientID = %d, want %d", got.ClientID, tt.req.ClientID)
			}
			if got.MessageID != qcom.MessageNASGetSysInfo {
				t.Fatalf("MessageID = 0x%04X, want 0x%04X", got.MessageID, qcom.MessageNASGetSysInfo)
			}
			if got.Timeout != tt.req.Timeout {
				t.Fatalf("Timeout = %v, want %v", got.Timeout, tt.req.Timeout)
			}
			if len(got.TLVs) != 0 {
				t.Fatalf("TLVs len = %d, want 0", len(got.TLVs))
			}
		})
	}
}

func TestNASGetSysInfoResponseUnmarshalTLVs(t *testing.T) {
	tests := []struct {
		name          string
		tlvs          tlv.TLVs
		wantKnown     bool
		wantSupported bool
	}{
		{name: "missing vops"},
		{name: "vops supported", tlvs: tlv.TLVs{tlv.Bytes(0x29, []byte{1})}, wantKnown: true, wantSupported: true},
		{name: "vops unsupported", tlvs: tlv.TLVs{tlv.Bytes(0x29, []byte{0})}, wantKnown: true},
		{name: "empty vops", tlvs: tlv.TLVs{tlv.Bytes(0x29, nil)}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got NASGetSysInfoResponse
			err := got.UnmarshalTLVs(tt.tlvs)
			if err != nil {
				t.Fatalf("UnmarshalTLVs() error = %v", err)
			}
			if got.SysInfo.VoPSKnown != tt.wantKnown {
				t.Fatalf("VoPSKnown = %v, want %v", got.SysInfo.VoPSKnown, tt.wantKnown)
			}
			if got.SysInfo.VoPSSupported != tt.wantSupported {
				t.Fatalf("VoPSSupported = %v, want %v", got.SysInfo.VoPSSupported, tt.wantSupported)
			}
		})
	}
}
