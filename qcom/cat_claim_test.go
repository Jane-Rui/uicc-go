package qcom

import (
	"context"
	"encoding/binary"
	"testing"

	"github.com/damonto/wwan-go/qcom/tlv"
)

func TestCATServiceState(t *testing.T) {
	tests := []struct {
		name string
		tlvs tlv.TLVs
		want CATServiceState
	}{
		{
			name: "all masks",
			tlvs: tlv.TLVs{
				tlv.Bytes(0x01, encodeCATMasks(0x00000003, 0x00000001)),
				tlv.Bytes(0x10, encodeCATMasks(0x00000004, 0x00000000)),
				tlv.Uint(0x11, uint32(0x0000001F)),
			},
			want: CATServiceState{
				RawGlobalMask:     0x00000003,
				RawClientMask:     0x00000001,
				DecodedGlobalMask: 0x00000004,
				DecodedClientMask: 0x00000000,
				FullFunctionMask:  0x0000001F,
			},
		},
		{
			name: "missing optional masks",
			want: CATServiceState{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			transport := &fakeTransport{t: t, calls: []transportCall{{
				check: func(req Request) {
					if req.Service != ServiceCAT2 || req.ClientID != 7 || req.MessageID != MessageCATGetServiceState {
						t.Fatalf("request = %+v", req)
					}
				},
				resp: successResponse(MessageCATGetServiceState, tt.tlvs...),
			}}}
			reader := &Client{
				transport:  transport,
				slot:       1,
				catService: ServiceCAT2,
				clientIDs:  map[ServiceType]uint8{ServiceCAT2: 7},
			}

			got, err := NewCAT(reader).ServiceState(context.Background())
			if err != nil {
				t.Fatalf("ServiceState() error = %v", err)
			}
			if got != tt.want {
				t.Fatalf("ServiceState() = %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestCATForceClaimEvents(t *testing.T) {
	tests := []struct {
		name       string
		clientID   uint8
		calls      []transportCall
		want       CATEventClaim
		wantErr    bool
		wantCalls  int
		wantCatCID uint8
	}{
		{
			name:     "claims without conflict",
			clientID: 7,
			calls: []transportCall{{
				check: func(req Request) {
					if req.Service != ServiceCAT2 || req.ClientID != 7 || req.MessageID != MessageCATSetEventReport {
						t.Fatalf("request = %+v", req)
					}
					assertTLV(t, req.TLVs, 0x10, []byte{0x01, 0x00, 0x00, 0x00})
					assertTLV(t, req.TLVs, 0x12, []byte{0x01})
				},
				resp: successResponse(MessageCATSetEventReport),
			}},
			want:       CATEventClaim{Service: ServiceCAT2, ClientID: 7},
			wantCalls:  1,
			wantCatCID: 7,
		},
		{
			name:     "releases lower owner",
			clientID: 5,
			calls: []transportCall{
				{
					resp: errorResponse(MessageCATSetEventReport, QMIErrorInvalidOperation, tlv.Uint(0x10, uint32(0x01))),
				},
				{
					check: func(req Request) {
						if req.MessageID != MessageCATGetServiceState {
							t.Fatalf("messageID = 0x%04X, want 0x%04X", req.MessageID, MessageCATGetServiceState)
						}
					},
					resp: successResponse(MessageCATGetServiceState, tlv.Bytes(0x01, encodeCATMasks(0x01, 0x00))),
				},
				{
					check: func(req Request) {
						if req.Service != ServiceCAT2 || req.ClientID != 1 || req.MessageID != MessageCATGetServiceState {
							t.Fatalf("request = %+v", req)
						}
					},
					resp: successResponse(MessageCATGetServiceState, tlv.Bytes(0x01, encodeCATMasks(0x01, 0x01))),
				},
				{
					check: func(req Request) {
						if req.Service != ServiceControl || req.MessageID != MessageReleaseClientID {
							t.Fatalf("request = %+v", req)
						}
						assertTLV(t, req.TLVs, 0x01, []byte{byte(ServiceCAT2), 1})
					},
					resp: successResponse(MessageReleaseClientID),
				},
				{
					resp: successResponse(MessageCATSetEventReport),
				},
			},
			want: CATEventClaim{
				Service:          ServiceCAT2,
				ClientID:         5,
				ReleasedClientID: 1,
				StateBefore:      CATServiceState{RawGlobalMask: 1},
			},
			wantCalls:  5,
			wantCatCID: 5,
		},
		{
			name:     "finds higher owner",
			clientID: 1,
			calls: []transportCall{
				{
					resp: errorResponse(MessageCATSetEventReport, QMIErrorInvalidOperation, tlv.Uint(0x10, uint32(0x01))),
				},
				{
					resp: successResponse(MessageCATGetServiceState, tlv.Bytes(0x01, encodeCATMasks(0x01, 0x00))),
				},
				{
					check: func(req Request) {
						if req.Service != ServiceCAT2 || req.ClientID != 2 || req.MessageID != MessageCATGetServiceState {
							t.Fatalf("request = %+v", req)
						}
					},
					resp: successResponse(MessageCATGetServiceState, tlv.Bytes(0x01, encodeCATMasks(0x01, 0x00))),
				},
				{
					check: func(req Request) {
						if req.Service != ServiceCAT2 || req.ClientID != 3 || req.MessageID != MessageCATGetServiceState {
							t.Fatalf("request = %+v", req)
						}
					},
					resp: successResponse(MessageCATGetServiceState, tlv.Bytes(0x01, encodeCATMasks(0x01, 0x00))),
				},
				{
					check: func(req Request) {
						if req.Service != ServiceCAT2 || req.ClientID != 4 || req.MessageID != MessageCATGetServiceState {
							t.Fatalf("request = %+v", req)
						}
					},
					resp: successResponse(MessageCATGetServiceState, tlv.Bytes(0x01, encodeCATMasks(0x01, 0x00))),
				},
				{
					check: func(req Request) {
						if req.Service != ServiceCAT2 || req.ClientID != 5 || req.MessageID != MessageCATGetServiceState {
							t.Fatalf("request = %+v", req)
						}
					},
					resp: successResponse(MessageCATGetServiceState, tlv.Bytes(0x01, encodeCATMasks(0x01, 0x01))),
				},
				{
					check: func(req Request) {
						if req.Service != ServiceControl || req.MessageID != MessageReleaseClientID {
							t.Fatalf("request = %+v", req)
						}
						assertTLV(t, req.TLVs, 0x01, []byte{byte(ServiceCAT2), 5})
					},
					resp: successResponse(MessageReleaseClientID),
				},
				{
					resp: successResponse(MessageCATSetEventReport),
				},
			},
			want: CATEventClaim{
				Service:          ServiceCAT2,
				ClientID:         1,
				ReleasedClientID: 5,
				StateBefore:      CATServiceState{RawGlobalMask: 1},
			},
			wantCalls:  8,
			wantCatCID: 1,
		},
		{
			name:     "rejects non raw conflict",
			clientID: 3,
			calls: []transportCall{
				{
					resp: errorResponse(MessageCATSetEventReport, QMIErrorInvalidOperation),
				},
				{
					resp: successResponse(MessageReleaseClientID),
				},
			},
			wantErr:    true,
			wantCalls:  2,
			wantCatCID: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			transport := &fakeTransport{t: t, calls: tt.calls}
			reader := &Client{
				transport:  transport,
				slot:       1,
				catService: ServiceCAT2,
				clientIDs:  map[ServiceType]uint8{ServiceCAT2: tt.clientID},
			}

			got, err := NewCAT(reader).ForceClaimEvents(context.Background(), CATEventClaimConfig{RawMask: 1})
			if tt.wantErr {
				if err == nil {
					t.Fatal("ForceClaimEvents() error = nil, want non-nil")
				}
			} else if err != nil {
				t.Fatalf("ForceClaimEvents() error = %v", err)
			}
			if !tt.wantErr && got != tt.want {
				t.Fatalf("ForceClaimEvents() = %+v, want %+v", got, tt.want)
			}
			if got := transport.callCount(); got != tt.wantCalls {
				t.Fatalf("Do() calls = %d, want %d", got, tt.wantCalls)
			}
			if got := reader.clientIDs[reader.catService]; got != tt.wantCatCID {
				t.Fatalf("CAT client ID = %d, want %d", got, tt.wantCatCID)
			}
		})
	}
}

func encodeCATMasks(global, client uint32) []byte {
	value := binary.LittleEndian.AppendUint32(nil, global)
	return binary.LittleEndian.AppendUint32(value, client)
}
