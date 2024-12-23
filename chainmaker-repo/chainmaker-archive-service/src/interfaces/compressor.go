/*
Copyright (C) BABEC. All rights reserved.


SPDX-License-Identifier: Apache-2.0
*/

// Package interfaces define file interface
package interfaces

// Compressor 压缩管理
type Compressor interface {
	// CompressFile 将fileDir目录下的fileName压缩到compressDir目录下
	CompressFile(fileDir string, fileName string, compressDir string) error
	// 将compressDir目录下的compressFileName解压到unCompressDir目录下
	DeCompressFile(compressDir string, compressFileName string, deCompressDir string) (string, error)
}
