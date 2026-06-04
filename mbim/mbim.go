package mbim

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"time"
)

const (
	slotPollInterval    = 500 * time.Millisecond
	slotReadyTimeout    = 5 * time.Second
	defaultCloseTimeout = 5 * time.Second
)

var masterFilePath = []byte{0x3F, 0x00}

type Reader struct {
	conn               Conn
	slot               uint32
	txn                atomic.Uint32
	proxy              bool
	maxControlTransfer int

	mu     sync.Mutex
	closed bool
}

const (
	FileStructureTransparent = 0x41
	FileStructureLinearFixed = 0x42
)

type Application struct {
	AID   []byte
	Label string
}

type FileRef struct {
	AID  []byte
	Path []byte
}

type FileAttributes struct {
	FileStructure byte
	FileType      byte
	RecordSize    uint16
	RecordCount   uint16
	FileSize      uint16
}

type TransparentRead struct {
	File   FileRef
	Offset uint16
	Length uint16
}

type RecordRead struct {
	File   FileRef
	Record uint16
}

func (r *Reader) Close() error {
	ctx, cancel := context.WithTimeout(context.Background(), defaultCloseTimeout)
	defer cancel()
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.closed {
		return nil
	}
	r.closed = true

	request := CloseRequest{TransactionID: r.nextTransactionID()}
	err := request.Request().Transmit(ctx, r.conn)
	return errors.Join(err, r.conn.Close())
}

func (r *Reader) nextTransactionID() uint32 {
	return r.txn.Add(1)
}
