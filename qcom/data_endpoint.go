package qcom

import "encoding/binary"

// AppendBinary appends the QMI data_ep_id_type_v01 representation to dst.
func (e DataEndpoint) AppendBinary(dst []byte) ([]byte, error) {
	dst = binary.LittleEndian.AppendUint32(dst, uint32(e.Type))
	dst = binary.LittleEndian.AppendUint32(dst, e.InterfaceID)
	return dst, nil
}

// MarshalBinary returns the QMI data_ep_id_type_v01 representation.
func (e DataEndpoint) MarshalBinary() ([]byte, error) {
	return e.AppendBinary(nil)
}
