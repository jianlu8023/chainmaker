/*
Copyright (C) BABEC. All rights reserved.


SPDX-License-Identifier: Apache-2.0
*/

// Package interfaces define file interface
package interfaces

import storePb "chainmaker.org/chainmaker/pb-go/v2/store"

// BinLogger 存储接口
type BinLogger interface {
	Close() error
	TruncateFront(index uint64) error
	ReadLastSegSection(index uint64) (data []byte, fileName string, offset, blkLen uint64, err error)
	LastIndex() (index uint64, err error)
	Write(index uint64, data []byte) (fileName string, offset, blkLen uint64,
		startHeight, endHeight uint64, needRecordDB bool, err error)
	ReadFileSection(isDeCompressed bool, fiIndex *storePb.StoreInfo) ([]byte, error)
	// 检查文件是否解压缩，和文件上次访问时间戳
	CheckDecompressFileExist(filePath string) (bool, int64, error)
	// 解压缩文件，压缩算法啥的交给底层filestore，
	DeCompressFile(compressFileName string) (string, error)
	// 查询压缩高度
	GetCanCompressHeight() uint64
	// 压缩指定高度下的文件
	CompressFileByStartHeight(startHeight uint64) (string, error)
	// 删除指定文件
	TryRemoveFile(fileName string, isDeCompressed bool) (bool, error)
}
