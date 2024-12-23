/*
Copyright (C) BABEC. All rights reserved.


SPDX-License-Identifier: Apache-2.0
*/

// Package process define process logic
package process

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"testing"
	"time"

	"chainmaker.org/chainmaker-archive-service/src/serverconf"
	acPb "chainmaker.org/chainmaker/pb-go/v2/accesscontrol"
	archivePb "chainmaker.org/chainmaker/pb-go/v2/archivecenter"
	commonPb "chainmaker.org/chainmaker/pb-go/v2/common"
	configPb "chainmaker.org/chainmaker/pb-go/v2/config"
	storePb "chainmaker.org/chainmaker/pb-go/v2/store"
	"chainmaker.org/chainmaker/pb-go/v2/syscontract"
	"chainmaker.org/chainmaker/utils/v2"
	"github.com/gogo/protobuf/proto"
	"github.com/stretchr/testify/assert"
)

func initConfig(t *testing.T) {
	readError := serverconf.ReadConfigFile("../../configs/config.yml")
	assert.Nil(t, readError)
	t.Logf("configs:  %+v", *serverconf.GlobalServerCFG)
}

func cleanTestDatas() {
	os.RemoveAll("../log")
	os.RemoveAll("../service_datas")
}

func generateTxId2(chainId string, height uint64, index int) string {
	txIdBytes := sha256.Sum256([]byte(fmt.Sprintf("%s-%d-%d", chainId, height, index)))
	return hex.EncodeToString(txIdBytes[:32])
}
func createConfigBlock(chainId string, height uint64,
	preConfHeight uint64, preBlockHash []byte) *commonPb.Block {
	config := configPb.ChainConfig{
		ChainId: chainId,
		Version: "2.3.0",
		Crypto: &configPb.CryptoConfig{
			Hash: "SHA256",
		},
	}
	configBytes, _ := proto.Marshal(&config)
	block := &commonPb.Block{
		Header: &commonPb.BlockHeader{
			ChainId:      chainId,
			BlockHeight:  height,
			Proposer:     &acPb.Member{MemberInfo: []byte("User1")},
			BlockType:    0,
			PreBlockHash: preBlockHash,
		},
		Txs: []*commonPb.Transaction{
			{
				Payload: &commonPb.Payload{
					ChainId:      chainId,
					TxType:       commonPb.TxType_INVOKE_CONTRACT,
					ContractName: syscontract.SystemContract_CHAIN_CONFIG.String(),
				},
				Sender: &commonPb.EndorsementEntry{
					Signer:    &acPb.Member{OrgId: "org1", MemberInfo: []byte("cert1...")},
					Signature: []byte("sign1"),
				},
				Result: &commonPb.Result{
					Code: commonPb.TxStatusCode_SUCCESS,
					ContractResult: &commonPb.ContractResult{
						Result: configBytes,
					},
				},
			},
		},
	}
	block.Txs[0].Payload.TxId = generateTxId2(chainId, height, 0)
	block.Header.PreConfHeight = preConfHeight
	block.Header.BlockHash, _ = utils.CalcBlockHash("SHA256", block)

	return block
}

func createBlockAndRWSets4(chainId string, height uint64,
	txNum int, preConfHeight uint64,
	preBlockHash []byte) (*commonPb.Block, []*commonPb.TxRWSet) {
	block := &commonPb.Block{
		Header: &commonPb.BlockHeader{
			ChainId:        chainId,
			BlockHeight:    height,
			Proposer:       &acPb.Member{MemberInfo: []byte("User1")},
			BlockTimestamp: time.Now().UnixNano() / 1e6,
			PreBlockHash:   preBlockHash,
		},
		Dag:            &commonPb.DAG{},
		AdditionalData: &commonPb.AdditionalData{},
	}

	for i := 0; i < txNum; i++ {
		tx := &commonPb.Transaction{
			Payload: &commonPb.Payload{
				ChainId: chainId,
				TxId:    generateTxId2(chainId, height, i),
			},
			Sender: &commonPb.EndorsementEntry{
				Signer:    &acPb.Member{OrgId: "org1", MemberInfo: []byte("cert1...")},
				Signature: []byte("sign1"),
			},
			Result: &commonPb.Result{
				Code: commonPb.TxStatusCode_SUCCESS,
				ContractResult: &commonPb.ContractResult{
					Result: []byte("ok"),
				},
			},
		}
		block.Txs = append(block.Txs, tx)
	}
	block.Header.PreConfHeight = preConfHeight
	var txRWSets []*commonPb.TxRWSet
	for i := 0; i < txNum; i++ {
		key := fmt.Sprintf("key_%d", i)
		value := fmt.Sprintf("value_%d", i)
		txRWset := &commonPb.TxRWSet{
			TxId: block.Txs[i].Payload.TxId,
			TxWrites: []*commonPb.TxWrite{
				{
					Key:          []byte(key),
					Value:        []byte(value),
					ContractName: "contract1",
				},
			},
		}
		txRWSets = append(txRWSets, txRWset)
	}
	block.Header.BlockHash, _ = utils.CalcBlockHash("SHA256", block)
	return block, txRWSets
}

// nolint
func TestInitProcessorMgr(t *testing.T) {
	initConfig(t)
	mgr := InitProcessorMgr(&serverconf.GlobalServerCFG.StoreageCFG,
		&serverconf.GlobalServerCFG.LogCFG)
	_ = mgr
	defer mgr.Close()
	defer cleanTestDatas()
	block0 := createConfigBlock("chain1", 0, 0, nil)
	block0WithRWSet := storePb.BlockWithRWSet{
		Block: block0,
	}
	block1, block1RWSet := createBlockAndRWSets4("chain1", 1, 2, 0, block0.Header.BlockHash)
	block1WithRwSet := storePb.BlockWithRWSet{
		Block:    block1,
		TxRWSets: block1RWSet,
	}
	block2, block2RWSet := createBlockAndRWSets4("chain1", 2, 5, 0, block1.Header.BlockHash)
	block2WithRwSet := storePb.BlockWithRWSet{
		Block:    block2,
		TxRWSets: block2RWSet,
	}
	block3, block3RWSet := createBlockAndRWSets4("chain1", 3, 7, 0, block2.Header.BlockHash)
	block3WithRwSet := storePb.BlockWithRWSet{
		Block:    block3,
		TxRWSets: block3RWSet,
	}
	block4, block4RWSet := createBlockAndRWSets4("chain1", 4, 5, 0, block3.Header.BlockHash)
	block4WithRwSet := storePb.BlockWithRWSet{
		Block:    block4,
		TxRWSets: block4RWSet,
	}
	block5, block5RWSet := createBlockAndRWSets4("chain1", 5, 7, 0, block4.Header.BlockHash)
	block5WithRwSet := storePb.BlockWithRWSet{
		Block:    block5,
		TxRWSets: block5RWSet,
	}
	// test register
	mgr.RegisterChainByGenesisBlock(&block0WithRWSet, &serverconf.GlobalServerCFG.LogCFG)
	mgr.MarkInArchive(block0.GetBlockHashStr(), mgr.GetChainProcessorByHash)
	// test archive
	statu1, statu1Err := mgr.ArchiveBlock(block0.GetBlockHashStr(), &block1WithRwSet, mgr.GetChainProcessorByHash)
	assert.Nil(t, statu1Err)
	t.Logf("status1 : %+v", statu1)
	statu2, statu2Err := mgr.ArchiveBlock(block0.GetBlockHashStr(), &block2WithRwSet, mgr.GetChainProcessorByHash)
	assert.Nil(t, statu2Err)
	t.Logf("status2 : %+v", statu2)
	statu3, statu3Err := mgr.ArchiveBlock(block0.GetBlockHashStr(), &block3WithRwSet, mgr.GetChainProcessorByHash)
	assert.Nil(t, statu3Err)
	t.Logf("status3 : %+v", statu3)
	statu4, statu4Err := mgr.ArchiveBlock(block0.GetBlockHashStr(), &block4WithRwSet, mgr.GetChainProcessorByHash)
	assert.Nil(t, statu4Err)
	t.Logf("status4 : %+v", statu4)
	statu5, statu5Err := mgr.ArchiveBlock(block0.GetBlockHashStr(), &block5WithRwSet, mgr.GetChainProcessorByHash)
	assert.Nil(t, statu5Err)
	t.Logf("status5 : %+v", statu5)

	statu11, statu11Err := mgr.ArchiveBlock(block0.GetBlockHashStr(), &block1WithRwSet, mgr.GetChainProcessorByHash)
	assert.Nil(t, statu11Err)
	assert.Equal(t, statu11, statu11, archivePb.ArchiveStatus_ArchiveStatusHasArchived)
	mgr.MarkNotInArchive(block0.GetBlockHashStr(), mgr.GetChainProcessorByHash)
	// test get archived height
	archivedHeight, get1Err := mgr.GetArchivedHeight(block0.GetBlockHashStr(), mgr.GetChainProcessorByHash)
	assert.Nil(t, get1Err)
	assert.Equal(t, archivedHeight, block5.Header.BlockHeight)
	// test get range blocks
	blocks, get2Err := mgr.GetRangeBlocks(block0.GetBlockHashStr(), 0, 2,
		mgr.GetChainProcessorByHash)
	assert.Nil(t, get2Err)
	t.Logf("[0 , 2] blocks : %+v ", blocks)
	assert.Equal(t, len(blocks), 3)
	// test get archive status
	mgr.GetInArchiveAndArchivedHeight(block0.GetBlockHashStr(),
		mgr.GetChainProcessorByHash)
	// test check block exists
	exist1, exist1Err := mgr.BlockExists(block0.GetBlockHashStr(), block1.Header.BlockHash,
		mgr.GetChainProcessorByHash)
	assert.Nil(t, exist1Err)
	assert.True(t, exist1)
	// test get block by hash
	_, hashErr := mgr.GetBlockByHash(block0.GetBlockHashStr(), block1.Header.BlockHash,
		mgr.GetChainProcessorByHash)
	assert.Nil(t, hashErr)
	// test get height by hash
	height, heightErr := mgr.GetHeightByHash(block0.GetBlockHashStr(), block1.Header.BlockHash,
		mgr.GetChainProcessorByHash)
	assert.Nil(t, heightErr)
	assert.Equal(t, height, block1.Header.BlockHeight)
	// get block header by height
	header, headerErr := mgr.GetBlockHeaderByHeight(block0.GetBlockHashStr(), 1,
		mgr.GetChainProcessorByHash)
	assert.Nil(t, headerErr)
	assert.Equal(t, uint64(1), header.BlockHeight)
	// get block by height
	blockO, blockOErr := mgr.GetBlock(block0.GetBlockHashStr(), 1,
		mgr.GetChainProcessorByHash)
	assert.Nil(t, blockOErr)
	assert.Equal(t, blockO.Header.BlockHeight, uint64(1))
	// test get all chain info
	chainStatus := mgr.GetAllChainInfo()
	t.Logf("chainstatus : %+v ", chainStatus)
	// test compress
	begin1, begin2, beginErr := mgr.CompressUnderHeight(block0.GetBlockHashStr(), 5, mgr.GetChainProcessorByHash)
	assert.Nil(t, beginErr)
	assert.Equal(t, begin1, begin2)
	// test get compress status
	_, _, compressStatusErr := mgr.GetCompressStatus(block0.GetBlockHashStr(), mgr.GetChainProcessorByHash)
	assert.Nil(t, compressStatusErr)
	// test get chain config
	_, configErr := mgr.GetChainConfigByHeight(block0.GetBlockHashStr(), 1,
		mgr.GetChainProcessorByHash)
	assert.Nil(t, configErr)
	// get commonblock by txid
	commonBlock, commonblockErr := mgr.GetCommonBlockInfoByTxId(block0.GetBlockHashStr(),
		block1.Txs[0].Payload.TxId, mgr.GetChainProcessorByHash)
	assert.Nil(t, commonblockErr)
	assert.Equal(t, block1.Header.BlockHeight, commonBlock.Block.Header.BlockHeight)
	// get merkle path by txid
	mgr.GetMerklePathByTxId(block0.GetBlockHashStr(),
		block1.Txs[0].Payload.TxId, mgr.GetChainProcessorByHash)
	//
	mgr.GetBlockByHeightTruncate(block0.GetBlockHashStr(), mgr.GetChainProcessorByHash,
		block0.Header.BlockHeight, true, 10000, "")
	mgr.GetTxByTxIdTruncate(block0.GetBlockHashStr(), block1.Txs[0].Payload.TxId,
		mgr.GetChainProcessorByHash, true, 10000, "")
	// get common block by hash
	commonBlockByHash, commonBlockByHashErr := mgr.GetCommonBlockInfoByHash(block0.GetBlockHashStr(),
		block1.Header.BlockHash, mgr.GetChainProcessorByHash)
	assert.Nil(t, commonBlockByHashErr)
	assert.Equal(t, block1.Header.BlockHash, commonBlockByHash.Block.Header.BlockHash)
	// get common block by height
	commonBlockByHeight, commonBlockByHeightErr := mgr.GetCommonBlockInfoByHeight(block0.GetBlockHashStr(),
		1, mgr.GetChainProcessorByHash)
	assert.Nil(t, commonBlockByHeightErr)
	assert.Equal(t, block1.Header.BlockHeight, commonBlockByHeight.Block.Header.BlockHeight)
	//get block with rwset by height
	blockWithRWSetByHeight, blockWithRWSetByHeightErr := mgr.GetBlockWithRWSetByHeight(block0.GetBlockHashStr(),
		1, mgr.GetChainProcessorByHash)
	assert.Nil(t, blockWithRWSetByHeightErr)
	assert.Equal(t, block1.Header.BlockHeight, blockWithRWSetByHeight.Block.Header.BlockHeight)
	// get tx
	block1Tx1 := block1.Txs[0]
	// test get tx
	_, gettxErr := mgr.GetTx(block0.GetBlockHashStr(), block1Tx1.Payload.TxId,
		mgr.GetChainProcessorByHash)
	assert.Nil(t, gettxErr)
	// test get tx rwset
	_, getTxRwSetErr := mgr.GetTxRWSet(block0.GetBlockHashStr(), block1Tx1.Payload.TxId,
		mgr.GetChainProcessorByHash)
	assert.Nil(t, getTxRwSetErr)
	// test get tx with block info
	_, getTxWithBlockInfoErr := mgr.GetTxWithBlockInfo(block0.GetBlockHashStr(), block1Tx1.Payload.TxId,
		mgr.GetChainProcessorByHash)
	assert.Nil(t, getTxWithBlockInfoErr)
	// test get common transactio info
	_, getCommonTransactionInfoErr := mgr.GetCommonTransactionInfo(block0.GetBlockHashStr(), block1Tx1.Payload.TxId,
		mgr.GetChainProcessorByHash)
	assert.Nil(t, getCommonTransactionInfoErr)
	// test get common transaction with rwset
	_, getCommonTransactionWithRWsetErr := mgr.GetCommonTransactionWithRWSet(block0.GetBlockHashStr(), block1Tx1.Payload.TxId,
		mgr.GetChainProcessorByHash)
	assert.Nil(t, getCommonTransactionWithRWsetErr)
	// test get tx info only
	_, getTxInfoOnlyErr := mgr.GetTxInfoOnly(block0.GetBlockHashStr(), block1Tx1.Payload.TxId,
		mgr.GetChainProcessorByHash)
	assert.Nil(t, getTxInfoOnlyErr)
	// test get tx height
	getTxHeight, getTxHeightErr := mgr.GetTxHeight(block0.GetBlockHashStr(), block1Tx1.Payload.TxId,
		mgr.GetChainProcessorByHash)
	assert.Nil(t, getTxHeightErr)
	assert.Equal(t, block1.Header.BlockHeight, getTxHeight)
	// test tx exists
	txExists, txExistsErr := mgr.TxExists(block0.GetBlockHashStr(), block1Tx1.Payload.TxId,
		mgr.GetChainProcessorByHash)
	assert.Nil(t, txExistsErr)
	assert.True(t, txExists)
	// test get tx confirmed time
	_, getTxConfirmedTimeErr := mgr.GetTxConfirmedTime(block0.GetBlockHashStr(), block1Tx1.Payload.TxId,
		mgr.GetChainProcessorByHash)
	assert.Nil(t, getTxConfirmedTimeErr)
	// test get last block
	getLastBlock, getLastBlockErr := mgr.GetLastBlock(block0.GetBlockHashStr(),
		mgr.GetChainProcessorByHash)
	assert.Nil(t, getLastBlockErr)
	assert.Equal(t, getLastBlock.Header.BlockHeight, block5.Header.BlockHeight)
	// test get filtered block
	_, getFilteredBlockErr := mgr.GetFilteredBlock(block0.GetBlockHashStr(), 1,
		mgr.GetChainProcessorByHash)
	assert.Nil(t, getFilteredBlockErr)
	// test get last save point
	getLastSavePoint, getLastSavePointErr := mgr.GetLastSavepoint(block0.GetBlockHashStr(),
		mgr.GetChainProcessorByHash)
	assert.Nil(t, getLastSavePointErr)
	assert.Equal(t, getLastSavePoint, block5.Header.BlockHeight)
	// test get last config block
	getLastConfigBlock, getLastConfigBlockErr := mgr.GetLastConfigBlock(block0.GetBlockHashStr(),
		mgr.GetChainProcessorByHash)
	assert.Nil(t, getLastConfigBlockErr)
	assert.Equal(t, block0.Header.BlockHeight, getLastConfigBlock.Header.BlockHeight)
	// test get last config block height
	getLastConfigBlockHeight, getLastConfigBlockHeightErr := mgr.GetLastConfigBlockHeight(block0.GetBlockHashStr(),
		mgr.GetChainProcessorByHash)
	assert.Nil(t, getLastConfigBlockHeightErr)
	assert.Equal(t, block0.Header.BlockHeight, getLastConfigBlockHeight)
	// test get block by tx
	_, getBlockByTxErr := mgr.GetBlockByTx(block0.GetBlockHashStr(), block1Tx1.Payload.TxId,
		mgr.GetChainProcessorByHash)
	assert.Nil(t, getBlockByTxErr)
}
