/*
Copyright (C) BABEC. All rights reserved.


SPDX-License-Identifier: Apache-2.0
*/

// Package process define process logic
package process

import (
	"strconv"

	"chainmaker.org/chainmaker-archive-service/src/archive_utils"
	"chainmaker.org/chainmaker-archive-service/src/store/filestore"
	storePb "chainmaker.org/chainmaker/pb-go/v2/store"
	"github.com/gogo/protobuf/proto"
)

// StartCurrentInfo 记录区块文件信息
type StartCurrentInfo struct {
	Start      uint64
	Current    uint64
	NeedRecord bool
}

// Recover 函数,扫描fdb文件中的区块高度，
// 对比一下kv中的高度，如果不一致复原，这个函数调用应该在blockfile初始化（读文件）之后再调用
func (cp *ChainProcessor) Recover() error {
	lastFileIndex, lastFileIndexError := cp.blockFile.LastIndex()
	if lastFileIndexError != nil {
		return lastFileIndexError
	}
	// 不需要恢复数据
	if lastFileIndex < 1 {
		return nil
	}
	dbSaveIndex, dbSaveIndexError := cp.blockDB.GetLastSavepoint()
	if dbSaveIndexError != nil {
		return dbSaveIndexError
	}
	if dbSaveIndex+1 >= lastFileIndex {
		// db 保存的块高和文件块高是一致的，不需要恢复
		return nil
	}
	cp.logger.Debugf("recover file lastindex %d , lastsave %d ", lastFileIndex, dbSaveIndex)
	blockStartArrs := make([]StartCurrentInfo, 0, lastFileIndex-dbSaveIndex-1)
	blocks := make([]*filestore.BlockWithSerializedInfo, 0, lastFileIndex-dbSaveIndex-1)
	for iterHeight := dbSaveIndex + 1; iterHeight <= lastFileIndex; {
		//遍历这些区块将区块信息提交到db
		tempBlock, tempBlockError := cp.getBlockFromLog(iterHeight)
		if tempBlockError != nil {
			return tempBlockError
		}
		blocks = append(blocks, tempBlock)
		startHeight, _ := strconv.ParseUint(tempBlock.Index.FileName, 10, 64)
		blockStartArrs = append(blockStartArrs, StartCurrentInfo{
			Start:   startHeight,
			Current: iterHeight,
		})
	}
	for i := 0; i < len(blockStartArrs)-2; i++ {
		if blockStartArrs[i].Start != blockStartArrs[i+1].Start {
			blockStartArrs[i].NeedRecord = true
		}
	}
	for i := 0; i < len(blockStartArrs); i++ {
		wrapBlock := &filestore.StartEndWrapBlock{
			BlockSerializedInfo: blocks[i],
		}
		if blockStartArrs[i].NeedRecord {
			wrapBlock.StartHeight = blockStartArrs[i].Start
			wrapBlock.EndHeight = blockStartArrs[i].Current
			wrapBlock.NeedRecord = true
		}
		writeError := cp.writeDb(wrapBlock)
		if writeError != nil {
			return writeError
		}
	}
	return nil
}

// getBlockFromLog 从文件中读取出区块
// @param num
// @return BlockWithSerializedInfo
// @return error
func (cp *ChainProcessor) getBlockFromLog(num uint64) (*filestore.BlockWithSerializedInfo, error) {
	var (
		err       error
		data      []byte
		storeInfo *storePb.StoreInfo
	)
	storeInfo = &storePb.StoreInfo{}
	data, storeInfo.FileName, storeInfo.Offset, storeInfo.ByteLen, err = cp.blockFile.ReadLastSegSection(num)
	if err != nil {
		return nil, err
	}
	blockWithRWSet, err := filestore.DeserializeBlock(data)
	if err != nil {
		return nil, err
	}
	buf, ok := cp.protoBufferPool.Get().(*proto.Buffer)
	if !ok {
		return nil, archive_utils.ErrorGetBufPool
	}
	buf.Reset()
	_, s, serializeError := filestore.SerializeBlock(blockWithRWSet)
	if s == nil {
		return nil, serializeError
	}
	s.Index = storeInfo
	cp.protoBufferPool.Put(buf)
	return s, nil
}
