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

func TestCompressGzipCompressFile(t *testing.T) {
	compressor := CompressGzip{suffix: "gzip"}
	compressErr := compressor.CompressFile("./", "./compress_gzip.go", "./")
	assert.Nil(t, compressErr)
	// do clean job
	removeErr := os.Remove("./compress_gzip.go.gzip")
	assert.Nil(t, removeErr)
}

func TestCompressGzipDecompressFile(t *testing.T) {
	compressor := CompressGzip{suffix: "gzip"}
	compressErr := compressor.CompressFile("./",
		"./compress_gzip.go", "./")
	assert.Nil(t, compressErr)
	pathErr := os.Mkdir("./decompress_gzip", os.ModePerm)
	assert.Nil(t, pathErr)
	decompressPath, decompressErr := compressor.DeCompressFile("./",
		"./compress_gzip.go.gzip", "./decompress_gzip")
	assert.Nil(t, decompressErr)
	isSame, compareErr := CompareFile("./compress_gzip.go", decompressPath)
	assert.Nil(t, compareErr)
	assert.True(t, isSame)
	// do clean job
	removeDecompressErr := os.RemoveAll("./decompress_gzip")
	assert.Nil(t, removeDecompressErr)
	removeGzipErr := os.Remove("./compress_gzip.go.gzip")
	assert.Nil(t, removeGzipErr)
}
