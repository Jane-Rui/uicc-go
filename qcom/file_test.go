package qcom

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/damonto/wwan-go/qcom/tlv"
)

func TestWriteRecord(t *testing.T) {
	tests := []struct {
		name string
		req  RecordWrite
	}{
		{
			name: "USIM MSISDN record",
			req: RecordWrite{
				File: File{
					Session: SessionPrimaryGWProvisioning,
					Path:    []byte{0x3F, 0x00, 0x7F, 0xFF, 0x6F, 0x40},
				},
				Record: 1,
				Data:   []byte{0xFF, 0x07, 0x91, 0x68, 0x31, 0x08, 0x10, 0x83, 0x00, 0xF8, 0xFF, 0xFF},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			transport := &fakeTransport{
				t: t,
				calls: []transportCall{{
					check: func(req Request) {
						if req.Service != ServiceUIM || req.MessageID != MessageWriteRecord {
							t.Fatalf("request = service 0x%02X message 0x%04X", req.Service, req.MessageID)
						}
						assertTLV(t, req.TLVs, 0x01, []byte{byte(tt.req.File.Session), 0x00})
						assertTLV(t, req.TLVs, 0x02, []byte{0x40, 0x6F, 0x04, 0x00, 0x3F, 0xFF, 0x7F})

						value, ok := tlv.Value(req.TLVs, 0x03)
						if !ok {
							t.Fatal("write record TLV missing")
						}
						want := []byte{0x01, 0x00, byte(len(tt.req.Data)), 0x00}
						want = append(want, tt.req.Data...)
						if !bytes.Equal(value, want) {
							t.Fatalf("write record TLV = % X, want % X", value, want)
						}
					},
					resp: successResponse(MessageWriteRecord, tlv.Bytes(qmiTLVCardResult, []byte{0x90, 0x00})),
				}},
			}
			reader := &Client{transport: transport, slot: 1, clientIDs: map[ServiceType]uint8{ServiceUIM: 7}}

			if err := reader.WriteRecord(context.Background(), tt.req); err != nil {
				t.Fatalf("WriteRecord() error = %v", err)
			}
		})
	}
}

func TestWriteRecordValidation(t *testing.T) {
	tests := []struct {
		name    string
		req     RecordWrite
		wantErr string
	}{
		{
			name: "zero record",
			req: RecordWrite{
				File: File{Path: []byte{0x6F, 0x40}},
			},
			wantErr: "record number is zero",
		},
		{
			name: "content too long",
			req: RecordWrite{
				File:   File{Path: []byte{0x6F, 0x40}},
				Record: 1,
				Data:   bytes.Repeat([]byte{0xFF}, maxRecordContentLength+1),
			},
			wantErr: "exceeds QMI UIM limit",
		},
		{
			name: "invalid path",
			req: RecordWrite{
				File:   File{Path: []byte{0x6F}},
				Record: 1,
			},
			wantErr: "path length must be an even number of bytes",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader := &Client{transport: &fakeTransport{t: t}, slot: 1, clientIDs: map[ServiceType]uint8{ServiceUIM: 7}}
			err := reader.WriteRecord(context.Background(), tt.req)
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("WriteRecord() error = %v, want text %q", err, tt.wantErr)
			}
		})
	}
}

func TestWriteRecordRejectsResponseIndication(t *testing.T) {
	tests := []struct {
		name string
		tlvs tlv.TLVs
	}{
		{
			name: "indication token",
			tlvs: tlv.TLVs{tlv.Uint(0x11, uint32(7))},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader := &Client{
				transport: &fakeTransport{t: t, calls: []transportCall{{
					resp: successResponse(MessageWriteRecord, tt.tlvs...),
				}}},
				slot:      1,
				clientIDs: map[ServiceType]uint8{ServiceUIM: 7},
			}

			err := reader.WriteRecord(context.Background(), RecordWrite{
				File:   File{Path: []byte{0x6F, 0x40}},
				Record: 1,
			})
			if err == nil || !strings.Contains(err.Error(), "response indication is not supported") {
				t.Fatalf("WriteRecord() error = %v, want indication error", err)
			}
		})
	}
}
