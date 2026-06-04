package qmi

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/damonto/uicc-go/qualcomm"
)

type QMUXHeader struct {
	IfType       uint8
	Length       uint16
	ControlFlags uint8
	ServiceType  qualcomm.ServiceType
	ClientID     uint8
}

type Header[T uint8 | uint16] struct {
	MessageType   qualcomm.MessageType
	TransactionID T
	MessageID     qualcomm.MessageID
	MessageLength uint16
}

type Request struct {
	qualcomm.Request
}

func (r Request) MarshalBinary() ([]byte, error) {
	return marshalRequest(r.Request)
}

type Transport struct {
	conn Conn
	mu   sync.Mutex
}

func New(conn Conn) *Transport {
	return &Transport{conn: conn}
}

func (t *Transport) Close() error {
	return t.conn.Close()
}

func (t *Transport) Do(ctx context.Context, req qualcomm.Request) (qualcomm.Response, error) {
	t.mu.Lock()
	defer t.mu.Unlock()

	packet, err := (Request{Request: req}).MarshalBinary()
	if err != nil {
		return qualcomm.Response{}, err
	}

	deadline, hasDeadline := qualcomm.RequestDeadline(ctx, req.Timeout)
	if hasDeadline {
		if err := t.conn.SetWriteDeadline(deadline); err != nil {
			return qualcomm.Response{}, fmt.Errorf("setting QMI write deadline: %w", err)
		}
		defer func() { _ = t.conn.SetWriteDeadline(time.Time{}) }()
	}
	if err := writeFull(t.conn, packet); err != nil {
		return qualcomm.Response{}, fmt.Errorf("writing QMI request: %w", err)
	}

	if hasDeadline {
		if err := t.conn.SetReadDeadline(deadline); err != nil {
			return qualcomm.Response{}, fmt.Errorf("setting QMI read deadline: %w", err)
		}
		defer func() { _ = t.conn.SetReadDeadline(time.Time{}) }()
	}

	for {
		frame, err := ReadFrame(t.conn)
		if err != nil {
			if ctx.Err() != nil {
				return qualcomm.Response{}, ctx.Err()
			}
			return qualcomm.Response{}, fmt.Errorf("reading QMI response: %w", err)
		}

		var wire Response
		if err := wire.UnmarshalBinary(frame); err != nil {
			return qualcomm.Response{}, err
		}
		resp := wire.Qualcomm()
		if resp.Service != req.Service || resp.TransactionID != req.TransactionID || resp.MessageID != req.MessageID {
			continue
		}
		if req.Service != qualcomm.ServiceControl && resp.ClientID != req.ClientID {
			continue
		}
		return resp, nil
	}
}

func MarshalRequest(req qualcomm.Request) ([]byte, error) {
	return (Request{Request: req}).MarshalBinary()
}

func marshalRequest(req qualcomm.Request) ([]byte, error) {
	payload, err := req.TLVs.MarshalBinary()
	if err != nil {
		return nil, fmt.Errorf("marshal QMI TLVs: %w", err)
	}
	maxPayloadLength := qualcomm.MaxQMUXServiceTLVLength
	if req.Service == qualcomm.ServiceControl {
		maxPayloadLength = qualcomm.MaxQMUXControlTLVLength
	}
	if len(payload) > maxPayloadLength {
		return nil, fmt.Errorf("QMI message TLVs length %d exceeds limit %d", len(payload), maxPayloadLength)
	}

	sdu := new(bytes.Buffer)
	if req.Service == qualcomm.ServiceControl {
		if err := binary.Write(sdu, binary.LittleEndian, Header[uint8]{
			MessageType:   qualcomm.MessageTypeRequest,
			TransactionID: uint8(req.TransactionID),
			MessageID:     req.MessageID,
			MessageLength: uint16(len(payload)),
		}); err != nil {
			return nil, fmt.Errorf("write control QMI header: %w", err)
		}
	} else {
		if err := binary.Write(sdu, binary.LittleEndian, Header[uint16]{
			MessageType:   qualcomm.MessageTypeRequest,
			TransactionID: req.TransactionID,
			MessageID:     req.MessageID,
			MessageLength: uint16(len(payload)),
		}); err != nil {
			return nil, fmt.Errorf("write service QMI header: %w", err)
		}
	}
	if _, err := sdu.Write(payload); err != nil {
		return nil, fmt.Errorf("write QMI payload: %w", err)
	}

	out := new(bytes.Buffer)
	if err := binary.Write(out, binary.LittleEndian, QMUXHeader{
		IfType:       qualcomm.QMUXIfType,
		Length:       uint16(sdu.Len() + 5),
		ControlFlags: qualcomm.QMUXControlFlagRequest,
		ServiceType:  req.Service,
		ClientID:     req.ClientID,
	}); err != nil {
		return nil, fmt.Errorf("write QMUX header: %w", err)
	}
	if _, err := out.Write(sdu.Bytes()); err != nil {
		return nil, fmt.Errorf("write QMUX payload: %w", err)
	}
	return out.Bytes(), nil
}

func writeFull(w io.Writer, p []byte) error {
	for len(p) > 0 {
		n, err := w.Write(p)
		if err != nil {
			return err
		}
		if n <= 0 {
			return io.ErrShortWrite
		}
		p = p[n:]
	}
	return nil
}
