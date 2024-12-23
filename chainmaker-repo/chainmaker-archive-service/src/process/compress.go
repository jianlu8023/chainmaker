/*
Copyright (C) BABEC. All rights reserved.


SPDX-License-Identifier: Apache-2.0
*/

// Package process define process logic
package process

import (
	"sync/atomic"

	"chainmaker.org/chainmaker-archive-service/src/archive_utils"
)

// CompressUnderHeight 压缩指定高度下的所有区块
// @receiver cp
// @param height
// @return uint64
// @return uint64
// @return error
func (cp *ChainProcessor) CompressUnderHeight(height uint64) (uint64, uint64, error) {
	//加锁
	archive_utils.GlobalServerLatch.Add(1)
	defer archive_utils.GlobalServerLatch.Done()
	locked := atomic.CompareAndSwapUint32(&cp.inCompress, 0, 1)
	if !locked {
		return 0, 0, archive_utils.ErrorChainInCompress
	}
	// 释放掉锁
	defer atomic.StoreUint32(&cp.inCompress, 0)
	return cp.blockDB.CompressUnderHeight(height)

}

// CheckInCompress 检查一下当前链是否正在进行压缩
// @receiver cp
// @return bool
// @return uint64
func (cp *ChainProcessor) CheckInCompress() (bool, error) {
	incompress := atomic.LoadUint32(&cp.inCompress)
	return incompress == 1, nil
}

// GetChainCompressStatus 检查一下当前链的压缩状态（当前压缩的最大高度，链是否正在压缩）
// @receiver cp
// @return uint64
// @return bool
// @return error
func (cp *ChainProcessor) GetChainCompressStatus() (uint64, bool, error) {
	compressedHeight, compressedError := cp.blockDB.GetCompressedHeight()
	if compressedError != nil {
		return 0, false, compressedError
	}
	inCompress := atomic.LoadUint32(&cp.inCompress)
	return compressedHeight, inCompress == 1, nil
}
