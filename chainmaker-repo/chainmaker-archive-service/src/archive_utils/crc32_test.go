/*
Copyright (C) BABEC. All rights reserved.


SPDX-License-Identifier: Apache-2.0
*/

// Package archive_utils define common utils
package archive_utils

import (
	"encoding/binary"
	"testing"
)

func appendUvarint(dst []byte, x uint64) []byte {
	var buf [10]byte
	n := binary.PutUvarint(buf[:], x)
	dst = append(dst, buf[:n]...)
	return dst
}

func TestCRC_Binary(t *testing.T) {
	buf := []byte("string")
	checksum := NewCRC(buf).Value()

	// write checksum
	data := make([]byte, 0)
	data = append(data, []byte("0000")...)
	binary.LittleEndian.PutUint32(data[len(data)-4:], checksum)
	// write data_size
	data = appendUvarint(data, uint64(len(buf)))
	// write data
	data = append(data, buf...)

	if checksum != binary.LittleEndian.Uint32(data[:4]) {
		t.Fatalf("unequal checksum, expected:%d, actual:%d", checksum, binary.LittleEndian.Uint32(data[:4]))
	}
}
