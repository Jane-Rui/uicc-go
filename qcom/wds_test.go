package qcom

import (
	"bytes"
	"encoding/binary"
	"net"
	"slices"
	"testing"
	"time"

	"github.com/damonto/wwan-go/qcom/tlv"
)

func TestWDSRequestEncoding(t *testing.T) {
	tests := []struct {
		name          string
		req           Request
		wantService   ServiceType
		wantClientID  uint8
		wantMessageID MessageID
		wantTimeout   time.Duration
		wantTLV       byte
		wantValue     []byte
	}{
		{
			name: "start network",
			req: WDSStartNetworkInterfaceRequest{
				ClientID:             7,
				TransactionID:        9,
				Timeout:              3 * time.Second,
				APN:                  "ims",
				IPPreference:         WDSIPPreferenceUnspecified,
				TechnologyPreference: WDSTechnologyPreference3GPP,
			}.Request(),
			wantService:   ServiceWDS,
			wantClientID:  7,
			wantMessageID: MessageWDSStartNetworkInterface,
			wantTimeout:   3 * time.Second,
			wantTLV:       0x14,
			wantValue:     []byte("ims"),
		},
		{
			name: "stop network",
			req: WDSStopNetworkInterfaceRequest{
				ClientID:         8,
				TransactionID:    10,
				Timeout:          4 * time.Second,
				PacketDataHandle: 0x01020304,
			}.Request(),
			wantService:   ServiceWDS,
			wantClientID:  8,
			wantMessageID: MessageWDSStopNetworkInterface,
			wantTimeout:   4 * time.Second,
			wantTLV:       0x01,
			wantValue:     []byte{0x04, 0x03, 0x02, 0x01},
		},
		{
			name: "runtime settings",
			req: WDSGetRuntimeSettingsRequest{
				ClientID:      9,
				TransactionID: 11,
				Timeout:       5 * time.Second,
			}.Request(),
			wantService:   ServiceWDS,
			wantClientID:  9,
			wantMessageID: MessageWDSGetRuntimeSettings,
			wantTimeout:   5 * time.Second,
			wantTLV:       0x10,
			wantValue:     uint32ValueForTest(uint32(WDSRuntimeRequestedIMSSettings)),
		},
		{
			name: "legacy bind mux data port",
			req: WDSLegacyBindMuxDataPortRequest{
				ClientID:      10,
				TransactionID: 12,
				Timeout:       6 * time.Second,
				DataPort:      WDSSIOPortA2MuxRMNET1,
			}.Request(),
			wantService:   ServiceWDS,
			wantClientID:  10,
			wantMessageID: MessageWDSLegacyBindMuxDataPort,
			wantTimeout:   6 * time.Second,
			wantTLV:       0x01,
			wantValue:     []byte{0x05, 0x0E},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.req.Service != tt.wantService {
				t.Fatalf("Service = 0x%02X, want 0x%02X", tt.req.Service, tt.wantService)
			}
			if tt.req.ClientID != tt.wantClientID {
				t.Fatalf("ClientID = %d, want %d", tt.req.ClientID, tt.wantClientID)
			}
			if tt.req.MessageID != tt.wantMessageID {
				t.Fatalf("MessageID = 0x%04X, want 0x%04X", tt.req.MessageID, tt.wantMessageID)
			}
			if tt.req.Timeout != tt.wantTimeout {
				t.Fatalf("Timeout = %v, want %v", tt.req.Timeout, tt.wantTimeout)
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

func TestWDSStartNetworkInterfaceRequestKeepsConnectionTLVsWithProfile(t *testing.T) {
	tests := []struct {
		name string
		req  WDSStartNetworkInterfaceRequest
		want map[byte][]byte
	}{
		{
			name: "profile and IPv6 IMS settings",
			req: WDSStartNetworkInterfaceRequest{
				APN:                  "ims",
				IPPreference:         WDSIPPreferenceIPv6,
				TechnologyPreference: WDSTechnologyPreference3GPP,
				ProfileIndex3GPP:     2,
			},
			want: map[byte][]byte{
				0x14: []byte("ims"),
				0x19: {byte(WDSIPPreferenceIPv6)},
				0x30: {byte(WDSTechnologyPreference3GPP)},
				0x31: {2},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := tt.req.Request()
			for kind, want := range tt.want {
				got, ok := tlv.Value(req.TLVs, kind)
				if !ok || !bytes.Equal(got, want) {
					t.Fatalf("TLV 0x%02X = % X, want % X", kind, got, want)
				}
			}
		})
	}
}

func TestWDSStartNetworkInterfaceRequestOptionalCallType(t *testing.T) {
	embedded := WDSCallTypeEmbedded
	tests := []struct {
		name     string
		callType *WDSCallType
		want     []byte
	}{
		{name: "omitted"},
		{name: "embedded", callType: &embedded, want: []byte{byte(WDSCallTypeEmbedded)}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := WDSStartNetworkInterfaceRequest{CallType: tt.callType}.Request()
			got, ok := tlv.Value(req.TLVs, 0x35)
			if ok != (tt.want != nil) {
				t.Fatalf("Call type TLV present = %v, want %v", ok, tt.want != nil)
			}
			if !bytes.Equal(got, tt.want) {
				t.Fatalf("Call type TLV = % X, want % X", got, tt.want)
			}
		})
	}
}

func TestWDSBindMuxDataPortRequestEncoding(t *testing.T) {
	tests := []struct {
		name         string
		dataPort     WDSMuxDataPort
		wantEndpoint []byte
		wantMuxID    byte
		wantReversed bool
		wantClient   *WDSClientType
	}{
		{
			name: "endpoint and mux",
			dataPort: WDSMuxDataPort{
				Endpoint: &DataEndpoint{
					Type:        DataEndpointBAMDMUX,
					InterfaceID: 1,
				},
				MuxID: 2,
			},
			wantEndpoint: []byte{0x05, 0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00},
			wantMuxID:    2,
		},
		{
			name:      "mux without endpoint",
			dataPort:  WDSMuxDataPort{MuxID: 1},
			wantMuxID: 1,
		},
		{
			name: "reversed port",
			dataPort: WDSMuxDataPort{
				Endpoint: &DataEndpoint{Type: DataEndpointHSUSB, InterfaceID: 4},
				MuxID:    3,
				Reversed: true,
			},
			wantEndpoint: []byte{0x02, 0x00, 0x00, 0x00, 0x04, 0x00, 0x00, 0x00},
			wantMuxID:    3,
			wantReversed: true,
		},
		{
			name: "tethered client",
			dataPort: WDSMuxDataPort{
				MuxID:      4,
				ClientType: WDSClientTypeTethered,
			},
			wantMuxID:  4,
			wantClient: ptr(WDSClientTypeTethered),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := WDSBindMuxDataPortRequest{
				ClientID:      7,
				TransactionID: 9,
				Timeout:       3 * time.Second,
				DataPort:      tt.dataPort,
			}.Request()
			if req.Service != ServiceWDS {
				t.Fatalf("Service = 0x%02X, want 0x%02X", req.Service, ServiceWDS)
			}
			if req.ClientID != 7 {
				t.Fatalf("ClientID = %d, want 7", req.ClientID)
			}
			if req.TransactionID != 9 {
				t.Fatalf("TransactionID = %d, want 9", req.TransactionID)
			}
			if req.MessageID != MessageWDSBindMuxDataPort {
				t.Fatalf("MessageID = 0x%04X, want 0x%04X", req.MessageID, MessageWDSBindMuxDataPort)
			}
			if req.Timeout != 3*time.Second {
				t.Fatalf("Timeout = %v, want %v", req.Timeout, 3*time.Second)
			}

			endpoint, hasEndpoint := tlv.Value(req.TLVs, 0x10)
			if hasEndpoint != (tt.wantEndpoint != nil) {
				t.Fatalf("Endpoint TLV present = %v, want %v", hasEndpoint, tt.wantEndpoint != nil)
			}
			if !bytes.Equal(endpoint, tt.wantEndpoint) {
				t.Fatalf("Endpoint TLV = % X, want % X", endpoint, tt.wantEndpoint)
			}
			muxID, ok := tlv.Value(req.TLVs, 0x11)
			if !ok {
				t.Fatal("Mux ID TLV missing")
			}
			if !bytes.Equal(muxID, []byte{tt.wantMuxID}) {
				t.Fatalf("Mux ID TLV = % X, want %02X", muxID, tt.wantMuxID)
			}
			reversed, hasReversed := tlv.Value(req.TLVs, 0x12)
			if hasReversed != tt.wantReversed {
				t.Fatalf("Reversed TLV present = %v, want %v", hasReversed, tt.wantReversed)
			}
			if tt.wantReversed && !bytes.Equal(reversed, []byte{1}) {
				t.Fatalf("Reversed TLV = % X, want 01", reversed)
			}
			clientType, hasClientType := tlv.Value(req.TLVs, 0x13)
			if hasClientType != (tt.wantClient != nil) {
				t.Fatalf("Client type TLV present = %v, want %v", hasClientType, tt.wantClient != nil)
			}
			if tt.wantClient != nil && !bytes.Equal(clientType, uint32ValueForTest(uint32(*tt.wantClient))) {
				t.Fatalf("Client type TLV = % X, want % X", clientType, uint32ValueForTest(uint32(*tt.wantClient)))
			}
		})
	}
}

func TestWDSGetRuntimeSettingsResponseUnmarshalTLVs(t *testing.T) {
	localIPv6 := net.ParseIP("2001:db8::1").To16()
	pcscfIPv6 := net.ParseIP("2001:db8::2").To16()

	tests := []struct {
		name        string
		tlvs        tlv.TLVs
		wantErr     bool
		wantIPv4    net.IP
		wantIPv6    net.IP
		wantPCSCF   []net.IP
		wantDNS     []net.IP
		wantGateway net.IP
		wantMask    net.IP
		wantMTU     uint32
		wantFamily  WDSIPFamily
		wantIMCN    bool
	}{
		{
			name: "runtime addresses and pcscf",
			tlvs: tlv.TLVs{
				tlv.Bytes(0x1E, []byte{3, 2, 1, 10}),
				tlv.Bytes(0x25, append(slices.Clone(localIPv6), 64)),
				tlv.Bytes(0x15, []byte{8, 8, 8, 8}),
				tlv.Bytes(0x20, []byte{1, 2, 1, 10}),
				tlv.Bytes(0x21, []byte{0, 255, 255, 255}),
				tlv.Uint(0x29, uint32(1428)),
				tlv.Bytes(0x23, ipv4ListForTest(net.IPv4(198, 51, 100, 10))),
				tlv.Bytes(0x2E, ipv6ListForTest(pcscfIPv6)),
				tlv.Bytes(0x2B, []byte{byte(WDSIPFamilyIPv6)}),
				tlv.Bytes(0x2C, []byte{1}),
			},
			wantIPv4:    net.IPv4(10, 1, 2, 3),
			wantIPv6:    localIPv6,
			wantPCSCF:   []net.IP{net.IPv4(198, 51, 100, 10), pcscfIPv6},
			wantDNS:     []net.IP{net.IPv4(8, 8, 8, 8)},
			wantGateway: net.IPv4(10, 1, 2, 1),
			wantMask:    net.IPv4(255, 255, 255, 0),
			wantMTU:     1428,
			wantFamily:  WDSIPFamilyIPv6,
			wantIMCN:    true,
		},
		{
			name:    "short local ipv4 fails",
			tlvs:    tlv.TLVs{tlv.Bytes(0x1E, []byte{10, 1, 2})},
			wantErr: true,
		},
		{
			name:    "truncated pcscf ipv4 list fails",
			tlvs:    tlv.TLVs{tlv.Bytes(0x23, []byte{1, 198})},
			wantErr: true,
		},
		{
			name:    "invalid active IP family fails",
			tlvs:    tlv.TLVs{tlv.Bytes(0x2B, []byte{byte(WDSIPPreferenceUnspecified)})},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got WDSGetRuntimeSettingsResponse
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
			if !got.Settings.LocalIPv4.Equal(tt.wantIPv4) {
				t.Fatalf("LocalIPv4 = %v, want %v", got.Settings.LocalIPv4, tt.wantIPv4)
			}
			if !got.Settings.LocalIPv6.Equal(tt.wantIPv6) {
				t.Fatalf("LocalIPv6 = %v, want %v", got.Settings.LocalIPv6, tt.wantIPv6)
			}
			if got.Settings.IPFamily != tt.wantFamily {
				t.Fatalf("IPFamily = %d, want %d", got.Settings.IPFamily, tt.wantFamily)
			}
			if got.Settings.IMCN != tt.wantIMCN {
				t.Fatalf("IMCN = %v, want %v", got.Settings.IMCN, tt.wantIMCN)
			}
			if len(got.Settings.PCSCFIPs) != len(tt.wantPCSCF) {
				t.Fatalf("PCSCFIPs len = %d, want %d", len(got.Settings.PCSCFIPs), len(tt.wantPCSCF))
			}
			for i, want := range tt.wantPCSCF {
				if !got.Settings.PCSCFIPs[i].Equal(want) {
					t.Fatalf("PCSCFIPs[%d] = %v, want %v", i, got.Settings.PCSCFIPs[i], want)
				}
			}
			if len(got.Settings.DNS) != len(tt.wantDNS) {
				t.Fatalf("DNS len = %d, want %d", len(got.Settings.DNS), len(tt.wantDNS))
			}
			for i, want := range tt.wantDNS {
				if !got.Settings.DNS[i].Equal(want) {
					t.Fatalf("DNS[%d] = %v, want %v", i, got.Settings.DNS[i], want)
				}
			}
			if !got.Settings.IPv4Gateway.Equal(tt.wantGateway) {
				t.Fatalf("IPv4Gateway = %v, want %v", got.Settings.IPv4Gateway, tt.wantGateway)
			}
			if !got.Settings.IPv4SubnetMask.Equal(tt.wantMask) {
				t.Fatalf("IPv4SubnetMask = %v, want %v", got.Settings.IPv4SubnetMask, tt.wantMask)
			}
			if got.Settings.MTU != tt.wantMTU {
				t.Fatalf("MTU = %d, want %d", got.Settings.MTU, tt.wantMTU)
			}
		})
	}
}

func ptr[T any](value T) *T { return &value }

func TestWDSStartNetworkInterfaceResponseUnmarshalTLVs(t *testing.T) {
	tests := []struct {
		name           string
		tlvs           tlv.TLVs
		wantErr        bool
		want           uint32
		wantCallEnd    WDSCallEndReason
		wantVerbose    WDSVerboseCallEndReason
		wantHasCallEnd bool
		wantHasVerbose bool
	}{
		{name: "handle present", tlvs: tlv.TLVs{tlv.Bytes(0x01, []byte{4, 3, 2, 1})}, want: 0x01020304},
		{name: "handle missing"},
		{
			name: "call end reasons without handle",
			tlvs: tlv.TLVs{
				tlv.Bytes(0x10, uint16ValueForTest(uint16(WDSCallEndReasonGenericUnspecified))),
				tlv.Bytes(0x11, verboseCallEndReasonForTest(WDSVerboseCallEndReasonTypeInternal, 241)),
			},
			wantCallEnd: WDSCallEndReasonGenericUnspecified,
			wantVerbose: WDSVerboseCallEndReason{
				Type:   WDSVerboseCallEndReasonTypeInternal,
				Reason: WDSVerboseCallEndReasonInternalInterfaceInUseConfigMatch,
			},
			wantHasCallEnd: true,
			wantHasVerbose: true,
		},
		{name: "short handle fails", tlvs: tlv.TLVs{tlv.Bytes(0x01, []byte{1, 2})}, wantErr: true},
		{name: "short call end reason fails", tlvs: tlv.TLVs{tlv.Bytes(0x10, []byte{1})}, wantErr: true},
		{name: "short verbose call end reason fails", tlvs: tlv.TLVs{tlv.Bytes(0x11, []byte{2, 0, 1})}, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got WDSStartNetworkInterfaceResponse
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
			if got.PacketDataHandle != tt.want {
				t.Fatalf("PacketDataHandle = 0x%08X, want 0x%08X", got.PacketDataHandle, tt.want)
			}
			if got.HasCallEndReason != tt.wantHasCallEnd {
				t.Fatalf("HasCallEndReason = %v, want %v", got.HasCallEndReason, tt.wantHasCallEnd)
			}
			if got.CallEndReason != tt.wantCallEnd {
				t.Fatalf("CallEndReason = %d, want %d", got.CallEndReason, tt.wantCallEnd)
			}
			if got.HasVerboseCallEndReason != tt.wantHasVerbose {
				t.Fatalf("HasVerboseCallEndReason = %v, want %v", got.HasVerboseCallEndReason, tt.wantHasVerbose)
			}
			if got.VerboseCallEndReason != tt.wantVerbose {
				t.Fatalf("VerboseCallEndReason = %+v, want %+v", got.VerboseCallEndReason, tt.wantVerbose)
			}
		})
	}
}

func uint16ValueForTest(value uint16) []byte {
	out := make([]byte, 2)
	binary.LittleEndian.PutUint16(out, value)
	return out
}

func uint32ValueForTest(value uint32) []byte {
	out := make([]byte, 4)
	binary.LittleEndian.PutUint32(out, value)
	return out
}

func verboseCallEndReasonForTest(reasonType WDSVerboseCallEndReasonType, reason int16) []byte {
	out := make([]byte, 4)
	binary.LittleEndian.PutUint16(out[:2], uint16(reasonType))
	binary.LittleEndian.PutUint16(out[2:], uint16(reason))
	return out
}

func ipv4ListForTest(ip net.IP) []byte {
	out := make([]byte, 5)
	out[0] = 1
	v4 := ip.To4()
	out[1], out[2], out[3], out[4] = v4[3], v4[2], v4[1], v4[0]
	return out
}

func ipv6ListForTest(ip net.IP) []byte {
	out := make([]byte, 17)
	out[0] = 1
	copy(out[1:], ip.To16())
	return out
}
