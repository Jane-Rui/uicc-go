package uim

import (
	"context"
	"errors"
	"fmt"

	"github.com/damonto/uicc-go/qualcomm"
	"github.com/damonto/uicc-go/qualcomm/tlv"
)

func (r *Reader) Reset(ctx context.Context) error {
	resp, err := r.request(ctx, qualcomm.QMIServiceUIM, r.clientID, qualcomm.QMIUIMReset, nil, DefaultRequestTimeout)
	if err != nil {
		return fmt.Errorf("resetting QMI UIM service: %w", err)
	}
	if err := resultOK(resp); err != nil {
		return fmt.Errorf("resetting QMI UIM service: %w", err)
	}
	return nil
}

func (r *Reader) PowerOffSIM(ctx context.Context, slot uint8) error {
	if slot == 0 {
		return errors.New("powering off QMI UIM SIM: slot is zero")
	}

	resp, err := r.request(ctx, qualcomm.QMIServiceUIM, r.clientID, qualcomm.QMIUIMPowerOffSIM, tlv.TLVs{
		tlv.Uint(0x01, slot),
	}, DefaultRequestTimeout)
	if err != nil {
		return fmt.Errorf("powering off QMI UIM SIM slot %d: %w", slot, err)
	}
	if err := resultOK(resp); err != nil {
		return fmt.Errorf("powering off QMI UIM SIM slot %d: %w", slot, err)
	}
	return nil
}

func (r *Reader) PowerOnSIM(ctx context.Context, req PowerOnSIMRequest) error {
	if req.Slot == 0 {
		return errors.New("powering on QMI UIM SIM: slot is zero")
	}

	tlvs := tlv.TLVs{tlv.Uint(0x01, req.Slot)}
	if req.IgnoreHotSwapSwitch {
		tlvs = append(tlvs, tlv.Uint(0x10, uint8(1)))
	}

	resp, err := r.request(ctx, qualcomm.QMIServiceUIM, r.clientID, qualcomm.QMIUIMPowerOnSIM, tlvs, DefaultRequestTimeout)
	if err != nil {
		return fmt.Errorf("powering on QMI UIM SIM slot %d: %w", req.Slot, err)
	}
	if err := resultOK(resp); err != nil {
		return fmt.Errorf("powering on QMI UIM SIM slot %d: %w", req.Slot, err)
	}
	return nil
}
