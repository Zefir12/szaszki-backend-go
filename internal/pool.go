package internal

import "sync"

var pool8 = sync.Pool{New: func() any {
	buf := make([]byte, 8)
	return &buf
}}
var pool512 = sync.Pool{New: func() any {
	buf := make([]byte, 512)
	return &buf
}}
var pool2K = sync.Pool{New: func() any {
	buf := make([]byte, 2048)
	return &buf
}}
var pool64K = sync.Pool{New: func() any {
	buf := make([]byte, 65536)
	return &buf
}}

// Select pool based on requested buffer size
func GetBufferForSize(size int) *[]byte { //sprawdzenie jak działa bez pointerów moze byc ciekawe
	switch {
	case size <= 8:
		return pool8.Get().(*[]byte)
	case size <= 512:
		return pool512.Get().(*[]byte)
	case size <= 2048:
		return pool2K.Get().(*[]byte)
	default:
		return pool64K.Get().(*[]byte)
	}
}

// Return buffer to appropriate pool
func PutBuffer(buf *[]byte) {
	switch cap(*buf) {
	case 8:
		pool8.Put(buf)
	case 512:
		pool512.Put(buf)
	case 2048:
		pool2K.Put(buf)
	case 65536:
		pool64K.Put(buf)
	}
}
