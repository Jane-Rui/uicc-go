package qrtr

import (
	"bytes"
	"encoding/binary"
	"fmt"

	"github.com/damonto/uicc-go/qualcomm"
	"github.com/damonto/uicc-go/qualcomm/tlv"
)

type Response struct {
	TransactionID uint16
	MessageID     qualcomm.MessageID
	MessageType   qualcomm.MessageType
	MessageLength uint16
	TLVs          tlv.TLVs
}

func (r Response) Qualcomm() qualcomm.Response {
	return qualcomm.Response{
		Service:       qualcomm.ServiceUIM,
		TransactionID: r.TransactionID,
		MessageID:     r.MessageID,
		TLVs:          r.TLVs,
	}
}

func (r *Response) UnmarshalBinary(data []byte) error {
	*r = Response{}
	const headerLen = 7
	if len(data) < headerLen {
		return fmt.Errorf("parsing QRTR response: data too short: got %d bytes", len(data))
	}

	reader := bytes.NewReader(data)
	var header Header
	if err := binary.Read(reader, binary.LittleEndian, &header); err != nil {
		return fmt.Errorf("parsing QRTR response: read QMI header: %w", err)
	}
	r.MessageType = header.MessageType
	r.TransactionID = header.TransactionID
	r.MessageID = header.MessageID
	r.MessageLength = header.MessageLength

	if r.MessageType != qualcomm.MessageTypeResponse {
		return fmt.Errorf("parsing QRTR response: unexpected message type 0x%02X", r.MessageType)
	}
	if got, want := reader.Len(), int(r.MessageLength); got != want {
		return fmt.Errorf("parsing QRTR response: QMI TLV length mismatch: got %d bytes, header declares %d", got, want)
	}
	if r.MessageLength > 0 {
		if err := r.TLVs.UnmarshalBinary(data[headerLen : headerLen+int(r.MessageLength)]); err != nil {
			return err
		}
	}
	return nil
}
