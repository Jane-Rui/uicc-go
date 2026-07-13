package mbim

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"time"
)

const maxQueuedIndications = 32

var (
	errClientClosed    = errors.New("MBIM client is closed")
	errReceiverStopped = errors.New("MBIM receiver stopped")
)

type responseWaiter struct {
	messageType   MessageType
	serviceID     [16]byte
	commandID     uint32
	expectCommand bool
	ch            chan responseResult
}

type responseResult struct {
	data []byte
	err  error
}

type indicationKey struct {
	serviceID [16]byte
	commandID uint32
}

func (c *Client) startReceiver() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.ensureReceiverLocked(false)
}

func (c *Client) ensureReceiverLocked(allowClosing bool) error {
	switch {
	case c.closed:
		return errClientClosed
	case c.closing && !allowClosing:
		return errClientClosed
	case c.receiverErr != nil:
		return c.receiverErr
	case c.receiverStarted:
		return nil
	case c.conn == nil:
		return errors.New("MBIM client connection is nil")
	}

	if c.pending == nil {
		c.pending = make(map[uint32]*responseWaiter)
	}
	if c.subs == nil {
		c.subs = make(map[indicationKey]map[chan Indication]struct{})
	}
	if c.waiters == nil {
		c.waiters = make(map[indicationKey][]chan Indication)
	}
	if c.indications == nil {
		c.indications = make(map[indicationKey][]Indication)
	}
	c.receiverStarted = true
	go c.receive()
	return nil
}

func (c *Client) beginClose() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.closed || c.closing {
		return false
	}
	c.closing = true
	return true
}

func (c *Client) finishClose() {
	c.mu.Lock()
	c.closed = true
	c.mu.Unlock()
}

func (c *Client) transmit(ctx context.Context, request *Request) error {
	return c.transmitRequest(ctx, request, false)
}

func (c *Client) transmitClosing(ctx context.Context, request *Request) error {
	return c.transmitRequest(ctx, request, true)
}

func (c *Client) transmitRequest(ctx context.Context, request *Request, allowClosing bool) error {
	ctx, cancel := requestContext(ctx, request.timeout())
	defer cancel()

	results, unregister, err := c.registerResponse(request, allowClosing)
	if err != nil {
		return err
	}
	defer unregister()

	c.writeMu.Lock()
	_, err = request.writeConn(ctx, c.conn)
	c.writeMu.Unlock()
	if err != nil {
		return err
	}

	select {
	case result := <-results:
		if result.err != nil {
			return result.err
		}
		return request.unmarshalResponse(result.data)
	case <-ctx.Done():
		return ctx.Err()
	}
}

func requestContext(ctx context.Context, timeout time.Duration) (context.Context, context.CancelFunc) {
	deadline, _ := requestDeadline(ctx, timeout)
	return context.WithDeadline(ctx, deadline)
}

func (c *Client) registerResponse(request *Request, allowClosing bool) (<-chan responseResult, func(), error) {
	messageType, ok := responseMessageType(request.MessageType)
	if !ok {
		return nil, nil, fmt.Errorf("registering MBIM response: unsupported request message type %#x", request.MessageType)
	}
	serviceID, commandID, expectCommand := request.expectedCommand()

	c.mu.Lock()
	defer c.mu.Unlock()
	if err := c.ensureReceiverLocked(allowClosing); err != nil {
		return nil, nil, err
	}
	if _, ok := c.pending[request.TransactionID]; ok {
		return nil, nil, fmt.Errorf("registering MBIM response: transaction ID %d is already pending", request.TransactionID)
	}

	ch := make(chan responseResult, 1)
	c.pending[request.TransactionID] = &responseWaiter{
		messageType:   messageType,
		serviceID:     serviceID,
		commandID:     commandID,
		expectCommand: expectCommand,
		ch:            ch,
	}
	return ch, func() { c.unregisterResponse(request.TransactionID, ch) }, nil
}

func (c *Client) unregisterResponse(transactionID uint32, ch <-chan responseResult) {
	c.mu.Lock()
	defer c.mu.Unlock()
	waiter, ok := c.pending[transactionID]
	if ok && waiter.ch == ch {
		delete(c.pending, transactionID)
	}
}

func (c *Client) receive() {
	var collector *fragmentCollector
	for {
		buf, err := readFrame(c.conn)
		if err != nil {
			if timeoutError(err) {
				continue
			}
			c.stopReceiver(fmt.Errorf("receiving MBIM message: %w", err))
			return
		}

		messageType := MessageType(binary.LittleEndian.Uint32(buf[:4]))
		if isFragmentMessage(messageType) {
			complete, err := collectFrame(&collector, buf)
			if err != nil {
				c.stopReceiver(err)
				return
			}
			if complete == nil {
				continue
			}
			buf = complete
			messageType = MessageType(binary.LittleEndian.Uint32(buf[:4]))
		}

		switch messageType {
		case MessageTypeOpenDone, MessageTypeCloseDone, MessageTypeCommandDone, MessageTypeFunctionError:
			c.deliverResponse(messageType, buf)
		case MessageTypeIndicateStatus:
			var indication Indication
			if err := indication.UnmarshalBinary(buf); err != nil {
				c.stopReceiver(err)
				return
			}
			c.publishIndication(indication)
		}
	}
}

func collectFrame(collector **fragmentCollector, buf []byte) ([]byte, error) {
	if *collector != nil {
		if err := (*collector).add(buf); err != nil {
			return nil, err
		}
		if !(*collector).complete() {
			return nil, nil
		}
		complete, err := (*collector).MarshalBinary()
		*collector = nil
		return complete, err
	}

	if len(buf) < 20 || binary.LittleEndian.Uint32(buf[12:16]) <= 1 {
		return buf, nil
	}
	next, err := newFragmentCollector(buf)
	if err != nil {
		return nil, err
	}
	if next.complete() {
		return next.MarshalBinary()
	}
	*collector = next
	return nil, nil
}

func (c *Client) deliverResponse(messageType MessageType, data []byte) {
	transactionID := binary.LittleEndian.Uint32(data[8:12])

	c.mu.Lock()
	waiter := c.pending[transactionID]
	if waiter == nil || !waiter.matches(messageType, data) {
		c.mu.Unlock()
		return
	}
	delete(c.pending, transactionID)
	c.mu.Unlock()

	waiter.ch <- responseResult{data: data}
}

func (w *responseWaiter) matches(messageType MessageType, data []byte) bool {
	if messageType != w.messageType && messageType != MessageTypeFunctionError {
		return false
	}
	if messageType != MessageTypeCommandDone || !w.expectCommand {
		return true
	}

	var header commandDoneHeader
	if err := header.UnmarshalBinary(data); err != nil {
		return true
	}
	return header.ServiceID == w.serviceID && header.CommandID == w.commandID
}

func (c *Client) stopReceiver(err error) {
	c.mu.Lock()
	if c.receiverErr == nil {
		c.receiverErr = err
	}
	pending := c.pending
	c.pending = make(map[uint32]*responseWaiter)
	subs := c.subs
	c.subs = make(map[indicationKey]map[chan Indication]struct{})
	waiters := c.waiters
	c.waiters = make(map[indicationKey][]chan Indication)
	c.receiverStarted = false
	c.mu.Unlock()

	for _, waiter := range pending {
		waiter.ch <- responseResult{err: err}
	}
	for _, set := range subs {
		for ch := range set {
			close(ch)
		}
	}
	for _, set := range waiters {
		for _, ch := range set {
			close(ch)
		}
	}
}

func (c *Client) nextIndication(ctx context.Context, key indicationKey) (Indication, error) {
	c.mu.Lock()
	if c.closed || c.closing {
		c.mu.Unlock()
		return Indication{}, errClientClosed
	}

	queue := c.indications[key]
	if len(queue) > 0 {
		indication := cloneIndication(queue[0])
		if len(queue) == 1 {
			delete(c.indications, key)
		} else {
			c.indications[key] = queue[1:]
		}
		c.mu.Unlock()
		return indication, nil
	}
	if c.receiverErr != nil {
		err := c.receiverErr
		c.mu.Unlock()
		return Indication{}, err
	}
	if err := c.ensureReceiverLocked(false); err != nil {
		c.mu.Unlock()
		return Indication{}, err
	}

	ch := make(chan Indication, 1)
	c.waiters[key] = append(c.waiters[key], ch)
	c.mu.Unlock()

	select {
	case <-ctx.Done():
		if indication, ok := c.cancelIndicationWaiter(key, ch); ok {
			return indication, nil
		}
		return Indication{}, ctx.Err()
	case indication, ok := <-ch:
		if !ok {
			return Indication{}, errReceiverStopped
		}
		return indication, nil
	}
}

func (c *Client) subscribeIndication(key indicationKey) (<-chan Indication, func(), error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.closed || c.closing {
		return nil, nil, errClientClosed
	}

	queued := c.indications[key]
	if c.receiverErr != nil {
		if len(queued) == 0 {
			return nil, nil, c.receiverErr
		}
		ch := make(chan Indication, len(queued))
		for _, indication := range queued {
			ch <- cloneIndication(indication)
		}
		close(ch)
		delete(c.indications, key)
		return ch, func() {}, nil
	}

	if err := c.ensureReceiverLocked(false); err != nil {
		return nil, nil, err
	}

	ch := make(chan Indication, maxQueuedIndications)
	if c.subs[key] == nil {
		c.subs[key] = make(map[chan Indication]struct{})
	}
	c.subs[key][ch] = struct{}{}
	for _, indication := range queued {
		ch <- cloneIndication(indication)
	}
	delete(c.indications, key)
	return ch, func() { c.unsubscribeIndication(key, ch) }, nil
}

func (c *Client) unsubscribeIndication(key indicationKey, ch chan Indication) {
	c.mu.Lock()
	defer c.mu.Unlock()
	subs := c.subs[key]
	if subs == nil {
		return
	}
	delete(subs, ch)
	if len(subs) == 0 {
		delete(c.subs, key)
	}
}

func (c *Client) cancelIndicationWaiter(key indicationKey, ch chan Indication) (Indication, bool) {
	c.mu.Lock()
	waiters := c.waiters[key]
	for i, waiter := range waiters {
		if waiter != ch {
			continue
		}
		waiters = append(waiters[:i], waiters[i+1:]...)
		if len(waiters) == 0 {
			delete(c.waiters, key)
		} else {
			c.waiters[key] = waiters
		}
		c.mu.Unlock()
		return Indication{}, false
	}
	c.mu.Unlock()

	select {
	case indication, ok := <-ch:
		if ok {
			return indication, true
		}
	default:
	}
	return Indication{}, false
}

func (c *Client) publishIndication(indication Indication) {
	key := indicationKey{serviceID: indication.ServiceID, commandID: indication.CommandID}

	c.mu.Lock()
	subs := c.subs[key]
	waiters := c.waiters[key]
	var waiter chan Indication
	if len(waiters) > 0 {
		waiter = waiters[0]
		if len(waiters) == 1 {
			delete(c.waiters, key)
		} else {
			c.waiters[key] = waiters[1:]
		}
	}
	if len(subs) == 0 && waiter == nil {
		c.queueIndicationLocked(key, indication)
		c.mu.Unlock()
		return
	}
	for ch := range subs {
		deliverIndication(ch, cloneIndication(indication))
	}
	if waiter != nil {
		waiter <- cloneIndication(indication)
	}
	c.mu.Unlock()
}

func (c *Client) queueIndicationLocked(key indicationKey, indication Indication) {
	queue := append(c.indications[key], cloneIndication(indication))
	if len(queue) > maxQueuedIndications {
		queue = queue[len(queue)-maxQueuedIndications:]
	}
	c.indications[key] = queue
}

func deliverIndication(ch chan Indication, indication Indication) {
	select {
	case ch <- indication:
		return
	default:
	}
	select {
	case <-ch:
	default:
	}
	select {
	case ch <- indication:
	default:
	}
}

func cloneIndication(indication Indication) Indication {
	indication.InformationBuffer = append([]byte(nil), indication.InformationBuffer...)
	return indication
}
