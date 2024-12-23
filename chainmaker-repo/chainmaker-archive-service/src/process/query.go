/*
Copyright (C) BABEC. All rights reserved.


SPDX-License-Identifier: Apache-2.0
*/

// Package process define process logic
package process

import (
	"encoding/json"
	"errors"
	"sync/atomic"

	"chainmaker.org/chainmaker/common/v2/crypto/hash"
	commonPb "chainmaker.org/chainmaker/pb-go/v2/common"
	configPb "chainmaker.org/chainmaker/pb-go/v2/config"
	storePb "chainmaker.org/chainmaker/pb-go/v2/store"
	"chainmaker.org/chainmaker/utils/v2"
	"github.com/gogo/protobuf/proto"
)

// GetChainConfigByBlockHeight 根据区块高度查询小于该高度的最近的链配置信息
// @receiver cp
// @param blockHeight
// @return ChainConfig
// @return error
func (cp *ChainProcessor) GetChainConfigByBlockHeight(blockHeight uint64) (*configPb.ChainConfig, error) {
	archivedHeight, _ := cp.GetArchivedHeight()
	// 说明该高度尚未被归档
	if archivedHeight < blockHeight {
		return nil, nil
	}
	header, headerError := cp.GetBlockHeaderByHeight(blockHeight)
	if headerError != nil {
		cp.logger.Errorf("GetChainConfigByBlockHeight height %d header error %s",
			blockHeight, headerError.Error())
		return nil, headerError
	}
	configBlock, configBlockError := cp.GetBlock(header.PreConfHeight)
	if configBlockError != nil {
		cp.logger.Errorf("GetChainConfigByBlockHeight height %d config height %d error %s",
			blockHeight, header.PreConfHeight, configBlockError.Error())
		return nil, configBlockError
	}
	cfg, cfgError := cp.parseConfigFromBlock(configBlock)
	if cfgError != nil {
		cp.logger.Errorf("GetChainConfigByBlockHeight height %d config height %d parse error %s",
			blockHeight, header.PreConfHeight, cfgError.Error())
		return nil, cfgError
	}
	return cfg, nil
}

// parseConfigFromBlock 解析链配置信息
// @receiver cp
// @param block
// @return ChainConfig
// @return error
func (cp *ChainProcessor) parseConfigFromBlock(block *commonPb.Block) (*configPb.ChainConfig, error) {
	if block == nil {
		return nil, nil
	}
	if len(block.Txs) == 0 || block.Txs[0] == nil ||
		block.Txs[0].Result == nil || block.Txs[0].Result.ContractResult == nil ||
		block.Txs[0].Result.ContractResult.Result == nil {
		return nil, errors.New("tx is not config tx")
	}
	result := block.Txs[0].Result.ContractResult.Result
	chainConfig := &configPb.ChainConfig{}
	err := proto.Unmarshal(result, chainConfig)
	if err != nil {
		return nil, err
	}
	return chainConfig, nil
}

// GetArchivedHeight 查询当前的归档高度
// @receiver cp
// @return uint64
// @return error
func (cp *ChainProcessor) GetArchivedHeight() (uint64, error) {
	height := atomic.LoadUint64(&cp.ArchivedHeight)
	return height, nil
}

// GetRangeBlocks 查询指定范围区块
// @receiver cp
// @param start
// @param end
// @return []BlockWithRWSet
// @return error
func (cp *ChainProcessor) GetRangeBlocks(start, end uint64) ([]*storePb.BlockWithRWSet, error) {
	height := atomic.LoadUint64(&cp.ArchivedHeight)
	// 查询到了尚未归档的数据
	if end < start || start > height {
		return nil, nil
	}
	if end > height {
		end = height
	}
	retBlocks := make([]*storePb.BlockWithRWSet, 0, end-start+1)
	for i := start; i <= end; i++ {
		tempBlock, tempError := cp.blockDB.GetBlockWithRWSetByHeight(i)
		if tempError != nil {
			return retBlocks, tempError
		}
		if tempBlock == nil { // 没有查到数据
			return retBlocks, nil
		}
		retBlocks = append(retBlocks, tempBlock)
	}
	return retBlocks, nil
}

// GetInArchiveAndArchivedHeight 查询归档中心状态
// @receiver cp
// @return uint64
// @return bool
// @return error
func (cp *ChainProcessor) GetInArchiveAndArchivedHeight() (uint64, bool, error) {
	height := atomic.LoadUint64(&cp.ArchivedHeight)
	inArchive := atomic.LoadInt32(&cp.InArchive)
	return height, inArchive == 1, nil
}

// BlockExists 区块是否存在
// @receiver cp
// @param blockHash
// @return bool
// @return error
func (cp *ChainProcessor) BlockExists(blockHash []byte) (bool, error) {
	return cp.blockDB.BlockExists(blockHash)
}

// GetBlockByHash 根据区块hash查询区块
// @receiver cp
// @param blockHash
// @return Block
// @return error
func (cp *ChainProcessor) GetBlockByHash(blockHash []byte) (*commonPb.Block, error) {
	return cp.blockDB.GetBlockByHash(blockHash)
}

// GetHeightByHash 根据区块hash查询高度
// @receiver cp
// @param blockHash
// @return uint64
// @return error
func (cp *ChainProcessor) GetHeightByHash(blockHash []byte) (uint64, error) {
	return cp.blockDB.GetHeightByHash(blockHash)
}

// GetBlockHeaderByHeight 根据高度查询区块头
// @receiver cp
// @param height
// @return BlockHeader
// @return error
func (cp *ChainProcessor) GetBlockHeaderByHeight(height uint64) (*commonPb.BlockHeader, error) {
	return cp.blockDB.GetBlockHeaderByHeight(height)
}

// GetBlock 根据高度查询区块
// @receiver cp
// @param height
// @return Block
// @return error
func (cp *ChainProcessor) GetBlock(height uint64) (*commonPb.Block, error) {
	return cp.blockDB.GetBlock(height)
}

// GetTx 根据txid查询交易
// @receiver cp
// @param txId
// @return Transaction
// @return error
func (cp *ChainProcessor) GetTx(txId string) (*commonPb.Transaction, error) {
	return cp.blockDB.GetTx(txId)
}

// GetTxWithBlockInfo 根据txid查询区块信息
// @receiver cp
// @param txId
// @return TransactionStoreInfo
// @return error
func (cp *ChainProcessor) GetTxWithBlockInfo(txId string) (*storePb.TransactionStoreInfo, error) {
	return cp.blockDB.GetTxWithBlockInfo(txId)
}

// GetTxInfoOnly 根据txid查询交易信息
// @receiver cp
// @param txId
// @return TransactionStoreInfo
// @return error
func (cp *ChainProcessor) GetTxInfoOnly(txId string) (*storePb.TransactionStoreInfo, error) {
	return cp.blockDB.GetTxInfoOnly(txId)
}

// GetTxHeight 根据txid获取区块高度
// @receiver cp
// @param txId
// @return uint64
// @return error
func (cp *ChainProcessor) GetTxHeight(txId string) (uint64, error) {
	return cp.blockDB.GetTxHeight(txId)
}

// TxExists 交易是否存在
// @receiver cp
// @param txId
// @return bool
// @return error
func (cp *ChainProcessor) TxExists(txId string) (bool, error) {
	return cp.blockDB.TxExists(txId)
}

// GetTxConfirmedTime 查询交易确认时间
// @receiver cp
// @param txId
// @return int64
// @return error
func (cp *ChainProcessor) GetTxConfirmedTime(txId string) (int64, error) {
	return cp.blockDB.GetTxConfirmedTime(txId)
}

// GetLastBlock 查询归档中心的最新区块
// @receiver cp
// @return Block
// @return error
func (cp *ChainProcessor) GetLastBlock() (*commonPb.Block, error) {
	return cp.blockDB.GetLastBlock()
}

// GetFilteredBlock 根据高度查询序列化的区块信息
// @receiver cp
// @param height
// @return SerializedBlock
// @return error
func (cp *ChainProcessor) GetFilteredBlock(height uint64) (*storePb.SerializedBlock, error) {
	return cp.blockDB.GetFilteredBlock(height)
}

// GetLastSavepoint 获取归档中心最新的归档高度
// @receiver cp
// @return uint64
// @return error
func (cp *ChainProcessor) GetLastSavepoint() (uint64, error) {
	return cp.blockDB.GetLastSavepoint()
}

// GetLastConfigBlock 查询归档中心最新的配置区块
// @receiver cp
// @return Block
// @return error
func (cp *ChainProcessor) GetLastConfigBlock() (*commonPb.Block, error) {
	return cp.blockDB.GetLastConfigBlock()
}

// GetLastConfigBlockHeight 查询归档中心最新的配置区块高度
// @receiver cp
// @return uint64
// @return error
func (cp *ChainProcessor) GetLastConfigBlockHeight() (uint64, error) {
	return cp.blockDB.GetLastConfigBlockHeight()
}

// GetBlockByTx 根据txid查询区块
// @receiver cp
// @param txId
// @return Block
// @return error
func (cp *ChainProcessor) GetBlockByTx(txId string) (*commonPb.Block, error) {
	return cp.blockDB.GetBlockByTx(txId)
}

// GetTxRWSet 根据txid查询读写集
// @receiver cp
// @param txId
// @return TxRWSet
// @return error
func (cp *ChainProcessor) GetTxRWSet(txId string) (*commonPb.TxRWSet, error) {
	return cp.blockDB.GetTxRWSet(txId)
}

// GetBlockWithRWSetByHeight 根据高度查询区块
// @receiver cp
// @param height
// @return BlockWithRWSet
// @return error
func (cp *ChainProcessor) GetBlockWithRWSetByHeight(height uint64) (*storePb.BlockWithRWSet, error) {
	return cp.blockDB.GetBlockWithRWSetByHeight(height)
}

// GetMerklePathByTxId 根据txid计算merkerl-path
// @receiver cp
// @param txId
// @return [][]byte
// @return error
func (cp *ChainProcessor) GetMerklePathByTxId(txId string) ([]byte, error) {
	var merkleTree [][]byte
	txBlock, txBlockErr := cp.blockDB.GetBlockByTx(txId)
	if txBlockErr != nil {
		cp.logger.Errorf("GetMerklePathByTxId GetBlockByTx [%s] error [%s]", txId, txBlockErr.Error())
		return nil, txBlockErr
	}
	if txBlock == nil {
		return nil, nil
	}
	hashType := cp.currentChainConfig.Crypto.Hash
	hashes := make([][]byte, len(txBlock.Txs))
	for i, tx := range txBlock.Txs {
		var txHash []byte
		txHash, txBlockErr = utils.CalcTxHash(hashType, tx)
		if txBlockErr != nil {
			cp.logger.Errorf("GetMerklePathByTxId  txId [%s] CalcTxHash error [%s]",
				txId, txBlockErr.Error())
			return nil, txBlockErr
		}
		hashes[i] = txHash
	}
	merkleTree, txBlockErr = hash.BuildMerkleTree(hashType, hashes)
	if txBlockErr != nil {
		cp.logger.Errorf("GetMerklePathByTxId  txId [%s] BuildMerkleTree error [%s]",
			txId, txBlockErr.Error())
		return nil, txBlockErr
	}
	merklePaths := make([][]byte, 0)
	hash.GetMerklePath(hashType, []byte(txId), merkleTree, &merklePaths, false)
	merklePathsBytes, _ := json.Marshal(merklePaths)
	//cp.logger.Debugf("GetMerklePathByTxId txId [%s] ,"+
	//	"block: %+v , merkleTree %+v ,merklePath %+v , merklePathBytes: %+v ", txId, txBlock,
	//	merkleTree, merklePaths, merklePathsBytes)
	return merklePathsBytes, nil
}

// GetTxByTxIdTruncate 根据TxId获得Transaction对象，并根据参数进行截断
// @receiver cp
// @param txId
// @param withRWSet
// @param truncateLength
// @param truncateModel
// @return *common.TransactionInfoWithRWSet
// @return error
func (cp *ChainProcessor) GetTxByTxIdTruncate(txId string, withRWSet bool, truncateLength int,
	truncateModel string) (*commonPb.TransactionInfoWithRWSet, error) {
	var txInfoWithRWSet commonPb.TransactionInfoWithRWSet
	txTransactionStoreInfo, txTransactionStoreInfoErr :=
		cp.blockDB.GetTxWithBlockInfo(txId)
	if txTransactionStoreInfoErr != nil {
		return nil, txTransactionStoreInfoErr
	}
	if txTransactionStoreInfo == nil {
		return nil, nil
	}
	txInfoWithRWSet.Transaction = txTransactionStoreInfo.Transaction
	txInfoWithRWSet.BlockHeight = txTransactionStoreInfo.BlockHeight
	txInfoWithRWSet.BlockHash = txTransactionStoreInfo.BlockHash
	txInfoWithRWSet.TxIndex = txTransactionStoreInfo.TxIndex
	txInfoWithRWSet.BlockTimestamp = txTransactionStoreInfo.BlockTimestamp
	if withRWSet {
		txRwSet, txRwSetErr := cp.blockDB.GetTxRWSet(txId)
		if txRwSetErr != nil {
			cp.logger.Errorf("GetTxByTxIdTruncate txId [%s] GetTxRWSet error %s",
				txId, txRwSetErr.Error())
			return &txInfoWithRWSet, txRwSetErr
		}
		txInfoWithRWSet.RwSet = txRwSet
	}
	if truncateLength > 0 {
		truncate := newTruncateConfig(truncateLength, truncateModel)
		truncate.TruncateTx(txInfoWithRWSet.Transaction)
	}
	return &txInfoWithRWSet, nil
}

// GetBlockByHeightTruncate 根据区块高度获得区块，
// GetBlockByHeightTruncate 但是对于长度超过100的ParameterValue则清空Value
// @receiver cp
// @param blockHeight
// @param withRWSet
// @param truncateLength 截断的长度设置
// @param truncateModel 截断的模式设置:hash,truncate,empty
// @return *common.BlockInfo
// @return error
func (cp *ChainProcessor) GetBlockByHeightTruncate(blockHeight uint64,
	withRWSet bool, truncateLength int,
	truncateModel string) (*commonPb.BlockInfo, error) {
	var retBlock commonPb.BlockInfo
	blockWithRwSet, blockWithRwSetErr := cp.blockDB.GetBlockWithRWSetByHeight(blockHeight)
	if blockWithRwSetErr != nil {
		cp.logger.Errorf("GetBlockByHeightTruncate height [%d] error %s",
			blockHeight, blockWithRwSetErr.Error())
		return nil, blockWithRwSetErr
	}
	if blockWithRwSet == nil {
		return nil, nil
	}
	retBlock.Block = blockWithRwSet.Block
	if withRWSet {
		retBlock.RwsetList = blockWithRwSet.TxRWSets
	}
	if truncateLength > 0 {
		truncate := newTruncateConfig(truncateLength, truncateModel)
		truncate.TruncateBlockWithRWSet(&retBlock)
	}
	return &retBlock, nil
}
