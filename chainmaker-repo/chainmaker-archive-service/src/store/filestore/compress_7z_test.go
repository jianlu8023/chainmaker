/*
Copyright (C) BABEC. All rights reserved.


SPDX-License-Identifier: Apache-2.0
*/

// Package filestore define file operation
package filestore

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCompress(t *testing.T) {
	compressor := Compress7z{suffix: "7z", maxTime: 2000}
	compressErr := compressor.CompressFile("./", "./compress_7z.go", "./")
	assert.Nil(t, compressErr)
	// do clean job
	os.Remove("./compress_7z.go.7z")
}

func TestDeCompressFile(t *testing.T) {
	compressor := Compress7z{suffix: "7z", maxTime: 2000}
	compresserErr := compressor.CompressFile("./", "./compress_7z.go", "./")
	assert.Nil(t, compresserErr)
	dirErr := os.Mkdir("./decompress", os.ModePerm)
	assert.Nil(t, dirErr)
	decompressFile, decompressErr := compressor.DeCompressFile("./",
		"./compress_7z.go.7z", "./decompress")
	assert.Nil(t, decompressErr)
	// do clean job
	isSame, compareError := CompareFile("./compress_7z.go", decompressFile)
	assert.Nil(t, compareError)
	assert.True(t, isSame)
	removeDecompressErr := os.RemoveAll("./decompress")
	assert.Nil(t, removeDecompressErr)
	removeCompressErr := os.Remove("./compress_7z.go.7z")
	assert.Nil(t, removeCompressErr)
}
