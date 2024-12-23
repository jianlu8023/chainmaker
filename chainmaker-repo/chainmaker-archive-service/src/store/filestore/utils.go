/*
Copyright (C) BABEC. All rights reserved.


SPDX-License-Identifier: Apache-2.0
*/

// Package filestore define file operation
package filestore

import (
	"os"

	storePb "chainmaker.org/chainmaker/pb-go/v2/store"
	"github.com/gogo/protobuf/proto"
)

// ConstructDBIndexInfo 索引序列化
// @param blkIndex
// @param offset
// @param byteLen
// @return []byte
func ConstructDBIndexInfo(blkIndex *storePb.StoreInfo, offset, byteLen uint64) []byte {
	//return []byte(fmt.Sprintf("%s:%d:%d", blkIndex.FileName, blkIndex.Offset+offset, byteLen))
	index := &storePb.StoreInfo{
		FileName: blkIndex.FileName,
		Offset:   blkIndex.Offset + offset,
		ByteLen:  byteLen,
	}
	indexByte, err := proto.Marshal(index)
	if err != nil {
		return nil
	}
	return indexByte
}

// DecodeValueToIndex 索引反序列化
// @param blockIndexByte
// @return StoreInfo
// @return error
func DecodeValueToIndex(blockIndexByte []byte) (*storePb.StoreInfo, error) {
	if len(blockIndexByte) == 0 {
		return nil, nil
	}
	var blockIndex storePb.StoreInfo
	err := proto.Unmarshal(blockIndexByte, &blockIndex)
	if err != nil {
		return nil, err
	}
	return &blockIndex, nil
}

// PathExists 判断path是否存在
// @param path
// @return bool
// @return error
func PathExists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}

// DecompressFileInfo 解压缩文件信息
type DecompressFileInfo struct {
	FileName  string `json:"file_name"`
	CreatedAt int64  `json:"created_at"`
}

var (
	// DecompressKVPrefix 解压缩文件kv保存的前缀
	DecompressKVPrefix = "dkp"
	// CompressKVPrefix 压缩文件kv保存的前缀
	CompressKVPrefix = "ckp"
)
