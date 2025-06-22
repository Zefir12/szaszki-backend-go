package bh

import (
	"encoding/binary"
	"errors"
	"fmt"
)

type FieldType int

const (
	Int8 FieldType = iota
	Uint8
	Int16
	Uint16
	Int32
	Uint32
	Int64
	Uint64
)

func sizeOf(f FieldType) int {
	switch f {
	case Int8, Uint8:
		return 1
	case Int16, Uint16:
		return 2
	case Int32, Uint32:
		return 4
	case Uint64, Int64:
		return 8
	default:
		return 0
	}
}

func estimateSize(format []FieldType) int {
	total := 0
	for _, f := range format {
		total += sizeOf(f)
	}
	return total
}

func Pack(format []FieldType, values []any) ([]byte, error) {
	if len(format) != len(values) {
		return nil, fmt.Errorf("format and values length mismatch")
	}

	buf := make([]byte, estimateSize(format))
	offset := 0

	for i, f := range format {
		switch f {
		case Int8:
			buf[offset] = byte(values[i].(int8))
			offset++
		case Uint8:
			buf[offset] = values[i].(uint8)
			offset++
		case Int16:
			binary.BigEndian.PutUint16(buf[offset:], uint16(values[i].(int16)))
			offset += 2
		case Uint16:
			binary.BigEndian.PutUint16(buf[offset:], values[i].(uint16))
			offset += 2
		case Int32:
			binary.BigEndian.PutUint32(buf[offset:], uint32(values[i].(int32)))
			offset += 4
		case Uint32:
			binary.BigEndian.PutUint32(buf[offset:], values[i].(uint32))
			offset += 4
		case Int64:
			binary.BigEndian.PutUint64(buf[offset:], uint64(values[i].(int64)))
			offset += 8
		case Uint64:
			binary.BigEndian.PutUint64(buf[offset:], values[i].(uint64))
			offset += 8
		default:
			return nil, fmt.Errorf("unsupported field type: %v", f)
		}
	}

	return buf, nil
}

func Unpack(data []byte, format []FieldType) ([]any, error) {
	result := make([]any, 0, len(format))
	offset := 0

	for _, f := range format {
		if offset+sizeOf(f) > len(data) {
			return nil, errors.New("not enough data to unpack")
		}

		switch f {
		case Int8:
			result = append(result, int8(data[offset]))
			offset++
		case Uint8:
			result = append(result, data[offset])
			offset++
		case Int16:
			v := int16(binary.BigEndian.Uint16(data[offset:]))
			result = append(result, v)
			offset += 2
		case Uint16:
			v := binary.BigEndian.Uint16(data[offset:])
			result = append(result, v)
			offset += 2
		case Int32:
			v := int32(binary.BigEndian.Uint32(data[offset:]))
			result = append(result, v)
			offset += 4
		case Uint32:
			v := binary.BigEndian.Uint32(data[offset:])
			result = append(result, v)
			offset += 4
		case Int64:
			v := int64(binary.BigEndian.Uint64(data[offset:]))
			result = append(result, v)
			offset += 8
		case Uint64:
			v := binary.BigEndian.Uint64(data[offset:])
			result = append(result, v)
			offset += 8
		default:
			return nil, fmt.Errorf("unsupported field type: %v", f)
		}
	}

	return result, nil
}
