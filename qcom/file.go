package qcom

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"slices"

	"github.com/damonto/wwan-go/qcom/tlv"
)

const maxRecordContentLength = 255

type RawFileAttributes struct {
	FileSize    uint16
	FileID      uint16
	FileType    QMIFileType
	RecordSize  uint16
	RecordCount uint16
	Raw         []byte
}

func (r *RawFileAttributes) UnmarshalBinary(data []byte) error {
	if len(data) < 9 {
		return errors.New("reading file attributes: attributes payload is truncated")
	}

	r.FileSize = binary.LittleEndian.Uint16(data[:2])
	r.FileID = binary.LittleEndian.Uint16(data[2:4])
	r.FileType = QMIFileType(data[4])
	r.RecordSize = binary.LittleEndian.Uint16(data[5:7])
	r.RecordCount = binary.LittleEndian.Uint16(data[7:9])

	if len(data) < 26 {
		return nil
	}

	rawLength := int(binary.LittleEndian.Uint16(data[24:26]))
	if len(data) < 26+rawLength {
		return errors.New("reading file attributes: raw data is truncated")
	}

	r.Raw = slices.Clone(data[26 : 26+rawLength])
	return nil
}

func (c *Client) FileAttributes(ctx context.Context, file File) (FileAttributes, error) {
	return c.GetFileAttributes(ctx, file)
}

func (c *Client) GetFileAttributes(ctx context.Context, file File) (FileAttributes, error) {
	response, err := c.fileAttributesResponse(ctx, file)
	if err != nil {
		return FileAttributes{}, err
	}
	return decodeFileAttributes(response)
}

func (c *Client) ReadTransparent(ctx context.Context, req TransparentRead) ([]byte, error) {
	length := req.Length
	if length == 0 {
		attrs, err := c.FileAttributes(ctx, req.File)
		if err != nil {
			return nil, err
		}
		if attrs.FileStructure != FileStructureTransparent {
			return nil, errors.New("reading transparent file: unexpected file structure")
		}
		if req.Offset > attrs.FileSize {
			return nil, errors.New("reading transparent file: offset exceeds file size")
		}
		length = attrs.FileSize - req.Offset
	}

	response, err := c.transparentResponse(ctx, req.File, req.Offset, length)
	if err != nil {
		return nil, err
	}

	value, ok := tlv.Value(response.TLVs, 0x11)
	if !ok {
		return nil, errors.New("reading transparent file: read result TLV missing")
	}
	return decodeLengthPrefixedBytes(value)
}

func (c *Client) ReadRecord(ctx context.Context, req RecordRead) ([]byte, error) {
	if req.Record == 0 {
		return nil, errors.New("reading record file: record number is zero")
	}

	length := req.Length
	if length == 0 {
		attrs, err := c.FileAttributes(ctx, req.File)
		if err != nil {
			return nil, err
		}
		if attrs.FileStructure != FileStructureLinearFixed {
			return nil, errors.New("reading record file: unexpected file structure")
		}
		length = attrs.RecordSize
	}

	response, err := c.recordResponse(ctx, req.File, req.Record, length)
	if err != nil {
		return nil, err
	}

	value, ok := tlv.Value(response.TLVs, 0x11)
	if !ok {
		return nil, errors.New("reading record file: read result TLV missing")
	}
	return decodeLengthPrefixedBytes(value)
}

func (c *Client) WriteRecord(ctx context.Context, req RecordWrite) error {
	if req.Record == 0 {
		return errors.New("writing record file: record number is zero")
	}
	if len(req.Data) > maxRecordContentLength {
		return fmt.Errorf("writing record file: content length %d exceeds QMI UIM limit %d", len(req.Data), maxRecordContentLength)
	}

	fileValue, err := putFileValue(req.File.Path)
	if err != nil {
		return fmt.Errorf("writing record file: %w", err)
	}

	recordValue := binary.LittleEndian.AppendUint16(nil, req.Record)
	recordValue = binary.LittleEndian.AppendUint16(recordValue, uint16(len(req.Data)))
	recordValue = append(recordValue, req.Data...)

	resp, err := c.request(ctx, MessageWriteRecord, tlv.TLVs{
		tlv.Bytes(0x01, putSessionValue(req.File.Session, req.File.AID)),
		tlv.Bytes(0x02, fileValue),
		tlv.Bytes(0x03, recordValue),
	})
	if err != nil {
		return fmt.Errorf("writing record file: %w", err)
	}
	if err := resultOK(resp); err != nil {
		return fmt.Errorf("writing record file: %w", err)
	}
	if _, ok := tlv.Value(resp.TLVs, 0x11); ok {
		return errors.New("writing record file: response indication is not supported")
	}
	if err := cardError(resp.TLVs); err != nil {
		return fmt.Errorf("writing record file: %w", err)
	}
	return nil
}

func (c *Client) transparentResponse(
	ctx context.Context,
	file File,
	offset uint16,
	length uint16,
) (Response, error) {
	fileValue, err := putFileValue(file.Path)
	if err != nil {
		return Response{}, err
	}

	info := joinBytes(
		binary.LittleEndian.AppendUint16(nil, offset),
		binary.LittleEndian.AppendUint16(nil, length),
	)
	resp, err := c.request(ctx, MessageReadTransparent, tlv.TLVs{
		tlv.Bytes(0x01, putSessionValue(file.Session, file.AID)),
		tlv.Bytes(0x02, fileValue),
		tlv.Bytes(0x03, info),
	})
	if err != nil {
		return Response{}, err
	}
	if err := ResultError(resp.TLVs); err != nil {
		if errors.Is(err, QMIErrorInsufficientResources) {
			if _, ok := tlv.Value(resp.TLVs, 0x15); ok {
				return Response{}, errors.New("reading transparent file: long response is not supported")
			}
		}
		return Response{}, err
	}
	if _, ok := tlv.Value(resp.TLVs, 0x12); ok {
		return Response{}, errors.New("reading transparent file: response indication is not supported")
	}
	if err := cardError(resp.TLVs); err != nil {
		return Response{}, err
	}
	return resp, nil
}

func (c *Client) recordResponse(
	ctx context.Context,
	file File,
	record uint16,
	length uint16,
) (Response, error) {
	fileValue, err := putFileValue(file.Path)
	if err != nil {
		return Response{}, err
	}

	recordValue := joinBytes(
		binary.LittleEndian.AppendUint16(nil, record),
		binary.LittleEndian.AppendUint16(nil, length),
	)
	resp, err := c.request(ctx, MessageReadRecord, tlv.TLVs{
		tlv.Bytes(0x01, putSessionValue(file.Session, file.AID)),
		tlv.Bytes(0x02, fileValue),
		tlv.Bytes(0x03, recordValue),
	})
	if err != nil {
		return Response{}, err
	}
	if err := ResultError(resp.TLVs); err != nil {
		return Response{}, err
	}
	if _, ok := tlv.Value(resp.TLVs, 0x13); ok {
		return Response{}, errors.New("reading record file: response indication is not supported")
	}
	if err := cardError(resp.TLVs); err != nil {
		return Response{}, err
	}
	return resp, nil
}

func (c *Client) fileAttributesResponse(
	ctx context.Context,
	file File,
) (Response, error) {
	fileValue, err := putFileValue(file.Path)
	if err != nil {
		return Response{}, err
	}

	resp, err := c.request(ctx, MessageGetFileAttributes, tlv.TLVs{
		tlv.Bytes(0x01, putSessionValue(file.Session, file.AID)),
		tlv.Bytes(0x02, fileValue),
	})
	if err != nil {
		return Response{}, err
	}
	if err := cardResultOK(resp); err != nil {
		return Response{}, err
	}
	return resp, nil
}

func decodeFileAttributes(resp Response) (FileAttributes, error) {
	value, ok := tlv.Value(resp.TLVs, 0x11)
	if !ok {
		return FileAttributes{}, errors.New("reading file attributes: attributes TLV missing")
	}

	var attrs RawFileAttributes
	if err := attrs.UnmarshalBinary(value); err != nil {
		return FileAttributes{}, err
	}

	return FileAttributes{
		FileSize:      attrs.FileSize,
		RecordSize:    attrs.RecordSize,
		RecordCount:   attrs.RecordCount,
		FileType:      fileTypeToSIMFileType(attrs.FileType),
		FileStructure: fileTypeToSIMFileStructure(attrs.FileType),
	}, nil
}

func fileTypeToSIMFileStructure(fileType QMIFileType) FileStructure {
	switch fileType {
	case QMIFileTypeTransparent:
		return FileStructureTransparent
	case QMIFileTypeLinearFixed:
		return FileStructureLinearFixed
	default:
		return 0
	}
}

func fileTypeToSIMFileType(fileType QMIFileType) FileType {
	switch fileType {
	case QMIFileTypeTransparent, QMIFileTypeCyclic, QMIFileTypeLinearFixed:
		return FileTypeWorkingEF
	case QMIFileTypeDedicated, QMIFileTypeMaster:
		return FileTypeDFOrADF
	default:
		return FileType(fileType)
	}
}
