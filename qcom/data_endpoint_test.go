package qcom

import (
	"bytes"
	"encoding"
	"testing"
)

var (
	_ encoding.BinaryMarshaler = DataEndpoint{}
	_ encoding.BinaryAppender  = DataEndpoint{}
)

func TestDataEndpointBinaryEncoding(t *testing.T) {
	tests := []struct {
		name     string
		endpoint DataEndpoint
		prefix   []byte
		want     []byte
	}{
		{
			name:     "BAM DMUX interface",
			endpoint: DataEndpoint{Type: DataEndpointBAMDMUX, InterfaceID: 1},
			want:     []byte{0x05, 0, 0, 0, 0x01, 0, 0, 0},
		},
		{
			name:     "HSUSB interface with prefix",
			endpoint: DataEndpoint{Type: DataEndpointHSUSB, InterfaceID: 4},
			prefix:   []byte{0xAA, 0xBB},
			want:     []byte{0xAA, 0xBB, 0x02, 0, 0, 0, 0x04, 0, 0, 0},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.endpoint.AppendBinary(bytes.Clone(tt.prefix))
			if err != nil {
				t.Fatalf("AppendBinary() error = %v", err)
			}
			if !bytes.Equal(got, tt.want) {
				t.Fatalf("AppendBinary() = % X, want % X", got, tt.want)
			}

			marshaled, err := tt.endpoint.MarshalBinary()
			if err != nil {
				t.Fatalf("MarshalBinary() error = %v", err)
			}
			wantMarshaled := tt.want[len(tt.prefix):]
			if !bytes.Equal(marshaled, wantMarshaled) {
				t.Fatalf("MarshalBinary() = % X, want % X", marshaled, wantMarshaled)
			}
		})
	}
}

func TestWDSDataEndpointCompatibilityAliases(t *testing.T) {
	tests := []struct {
		name     string
		endpoint WDSDataEndpoint
		wantType WDSDataEndpointType
	}{
		{
			name:     "BAM DMUX",
			endpoint: WDSDataEndpoint{Type: WDSDataEndpointBAMDMUX, InterfaceID: 1},
			wantType: WDSDataEndpointBAMDMUX,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.endpoint.Type != tt.wantType {
				t.Fatalf("Type = %d, want %d", tt.endpoint.Type, tt.wantType)
			}
		})
	}
}
