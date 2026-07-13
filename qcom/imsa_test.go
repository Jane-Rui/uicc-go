package qcom

import (
	"context"
	"testing"
	"time"

	"github.com/damonto/wwan-go/qcom/tlv"
)

func TestIMSARequestEncoding(t *testing.T) {
	tests := []struct {
		name          string
		req           Request
		wantMessageID MessageID
	}{
		{
			name: "registration status",
			req: IMSAGetRegistrationStatusRequest{
				ClientID:      7,
				TransactionID: 9,
				Timeout:       3 * time.Second,
			}.Request(),
			wantMessageID: MessageIMSAGetRegistrationStatus,
		},
		{
			name: "service status",
			req: IMSAGetServiceStatusRequest{
				ClientID:      8,
				TransactionID: 10,
				Timeout:       4 * time.Second,
			}.Request(),
			wantMessageID: MessageIMSAGetServiceStatus,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.req.Service != ServiceIMSA {
				t.Fatalf("Service = 0x%02X, want 0x%02X", tt.req.Service, ServiceIMSA)
			}
			if tt.req.MessageID != tt.wantMessageID {
				t.Fatalf("MessageID = 0x%04X, want 0x%04X", tt.req.MessageID, tt.wantMessageID)
			}
			if len(tt.req.TLVs) != 0 {
				t.Fatalf("TLVs len = %d, want 0", len(tt.req.TLVs))
			}
		})
	}
}

func TestIMSARegistrationStatusResponseUnmarshalTLVs(t *testing.T) {
	tests := []struct {
		name        string
		tlvs        tlv.TLVs
		wantErr     bool
		wantKnown   bool
		wantStatus  IMSRegistrationStatus
		wantFailure bool
		wantCode    uint16
	}{
		{name: "missing status"},
		{
			name:       "new status registered",
			tlvs:       tlv.TLVs{tlv.Uint(imsaTLVRegStatus, uint32(IMSRegistrationStatusRegistered))},
			wantKnown:  true,
			wantStatus: IMSRegistrationStatusRegistered,
		},
		{
			name:       "new status wins over old boolean",
			tlvs:       tlv.TLVs{tlv.Bytes(imsaTLVIMSRegistered, []byte{0}), tlv.Uint(imsaTLVRegStatus, uint32(IMSRegistrationStatusRegistering))},
			wantKnown:  true,
			wantStatus: IMSRegistrationStatusRegistering,
		},
		{
			name:       "old boolean registered",
			tlvs:       tlv.TLVs{tlv.Bytes(imsaTLVIMSRegistered, []byte{1})},
			wantKnown:  true,
			wantStatus: IMSRegistrationStatusRegistered,
		},
		{
			name:        "failure code",
			tlvs:        tlv.TLVs{tlv.Uint(imsaTLVRegStatus, uint32(IMSRegistrationStatusNotRegistered)), tlv.Uint(imsaTLVFailureCode, uint16(403))},
			wantKnown:   true,
			wantFailure: true,
			wantCode:    403,
		},
		{name: "truncated new status", tlvs: tlv.TLVs{tlv.Bytes(imsaTLVRegStatus, []byte{1})}, wantErr: true},
		{name: "truncated old boolean", tlvs: tlv.TLVs{tlv.Bytes(imsaTLVIMSRegistered, nil)}, wantErr: true},
		{name: "truncated failure code", tlvs: tlv.TLVs{tlv.Bytes(imsaTLVFailureCode, []byte{1})}, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got IMSARegistrationStatusResponse
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
			if got.Status.RegistrationKnown != tt.wantKnown {
				t.Fatalf("RegistrationKnown = %v, want %v", got.Status.RegistrationKnown, tt.wantKnown)
			}
			if got.Status.Registration != tt.wantStatus {
				t.Fatalf("Registration = %d, want %d", got.Status.Registration, tt.wantStatus)
			}
			if got.Status.FailureCodeKnown != tt.wantFailure {
				t.Fatalf("FailureCodeKnown = %v, want %v", got.Status.FailureCodeKnown, tt.wantFailure)
			}
			if got.Status.FailureCode != tt.wantCode {
				t.Fatalf("FailureCode = %d, want %d", got.Status.FailureCode, tt.wantCode)
			}
		})
	}
}

func TestIMSAServiceStatusResponseUnmarshalTLVs(t *testing.T) {
	tests := []struct {
		name            string
		tlvs            tlv.TLVs
		wantErr         bool
		wantService     IMSServiceStatus
		wantServiceSeen bool
		wantRAT         IMSServiceRAT
		wantRATSeen     bool
	}{
		{name: "missing service"},
		{
			name:            "volte service",
			tlvs:            tlv.TLVs{tlv.Uint(imsaTLVVoIPService, uint32(IMSServiceStatusFullService)), tlv.Uint(imsaTLVVoIPRAT, uint32(IMSServiceRATWWAN))},
			wantService:     IMSServiceStatusFullService,
			wantServiceSeen: true,
			wantRAT:         IMSServiceRATWWAN,
			wantRATSeen:     true,
		},
		{name: "truncated service", tlvs: tlv.TLVs{tlv.Bytes(imsaTLVVoIPService, []byte{1})}, wantErr: true},
		{name: "truncated rat", tlvs: tlv.TLVs{tlv.Bytes(imsaTLVVoIPRAT, []byte{1})}, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got IMSAServiceStatusResponse
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
			if got.Status.VoIPServiceKnown != tt.wantServiceSeen {
				t.Fatalf("VoIPServiceKnown = %v, want %v", got.Status.VoIPServiceKnown, tt.wantServiceSeen)
			}
			if got.Status.VoIPService != tt.wantService {
				t.Fatalf("VoIPService = %d, want %d", got.Status.VoIPService, tt.wantService)
			}
			if got.Status.VoIPRATKnown != tt.wantRATSeen {
				t.Fatalf("VoIPRATKnown = %v, want %v", got.Status.VoIPRATKnown, tt.wantRATSeen)
			}
			if got.Status.VoIPRAT != tt.wantRAT {
				t.Fatalf("VoIPRAT = %d, want %d", got.Status.VoIPRAT, tt.wantRAT)
			}
		})
	}
}

func TestClientIMSAStatus(t *testing.T) {
	transport := &fakeTransport{
		t: t,
		calls: []transportCall{
			{
				check: func(req Request) {
					if req.Service != ServiceControl || req.MessageID != MessageAllocateClientID {
						t.Fatalf("allocate request = service 0x%02X message 0x%04X", req.Service, req.MessageID)
					}
					assertTLV(t, req.TLVs, 0x01, []byte{byte(ServiceIMSA)})
				},
				resp: successResponse(MessageAllocateClientID, tlv.Bytes(0x01, []byte{byte(ServiceIMSA), 5})),
			},
			{
				check: func(req Request) {
					if req.Service != ServiceIMSA || req.ClientID != 5 || req.MessageID != MessageIMSAGetRegistrationStatus {
						t.Fatalf("request = service 0x%02X client %d message 0x%04X, want IMSA registration", req.Service, req.ClientID, req.MessageID)
					}
				},
				resp: successResponse(MessageIMSAGetRegistrationStatus, tlv.Uint(imsaTLVRegStatus, uint32(IMSRegistrationStatusRegistered))),
			},
			{
				check: func(req Request) {
					if req.Service != ServiceIMSA || req.ClientID != 5 || req.MessageID != MessageIMSAGetServiceStatus {
						t.Fatalf("request = service 0x%02X client %d message 0x%04X, want IMSA service", req.Service, req.ClientID, req.MessageID)
					}
				},
				resp: successResponse(MessageIMSAGetServiceStatus,
					tlv.Uint(imsaTLVVoIPService, uint32(IMSServiceStatusFullService)),
					tlv.Uint(imsaTLVVoIPRAT, uint32(IMSServiceRATWWAN))),
			},
			{
				check: func(req Request) {
					if req.Service != ServiceControl || req.MessageID != MessageReleaseClientID {
						t.Fatalf("release request = service 0x%02X message 0x%04X", req.Service, req.MessageID)
					}
					assertTLV(t, req.TLVs, 0x01, []byte{byte(ServiceIMSA), 5})
				},
				resp: successResponse(MessageReleaseClientID),
			},
		},
	}
	reader := &Client{
		transport: transport,
		slot:      1,
	}

	got, err := reader.IMSAStatus(context.Background())
	if err != nil {
		t.Fatalf("IMSAStatus() error = %v", err)
	}
	if !got.IMSRegistered() {
		t.Fatal("IMSRegistered() = false, want true")
	}
	if !got.VoLTERegistered() {
		t.Fatal("VoLTERegistered() = false, want true")
	}
	if err := reader.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}
	if got := transport.callCount(); got != 4 {
		t.Fatalf("Do() calls = %d, want 4", got)
	}
}
