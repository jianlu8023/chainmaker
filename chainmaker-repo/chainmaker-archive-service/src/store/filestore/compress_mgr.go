/*
Copyright (C) BABEC. All rights reserved.


SPDX-License-Identifier: Apache-2.0
*/

// Package filestore define file operation
package filestore

var (
	// DecompressFileRetainSeconds 默认解压缩文件保留10小时
	DecompressFileRetainSeconds = 10 * 3600
)

const (
	compressedHeightKey = "compressedHeightKey"
)
