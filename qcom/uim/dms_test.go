package uim

import (
	"bytes"
	"context"
	"testing"
	"time"

	"github.com/damonto/uicc-go/qcom"
	"github.com/damonto/uicc-go/qcom/tlv"
)

func TestDMSRequestEncoding(t *testing.T) {
	tests := []struct {
		name          string
		req           qcom.Request
		wantMessageID qcom.MessageID
		wantTLV       byte
		wantValue     []byte
	}{
		{
			name: "get MSISDN",
			req: DMSGetMSISDNRequest{
				ClientID:      6,
				TransactionID: 8,
				Timeout:       2 * time.Second,
			}.Request(),
			wantMessageID: qcom.MessageDMSGetMSISDN,
		},
		{
			name: "get operating mode",
			req: DMSGetOperatingModeRequest{
				ClientID:      7,
				TransactionID: 9,
				Timeout:       3 * time.Second,
			}.Request(),
			wantMessageID: qcom.MessageDMSGetOperatingMode,
		},
		{
			name: "set operating mode",
			req: DMSSetOperatingModeRequest{
				ClientID:      8,
				TransactionID: 10,
				Timeout:       4 * time.Second,
				Mode:          qcom.DMSOperatingModeLowPower,
			}.Request(),
			wantMessageID: qcom.MessageDMSSetOperatingMode,
			wantTLV:       dmsTLVOperatingMode,
			wantValue:     []byte{byte(qcom.DMSOperatingModeLowPower)},
		},
		{
			name: "set event report",
			req: DMSSetEventReportRequest{
				ClientID:            9,
				TransactionID:       11,
				Timeout:             5 * time.Second,
				ReportOperatingMode: true,
			}.Request(),
			wantMessageID: qcom.MessageDMSSetEventReport,
			wantTLV:       dmsTLVReportOperatingMode,
			wantValue:     []byte{1},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.req.Service != qcom.ServiceDMS {
				t.Fatalf("Service = 0x%02X, want 0x%02X", tt.req.Service, qcom.ServiceDMS)
			}
			if tt.req.MessageID != tt.wantMessageID {
				t.Fatalf("MessageID = 0x%04X, want 0x%04X", tt.req.MessageID, tt.wantMessageID)
			}
			if tt.wantTLV == 0 {
				if len(tt.req.TLVs) != 0 {
					t.Fatalf("TLVs len = %d, want 0", len(tt.req.TLVs))
				}
				return
			}

			value, ok := tlv.Value(tt.req.TLVs, tt.wantTLV)
			if !ok {
				t.Fatalf("TLV 0x%02X missing", tt.wantTLV)
			}
			if !bytes.Equal(value, tt.wantValue) {
				t.Fatalf("TLV 0x%02X = % X, want % X", tt.wantTLV, value, tt.wantValue)
			}
		})
	}
}

func TestDMSGetMSISDNResponseUnmarshalTLVs(t *testing.T) {
	tests := []struct {
		name    string
		tlvs    tlv.TLVs
		want    DMSGetMSISDNResponse
		wantErr bool
	}{
		{
			name: "voice number only",
			tlvs: tlv.TLVs{tlv.Bytes(dmsTLVVoiceNumber, []byte("+8613800138000"))},
			want: DMSGetMSISDNResponse{VoiceNumber: "+8613800138000"},
		},
		{
			name: "all identifiers",
			tlvs: tlv.TLVs{
				tlv.Bytes(dmsTLVVoiceNumber, []byte("+15551234567")),
				tlv.Bytes(dmsTLVMobileIDNumber, []byte("5551234567")),
				tlv.Bytes(dmsTLVIMSI, []byte("310260123456789")),
			},
			want: DMSGetMSISDNResponse{
				VoiceNumber:    "+15551234567",
				MobileIDNumber: "5551234567",
				IMSI:           "310260123456789",
			},
		},
		{name: "empty voice number", tlvs: tlv.TLVs{tlv.Bytes(dmsTLVVoiceNumber, nil)}},
		{name: "missing voice number", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got DMSGetMSISDNResponse
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
			if got != tt.want {
				t.Fatalf("UnmarshalTLVs() = %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestReaderMSISDN(t *testing.T) {
	tests := []struct {
		name string
		tlvs tlv.TLVs
		want DMSGetMSISDNResponse
	}{
		{
			name: "voice number",
			tlvs: tlv.TLVs{tlv.Bytes(dmsTLVVoiceNumber, []byte("+8613800138000"))},
			want: DMSGetMSISDNResponse{VoiceNumber: "+8613800138000"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			transport := &fakeTransport{
				t: t,
				calls: []transportCall{
					{
						check: func(req qcom.Request) {
							if req.Service != qcom.ServiceControl || req.MessageID != qcom.MessageAllocateClientID {
								t.Fatalf("allocate request = service 0x%02X message 0x%04X", req.Service, req.MessageID)
							}
							assertTLV(t, req.TLVs, 0x01, []byte{byte(qcom.ServiceDMS)})
						},
						resp: successResponse(qcom.MessageAllocateClientID, tlv.Bytes(0x01, []byte{byte(qcom.ServiceDMS), 5})),
					},
					{
						check: func(req qcom.Request) {
							if req.Service != qcom.ServiceDMS || req.ClientID != 5 || req.MessageID != qcom.MessageDMSGetMSISDN {
								t.Fatalf("unexpected DMS request: %+v", req)
							}
							if len(req.TLVs) != 0 {
								t.Fatalf("TLVs len = %d, want 0", len(req.TLVs))
							}
						},
						resp: successResponse(qcom.MessageDMSGetMSISDN, tt.tlvs...),
					},
					{
						check: func(req qcom.Request) {
							if req.Service != qcom.ServiceControl || req.MessageID != qcom.MessageReleaseClientID {
								t.Fatalf("release request = service 0x%02X message 0x%04X", req.Service, req.MessageID)
							}
							assertTLV(t, req.TLVs, 0x01, []byte{byte(qcom.ServiceDMS), 5})
						},
						resp: successResponse(qcom.MessageReleaseClientID),
					},
				},
			}
			reader := &Reader{transport: transport, slot: 1}

			got, err := reader.MSISDN(context.Background())
			if err != nil {
				t.Fatalf("MSISDN() error = %v", err)
			}
			if got != tt.want {
				t.Fatalf("MSISDN() = %+v, want %+v", got, tt.want)
			}
			if got := transport.callCount(); got != 3 {
				t.Fatalf("Do() calls = %d, want 3", got)
			}
		})
	}
}

func TestReaderMSISDNUsesBoundDMSService(t *testing.T) {
	tests := []struct {
		name string
		want DMSGetMSISDNResponse
	}{
		{
			name: "QRTR-style DMS transport",
			want: DMSGetMSISDNResponse{VoiceNumber: "+8613800138000"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			transport := &serviceBoundFakeTransport{
				fakeTransport: fakeTransport{
					t: t,
					calls: []transportCall{{
						check: func(req qcom.Request) {
							if req.Service != qcom.ServiceDMS || req.ClientID != 0 || req.MessageID != qcom.MessageDMSGetMSISDN {
								t.Fatalf("unexpected DMS request: %+v", req)
							}
						},
						resp: successResponse(qcom.MessageDMSGetMSISDN, tlv.Bytes(dmsTLVVoiceNumber, []byte(tt.want.VoiceNumber))),
					}},
				},
				service: qcom.ServiceDMS,
			}
			reader, err := New(context.Background(), transport)
			if err != nil {
				t.Fatalf("New() error = %v", err)
			}

			got, err := reader.MSISDN(context.Background())
			if err != nil {
				t.Fatalf("MSISDN() error = %v", err)
			}
			if got != tt.want {
				t.Fatalf("MSISDN() = %+v, want %+v", got, tt.want)
			}
			if got := transport.callCount(); got != 1 {
				t.Fatalf("Do() calls = %d, want 1", got)
			}
		})
	}
}

func TestDMSGetOperatingModeResponseUnmarshalTLVs(t *testing.T) {
	tests := []struct {
		name    string
		tlvs    tlv.TLVs
		want    qcom.DMSOperatingMode
		wantErr bool
	}{
		{
			name: "online",
			tlvs: tlv.TLVs{tlv.Bytes(dmsTLVOperatingMode, []byte{byte(qcom.DMSOperatingModeOnline)})},
			want: qcom.DMSOperatingModeOnline,
		},
		{name: "missing", wantErr: true},
		{name: "truncated", tlvs: tlv.TLVs{tlv.Bytes(dmsTLVOperatingMode, nil)}, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got DMSGetOperatingModeResponse
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
			if got.Mode != tt.want {
				t.Fatalf("Mode = %d, want %d", got.Mode, tt.want)
			}
		})
	}
}

func TestSetOperatingModeUsesDMSOnly(t *testing.T) {
	tests := []struct {
		name string
		mode qcom.DMSOperatingMode
	}{
		{name: "low power", mode: qcom.DMSOperatingModeLowPower},
		{name: "online", mode: qcom.DMSOperatingModeOnline},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			transport := &fakeTransport{
				t: t,
				calls: []transportCall{
					{
						check: func(req qcom.Request) {
							if req.Service != qcom.ServiceControl || req.MessageID != qcom.MessageAllocateClientID {
								t.Fatalf("allocate request = service 0x%02X message 0x%04X", req.Service, req.MessageID)
							}
							assertTLV(t, req.TLVs, 0x01, []byte{byte(qcom.ServiceDMS)})
						},
						resp: successResponse(qcom.MessageAllocateClientID, tlv.Bytes(0x01, []byte{byte(qcom.ServiceDMS), 5})),
					},
					{
						check: func(req qcom.Request) {
							if req.Service != qcom.ServiceDMS {
								t.Fatalf("Service = 0x%02X, want DMS", req.Service)
							}
							if req.ClientID != 5 {
								t.Fatalf("ClientID = %d, want 5", req.ClientID)
							}
							if req.MessageID != qcom.MessageDMSSetOperatingMode {
								t.Fatalf("MessageID = 0x%04X, want 0x%04X", req.MessageID, qcom.MessageDMSSetOperatingMode)
							}
							assertTLV(t, req.TLVs, dmsTLVOperatingMode, []byte{byte(tt.mode)})
						},
						resp: successResponse(qcom.MessageDMSSetOperatingMode),
					},
					{
						check: func(req qcom.Request) {
							if req.Service != qcom.ServiceControl || req.MessageID != qcom.MessageReleaseClientID {
								t.Fatalf("release request = service 0x%02X message 0x%04X", req.Service, req.MessageID)
							}
							assertTLV(t, req.TLVs, 0x01, []byte{byte(qcom.ServiceDMS), 5})
						},
						resp: successResponse(qcom.MessageReleaseClientID),
					},
				},
			}
			reader := &Reader{
				transport: transport,
				slot:      1,
			}

			if err := reader.SetOperatingMode(context.Background(), tt.mode); err != nil {
				t.Fatalf("SetOperatingMode() error = %v", err)
			}
			if got := transport.callCount(); got != 3 {
				t.Fatalf("Do() calls = %d, want 3", got)
			}
		})
	}
}
