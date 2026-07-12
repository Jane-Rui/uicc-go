package qcom

import (
	"bytes"
	"context"
	"testing"
	"time"

	"github.com/damonto/wwan-go/qcom/tlv"
)

func TestWDARequestEncoding(t *testing.T) {
	rawIP := WDALinkLayerRawIP
	qmap := WDAAggregationQMAP
	disabled := WDAAggregationDisabled
	flowControl := true
	padding := uint32(4)

	tests := []struct {
		name        string
		req         Request
		wantMessage MessageID
		wantTLVs    map[byte][]byte
	}{
		{
			name: "get default data port",
			req: WDAGetDataFormatRequest{
				ClientID:      7,
				TransactionID: 9,
				Timeout:       3 * time.Second,
			}.Request(),
			wantMessage: MessageWDAGetDataFormat,
		},
		{
			name: "get BAM DMUX endpoint",
			req: WDAGetDataFormatRequest{
				ClientID:      8,
				TransactionID: 10,
				Endpoint: &DataEndpoint{
					Type:        DataEndpointBAMDMUX,
					InterfaceID: 1,
				},
			}.Request(),
			wantMessage: MessageWDAGetDataFormat,
			wantTLVs: map[byte][]byte{
				wdaTLVGetEndpoint: {0x05, 0, 0, 0, 1, 0, 0, 0},
			},
		},
		{
			name: "set raw IP",
			req: WDASetDataFormatRequest{
				ClientID:      9,
				TransactionID: 11,
				Config: WDADataFormatConfig{
					LinkLayerProtocol: &rawIP,
				},
			}.Request(),
			wantMessage: MessageWDASetDataFormat,
			wantTLVs: map[byte][]byte{
				wdaTLVLinkProtocol: {0x02, 0, 0, 0},
			},
		},
		{
			name: "set QMAP aggregation",
			req: WDASetDataFormatRequest{
				Config: WDADataFormatConfig{
					UplinkAggregation:            &qmap,
					DownlinkAggregation:          &disabled,
					DownlinkMinimumPadding:       &padding,
					TerminalEquipmentFlowControl: &flowControl,
				},
			}.Request(),
			wantMessage: MessageWDASetDataFormat,
			wantTLVs: map[byte][]byte{
				wdaTLVUplinkAggregation:   {0x05, 0, 0, 0},
				wdaTLVDownlinkAggregation: {0, 0, 0, 0},
				wdaTLVSetDownlinkPadding:  {4, 0, 0, 0},
				wdaTLVSetFlowControl:      {1},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.req.Service != ServiceWDA {
				t.Fatalf("Service = 0x%02X, want 0x%02X", tt.req.Service, ServiceWDA)
			}
			if tt.req.MessageID != tt.wantMessage {
				t.Fatalf("MessageID = 0x%04X, want 0x%04X", tt.req.MessageID, tt.wantMessage)
			}
			if len(tt.req.TLVs) != len(tt.wantTLVs) {
				t.Fatalf("TLVs len = %d, want %d", len(tt.req.TLVs), len(tt.wantTLVs))
			}
			for kind, want := range tt.wantTLVs {
				got, ok := tlv.Value(tt.req.TLVs, kind)
				if !ok {
					t.Fatalf("TLV 0x%02X missing", kind)
				}
				if !bytes.Equal(got, want) {
					t.Fatalf("TLV 0x%02X = % X, want % X", kind, got, want)
				}
			}
		})
	}
}

func TestWDADataFormatResponseUnmarshalTLVs(t *testing.T) {
	tests := []struct {
		name    string
		tlvs    tlv.TLVs
		check   func(*testing.T, WDADataFormat)
		wantErr bool
	}{
		{
			name: "raw IP and QMAP",
			tlvs: tlv.TLVs{
				tlv.Uint(wdaTLVQoSFormat, uint8(0)),
				tlv.Uint(wdaTLVLinkProtocol, uint32(WDALinkLayerRawIP)),
				tlv.Uint(wdaTLVUplinkAggregation, uint32(WDAAggregationQMAP)),
				tlv.Uint(wdaTLVDownlinkAggregation, uint32(WDAAggregationQMAP)),
				tlv.Uint(wdaTLVDownlinkMaxDatagrams, uint32(32)),
				tlv.Uint(wdaTLVDownlinkMaxSize, uint32(32768)),
				tlv.Uint(wdaTLVUplinkMaxDatagrams, uint32(16)),
				tlv.Uint(wdaTLVUplinkMaxSize, uint32(16384)),
				tlv.Uint(wdaTLVResponseQoSHeader, uint32(WDAQoSHeader6Bytes)),
				tlv.Uint(wdaTLVResponseDownlinkPadding, uint32(4)),
				tlv.Uint(wdaTLVResponseFlowControl, uint8(1)),
			},
			check: func(t *testing.T, got WDADataFormat) {
				if !got.LinkLayerProtocolKnown || got.LinkLayerProtocol != WDALinkLayerRawIP {
					t.Fatalf("LinkLayerProtocol = %d, known %v", got.LinkLayerProtocol, got.LinkLayerProtocolKnown)
				}
				if !got.UplinkAggregationKnown || got.UplinkAggregation != WDAAggregationQMAP {
					t.Fatalf("UplinkAggregation = %d, known %v", got.UplinkAggregation, got.UplinkAggregationKnown)
				}
				if !got.DownlinkMaxSizeKnown || got.DownlinkMaxSize != 32768 {
					t.Fatalf("DownlinkMaxSize = %d, known %v", got.DownlinkMaxSize, got.DownlinkMaxSizeKnown)
				}
				if !got.UplinkMaxDatagramsKnown || got.UplinkMaxDatagrams != 16 {
					t.Fatalf("UplinkMaxDatagrams = %d, known %v", got.UplinkMaxDatagrams, got.UplinkMaxDatagramsKnown)
				}
				if !got.TerminalEquipmentFlowControlKnown || !got.TerminalEquipmentFlowControl {
					t.Fatalf("TerminalEquipmentFlowControl = %v, known %v", got.TerminalEquipmentFlowControl, got.TerminalEquipmentFlowControlKnown)
				}
			},
		},
		{
			name: "optional fields absent",
			check: func(t *testing.T, got WDADataFormat) {
				if got.LinkLayerProtocolKnown || got.QoSEnabledKnown {
					t.Fatalf("known flags set for empty response: %+v", got)
				}
			},
		},
		{
			name:    "truncated link protocol",
			tlvs:    tlv.TLVs{tlv.Bytes(wdaTLVLinkProtocol, []byte{2})},
			wantErr: true,
		},
		{
			name:    "truncated flow control",
			tlvs:    tlv.TLVs{tlv.Bytes(wdaTLVResponseFlowControl, nil)},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var response WDADataFormatResponse
			err := response.UnmarshalTLVs(tt.tlvs)
			if tt.wantErr {
				if err == nil {
					t.Fatal("UnmarshalTLVs() error = nil, want non-nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("UnmarshalTLVs() error = %v", err)
			}
			tt.check(t, response.Format)
		})
	}
}

func TestSetWDALinkLayerProtocolUsesWDA(t *testing.T) {
	tests := []struct {
		name     string
		protocol WDALinkLayerProtocol
	}{
		{name: "raw IP", protocol: WDALinkLayerRawIP},
		{name: "Ethernet", protocol: WDALinkLayerEthernet},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			transport := &fakeTransport{
				t: t,
				calls: []transportCall{
					{
						check: func(req Request) {
							assertTLV(t, req.TLVs, 0x01, []byte{byte(ServiceWDA)})
						},
						resp: successResponse(MessageAllocateClientID, tlv.Bytes(0x01, []byte{byte(ServiceWDA), 5})),
					},
					{
						check: func(req Request) {
							if req.Service != ServiceWDA || req.ClientID != 5 || req.MessageID != MessageWDASetDataFormat {
								t.Fatalf("WDA request = service 0x%02X client %d message 0x%04X", req.Service, req.ClientID, req.MessageID)
							}
							assertTLV(t, req.TLVs, wdaTLVLinkProtocol, uint32ValueForTest(uint32(tt.protocol)))
						},
						resp: successResponse(MessageWDASetDataFormat, tlv.Uint(wdaTLVLinkProtocol, uint32(tt.protocol))),
					},
					{
						check: func(req Request) {
							assertTLV(t, req.TLVs, 0x01, []byte{byte(ServiceWDA), 5})
						},
						resp: successResponse(MessageReleaseClientID),
					},
				},
			}
			reader := &Client{transport: transport, slot: 1}

			if err := reader.SetWDALinkLayerProtocol(context.Background(), tt.protocol); err != nil {
				t.Fatalf("SetWDALinkLayerProtocol() error = %v", err)
			}
			if err := reader.Close(); err != nil {
				t.Fatalf("Close() error = %v", err)
			}
			if got := transport.callCount(); got != 3 {
				t.Fatalf("Do() calls = %d, want 3", got)
			}
		})
	}
}
