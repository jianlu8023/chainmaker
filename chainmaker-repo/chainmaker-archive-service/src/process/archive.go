/*
Copyright (C) BABEC. All rights reserved.


SPDX-License-Identifier: Apache-2.0
*/

// Package process define process logic
package process

import (
	"bytes"
	"fmt"
	"sync/atomic"

	"chainmaker.org/chainmaker-archive-service/src/archive_utils"
	archivePb "chainmaker.org/chainmaker/pb-go/v2/archivecenter"
	"chainmaker.org/chainmaker/pb-go/v2/config"
	storePb "chainmaker.org/chainmaker/pb-go/v2/store"
	"chainmaker.org/chainmaker/utils/v2"
)

// ArchiveBlock 归档区块
// @receiver cp
// @param chainGenesisHash
// @param block
// @return ArchiveStatus
// @return error
func (cp *ChainProcessor) ArchiveBlock(chainGenesisHash string,
	block *storePb.BlockWithRWSet) (archivePb.ArchiveStatus, error) {
	archive_utils.GlobalServerLatch.Add(1)
	defer archive_utils.GlobalServerLatch.Done()
	tempBlockHeight := block.Block.Header.GetBlockHeight()
	memArchivedHeight := atomic.LoadUint64(&cp.ArchivedHeight)
	cp.logger.Debugf("ArchiveBlock tempBlockHeight %d , memArchivedHeight %d", tempBlockHeight, memArchivedHeight)
	if tempBlockHeight > memArchivedHeight+1 {
		return archivePb.ArchiveStatus_ArchiveStatusFailed,
			fmt.Errorf("archivedHeight(%d) and blockHeight(%d) mismatched", memArchivedHeight, tempBlockHeight)
	}
	if tempBlockHeight <= memArchivedHeight {
		// 这个区块已经归档过了，查一下这个区块和归档的区块是否一致
		archivedHeader, archivedHeaderError := cp.GetBlockHeaderByHeight(tempBlockHeight)
		if archivedHeaderError != nil {
			cp.logger.Errorf("ArchiveBlock GetBlockHeaderByHeight %d got error %s", tempBlockHeight, archivedHeaderError.Error())
			return archivePb.ArchiveStatus_ArchiveStatusFailed, archivedHeaderError
		}
		if !bytes.Equal(archivedHeader.BlockHash, block.Block.Header.BlockHash) ||
			!bytes.Equal(archivedHeader.PreBlockHash, block.Block.Header.PreBlockHash) {
			return archivePb.ArchiveStatus_ArchiveStatusFailed, fmt.Errorf("block(%s) not equal archived block(%s)",
				block.Block.GetBlockHashStr(), archive_utils.EncodeGenesisHashToString(archivedHeader.BlockHash))
		}
		return archivePb.ArchiveStatus_ArchiveStatusHasArchived, nil
	}
	// 需要归档的区块
	// 1. 校验数据本身的区块hash是否正确
	hashTrue, hashError := archive_utils.VerifyBlockHash(cp.currentChainConfig.Crypto.GetHash(), block.Block)
	if hashError != nil {
		return archivePb.ArchiveStatus_ArchiveStatusFailed, hashError
	}
	if !hashTrue {
		return archivePb.ArchiveStatus_ArchiveStatusFailed, archive_utils.ErrorBlockHashVerifyFailed
	}
	if tempBlockHeight > 0 {
		// 2. 校验
		lastBlock, lastBlockError := cp.GetLastBlock()
		if lastBlockError != nil {
			cp.logger.Errorf("ArchiveBlock GetLastBlock error %s", lastBlockError.Error())
			return archivePb.ArchiveStatus_ArchiveStatusFailed, lastBlockError
		}
		if !bytes.Equal(lastBlock.Header.BlockHash, block.Block.Header.PreBlockHash) {
			return archivePb.ArchiveStatus_ArchiveStatusFailed, archive_utils.ErrorArchiveHashIllegal
		}
	}
	// 3. 看一下是否为配置区块
	var nowConfig *config.ChainConfig
	if utils.IsConfBlock(block.Block) {
		tempConfig, tempConfigError := archive_utils.RestoreChainConfigFromBlock(block.Block)
		if tempConfigError != nil {
			cp.logger.Errorf("ArchiveBlock RestoreChainConfigFromBlock error %s ", tempConfigError.Error())
			return archivePb.ArchiveStatus_ArchiveStatusFailed, tempConfigError
		}
		nowConfig = tempConfig
	}
	appendError := cp.appendBlock(block)
	if appendError != nil {
		cp.logger.Errorf("ArchiveBlock appendBlock error %s ", appendError.Error())
		return archivePb.ArchiveStatus_ArchiveStatusFailed, appendError
	}
	// 3. 更新一下缓存
	if tempBlockHeight > 0 {
		atomic.SwapUint64(&cp.ArchivedHeight, memArchivedHeight+1)
	}
	if utils.IsConfBlock(block.Block) {
		cp.currentChainConfig = nowConfig
	}
	return archivePb.ArchiveStatus_ArchiveStatusSuccess, nil
}

// MarkInArchive 标记链正在归档文件
func (cp *ChainProcessor) MarkInArchive() {
	atomic.StoreInt32(&cp.InArchive, 1)
}

// MarkNotInArchive 标记链未在归档文件
func (cp *ChainProcessor) MarkNotInArchive() {
	atomic.StoreInt32(&cp.InArchive, 0)
}
