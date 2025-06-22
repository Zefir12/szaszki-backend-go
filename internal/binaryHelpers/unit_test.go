package bh

import (
	"math/rand"
	"reflect"
	"testing"
	"time"
)

func randomValue(ft FieldType) any {
	switch ft {
	case Int8:
		return int8(rand.Intn(1<<8) - 1<<7)
	case Uint8:
		return uint8(rand.Intn(1 << 8))
	case Int16:
		return int16(rand.Intn(1<<16) - 1<<15)
	case Uint16:
		return uint16(rand.Intn(1 << 16))
	case Int32:
		return int32(rand.Int31())
	case Uint32:
		return uint32(rand.Uint32())
	case Uint64:
		return rand.Uint64()
	default:
		return nil
	}
}

func TestPackUnpackFormatRandom(t *testing.T) {
	rand.Seed(time.Now().UnixNano())

	// Define some format sets to test
	formats := [][]FieldType{
		{Uint16, Int8, Uint64},
		{Int8, Uint8, Int16, Uint32},
		{Uint64, Uint64, Uint64},
		{Int32, Uint16},
	}

	for _, format := range formats {
		for i := 0; i < 100; i++ { // 100 random tests per format
			values := make([]any, len(format))
			for idx, ft := range format {
				values[idx] = randomValue(ft)
			}

			packed, err := Pack(format, values)
			if err != nil {
				t.Fatalf("Pack failed for format %v values %v: %v", format, values, err)
			}

			unpacked, err := Unpack(packed, format)
			if err != nil {
				t.Fatalf("Unpack failed for format %v values %v: %v", format, values, err)
			}

			if !reflect.DeepEqual(values, unpacked) {
				t.Errorf("Mismatch after unpack\nFormat: %v\nOriginal: %v\nUnpacked: %v", format, values, unpacked)
			}
		}
	}
}
