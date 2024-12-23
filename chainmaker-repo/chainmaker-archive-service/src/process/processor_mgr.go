/*
Copyright (C) BABEC. All rights reserved.


SPDX-License-Identifier: Apache-2.0
*/

// Package process define process logic
package process

import (
	"encoding/base64"
	"errors"
	"fmt"
	"path"
	"strconv"
	"strings"
	"sync"

	"chainmaker.org/chainmaker-archive-service/src/archive_utils"
	"chainmaker.org/chainmaker-archive-service/src/logger"
	"chainmaker.org/chainmaker-archive-service/src/serverconf"
	"chainmaker.org/chainmaker-archive-service/src/store/filestore"
	"chainmaker.org/chainmaker-archive-service/src/store/levelkv"
	archivePb "chainmaker.org/chainmaker/pb-go/v2/archivecenter"
	commonPb "chainmaker.org/chainmaker/pb-go/v2/common"
	configPb "chainmaker.org/chainmaker/pb-go/v2/config"
	storePb "chainmaker.org/chainmaker/pb-go/v2/store"
	"chainmaker.org/chainmaker/protocol/v2"
	"github.com/gogo/protobuf/proto"
	"github.com/mitchellh/mapstructure"
)

var (
	// keyMapGenesisHashToChainIdPrefix hash到链id的映射
	keyMapGenesisHashToChainIdPrefix = "gChain"
	// keyMapGenesisHashToGenesisBlockPrefix hash到创世块的映射
	keyMapGenesisHashToGenesisBlockPrefix = "gBlock"
	// keyMapGenesisHashToConfigPrefix hash到初始配置的影射
	keyMapGenesisHashToConfigPrefix = "gConfig"

	// keyCAPrefix ca 证书prefix
	keyCAPrefix = "gCA"
	// keyHttpToken 生成的http token
	keyHttpToken = "httpToken"
	// registerInProcess 正在注册
	registerInProcess = int8(1)
	// 注册完毕，可以接受请求
	registerProcessed = int8(2)
)

// ProcessorMgr 链管理器
type ProcessorMgr struct {
	registerLock            sync.Mutex                // 主要用于启动，或者注册的时候写入genesishash 时候使用
	processorMp             sync.Map                  // chainGenesisHash -> ChainProcessor
	genesisInfoMp           sync.Map                  // chainGenesisHash -> *GenesisInfo
	genesisChainRegisteryMp sync.Map                  // 1,正在注册;2,注册成功，可以接受请求;
	systemDB                protocol.DBHandle         // 存放系统信息的数据库
	mgrConfig               *serverconf.StorageConfig // 存放的是配置信息
	caArrays                [][]byte                  // ca 证书列表，todo暂时放到内存
	caUpdatedChanel         chan struct{}             // 增加证书的时候会向这个channel中写数据
	logger                  *logger.WrappedLogger
	httpToken               string //
}

// SendUpdateCASignal 发送更新CA证书信号,异步非阻塞操作
func (pm *ProcessorMgr) SendUpdateCASignal() {
	go func() {
		pm.caUpdatedChanel <- struct{}{} //发送一个信号
	}()

}

// ReceiveUpdateCASignal 接收更新CA证书信号
func (pm *ProcessorMgr) ReceiveUpdateCASignal() chan struct{} {
	return pm.caUpdatedChanel
}

// GenesisInfo 链初始区块信息
type GenesisInfo struct {
	ChainId       string
	GenesisConfig *configPb.ChainConfig
	GenesisBlock  *commonPb.Block
}

// GetHttpToken 返回服务的http的token
func (pm *ProcessorMgr) GetHttpToken() string {
	return pm.httpToken
}

// InitProcessorMgr 初始化归档中心链管理器
func InitProcessorMgr(config *serverconf.StorageConfig, globalLogCFG *serverconf.LogConfig) *ProcessorMgr {
	processorMgrOnce.Do(func() {
		systemLogger := logger.NewLogger("PROCESSOR", &serverconf.LogConfig{
			LogLevel:     globalLogCFG.LogLevel,
			LogPath:      fmt.Sprintf("%s/processor.log", globalLogCFG.LogPath),
			LogInConsole: globalLogCFG.LogInConsole,
			ShowColor:    globalLogCFG.ShowColor,
			MaxSize:      globalLogCFG.MaxSize,
			MaxBackups:   globalLogCFG.MaxBackups,
			MaxAge:       globalLogCFG.MaxAge,
			Compress:     globalLogCFG.Compress,
		})
		_processMgrInstance = &ProcessorMgr{
			mgrConfig:       config,
			caArrays:        [][]byte{},
			logger:          systemLogger,
			caUpdatedChanel: make(chan struct{}),
		}
		// 初始化db
		var kvConfig levelkv.LevelDbConfig
		decodeConfigError := mapstructure.Decode(config.LevelDBCFG, &kvConfig)
		if decodeConfigError != nil {
			panic(fmt.Sprintf("decode config error (%s)", decodeConfigError))
		}

		_processMgrInstance.systemDB = levelkv.NewLevelDBHandle(&levelkv.NewLevelDBOptions{
			Config:     &kvConfig,
			PathPrefix: config.StorePath,
			Logger:     systemLogger,
		})
		// 初始化所有的链处理器
		_processMgrInstance.LoadAllRegisterChainGenesisFromDB(globalLogCFG)
		// load http token
		_processMgrInstance.LoadOrSaveHttpToken()

	})
	return _processMgrInstance
}

// LoadOrSaveHttpToken 加载http的token头
func (pm *ProcessorMgr) LoadOrSaveHttpToken() {
	keyValue, keyError := pm.systemDB.Get([]byte(keyHttpToken))
	if keyError != nil {
		pm.logger.Errorf("LoadOrSaveHttpToken got error %s ", keyError.Error())
		panic("LoadOrSaveHttpToken got error " + keyError.Error())
	}
	if len(keyValue) > 0 {
		pm.httpToken = string(keyValue)
		pm.logger.Infof("LoadOrSaveHttpToken http token %s ", pm.httpToken)
		return
	}
	randToken := archive_utils.RandSeq(10)
	saveError := pm.systemDB.Put([]byte(keyHttpToken), []byte(randToken))
	if saveError != nil {
		pm.logger.Errorf("LoadOrSaveHttpToken save token %s error %s ", randToken, saveError.Error())
		panic("LoadOrSaveHttpToken save token error " + saveError.Error())
	}
	pm.httpToken = randToken
	pm.logger.Infof("LoadOrSaveHttpToken http token %s ", pm.httpToken)
}

var (
	processorMgrOnce    sync.Once
	_processMgrInstance *ProcessorMgr
)

// FnGetChainProcessor 查询链处理器函数类型
type FnGetChainProcessor func(string) (*ChainProcessor, error)

// GetChainProcessorByHash 根据创世块的hash来获取链处理器
func (pm *ProcessorMgr) GetChainProcessorByHash(chainGenesisHash string) (*ChainProcessor, error) {
	chainProcessor, ok := pm.processorMp.Load(chainGenesisHash)
	if !ok || chainProcessor == nil {
		return nil, errors.New("chain genesis not exists")
	}
	retProcessor, _ := chainProcessor.(*ChainProcessor)
	return retProcessor, nil
}

// GetChainProcessorById 根据链id查询链处理器
func (pm *ProcessorMgr) GetChainProcessorById(chainId string) (*ChainProcessor, error) {

	return pm.GetChainProcessorByHash(chainId)
}

// // ArchiveBlocks 归档区块
// func (pm *ProcessorMgr) ArchiveBlocks(chainGenesisHash string,
// 	blocks []*storePb.BlockWithRWSet,
// 	processorGetter FnGetChainProcessor) (uint64, uint64, error) {
// 	processor, err := processorGetter(chainGenesisHash)
// 	if err != nil {
// 		return 0, 0, err
// 	}
// 	return processor.ArchiveBlocks(blocks)
// }

// ArchiveBlock 归档区块
// @receiver pm
// @param chainGenesisHash
// @param block
// @param processorGetter
// @return ArchiveStatus
// @return error
func (pm *ProcessorMgr) ArchiveBlock(chainGenesisHash string,
	block *storePb.BlockWithRWSet,
	processorGetter FnGetChainProcessor) (archivePb.ArchiveStatus, error) {
	processor, err := processorGetter(chainGenesisHash)
	if err != nil {
		return 0, err
	}
	return processor.ArchiveBlock(chainGenesisHash, block)
}

// MarkInArchive 标记链正在归档
func (pm *ProcessorMgr) MarkInArchive(chainGenesisHash string, processorGetter FnGetChainProcessor) {
	processor, err := processorGetter(chainGenesisHash)
	if err != nil {
		return
	}
	processor.MarkInArchive()
}

// MarkNotInArchive 标记链未在归档
func (pm *ProcessorMgr) MarkNotInArchive(chainGenesisHash string, processorGetter FnGetChainProcessor) {
	processor, err := processorGetter(chainGenesisHash)
	if err != nil {
		return
	}
	processor.MarkNotInArchive()
}

//	GetArchivedHeight 获取当前归档高度
//
// @receiver pm
// @param chainGenesisHash
// @param processorGetter
// @return uint64
// @return error
func (pm *ProcessorMgr) GetArchivedHeight(chainGenesisHash string,
	processorGetter FnGetChainProcessor) (uint64, error) {
	processor, err := processorGetter(chainGenesisHash)
	if err != nil {
		return 0, err
	}
	return processor.GetArchivedHeight()
}

// GetRangeBlocks 批量查询区块接口
// @receiver pm
// @param chainGenesisHash
// @param processorGetter
// @param start
// @param end
// @return []BlockWithRWSet
// @return error
func (pm *ProcessorMgr) GetRangeBlocks(chainGenesisHash string, start, end uint64,
	processorGetter FnGetChainProcessor) ([]*storePb.BlockWithRWSet, error) {
	processor, err := processorGetter(chainGenesisHash)
	if err != nil {
		return nil, err
	}
	return processor.GetRangeBlocks(start, end)
}

// GetInArchiveAndArchivedHeight 查询当前归档高度，指定的链是否正在归档
func (pm *ProcessorMgr) GetInArchiveAndArchivedHeight(chainGenesisHash string,
	processorGetter FnGetChainProcessor) (uint64, bool, error) {
	processor, err := processorGetter(chainGenesisHash)
	if err != nil {
		return 0, false, err
	}
	return processor.GetInArchiveAndArchivedHeight()
}

// BlockExists 根据区块hash查询区块是否存在
func (pm *ProcessorMgr) BlockExists(chainGenesisHash string, blockHash []byte,
	processorGetter FnGetChainProcessor) (bool, error) {
	processor, err := processorGetter(chainGenesisHash)
	if err != nil {
		return false, err
	}
	return processor.BlockExists(blockHash)
}

// GetBlockByHash 根据区块hash查询区块
func (pm *ProcessorMgr) GetBlockByHash(chainGenesisHash string, blockHash []byte,
	processorGetter FnGetChainProcessor) (*commonPb.Block, error) {
	processor, err := processorGetter(chainGenesisHash)
	if err != nil {
		return nil, err
	}
	return processor.GetBlockByHash(blockHash)
}

// GetHeightByHash 根据hash查询块高度
func (pm *ProcessorMgr) GetHeightByHash(chainGenesisHash string, blockHash []byte,
	processorGetter FnGetChainProcessor) (uint64, error) {
	processor, err := processorGetter(chainGenesisHash)
	if err != nil {
		return 0, err
	}
	return processor.GetHeightByHash(blockHash)
}

// GetBlockHeaderByHeight 根据块高查询区块头
func (pm *ProcessorMgr) GetBlockHeaderByHeight(chainGenesisHash string, height uint64,
	processorGetter FnGetChainProcessor) (*commonPb.BlockHeader, error) {
	processor, err := processorGetter(chainGenesisHash)
	if err != nil {
		return nil, err
	}
	return processor.GetBlockHeaderByHeight(height)
}

// GetBlock 根据块高查询区块
func (pm *ProcessorMgr) GetBlock(chainGenesisHash string, height uint64,
	processorGetter FnGetChainProcessor) (*commonPb.Block, error) {
	processor, err := processorGetter(chainGenesisHash)
	if err != nil {
		return nil, err
	}
	return processor.GetBlock(height)
}

// GetTx 根据事务id查询事务
func (pm *ProcessorMgr) GetTx(chainGenesisHash string, txId string,
	processorGetter FnGetChainProcessor) (*commonPb.Transaction, error) {
	processor, err := processorGetter(chainGenesisHash)
	if err != nil {
		return nil, err
	}
	return processor.GetTx(txId)
}

// GetTxWithBlockInfo 根据事务id查询事务信息，TransactionStoreInfo为空（文件索引信息无用）
func (pm *ProcessorMgr) GetTxWithBlockInfo(chainGenesisHash string, txId string,
	processorGetter FnGetChainProcessor) (*storePb.TransactionStoreInfo, error) {
	processor, err := processorGetter(chainGenesisHash)
	if err != nil {
		return nil, err
	}
	return processor.GetTxWithBlockInfo(txId)
}

// GetCommonTransactionInfo 根据txid获取交易信息
func (pm *ProcessorMgr) GetCommonTransactionInfo(chainGenesisHash string, txId string,
	processorGetter FnGetChainProcessor) (*commonPb.TransactionInfo, error) {
	processor, err := processorGetter(chainGenesisHash)
	if err != nil {
		return nil, err
	}
	txStore, txStoreError := processor.GetTxWithBlockInfo(txId)
	if txStoreError != nil {
		return nil, txStoreError
	}
	if txStore == nil {
		return nil, nil
	}
	var retTransactionInfo commonPb.TransactionInfo
	retTransactionInfo.BlockHeight = txStore.BlockHeight
	retTransactionInfo.BlockHash = txStore.BlockHash
	retTransactionInfo.BlockTimestamp = txStore.BlockTimestamp
	retTransactionInfo.Transaction = txStore.Transaction
	retTransactionInfo.TxIndex = txStore.TxIndex
	return &retTransactionInfo, nil
}

// GetCommonTransactionWithRWSet 根据txid查询带有读写集合的事务信息
func (pm *ProcessorMgr) GetCommonTransactionWithRWSet(chainGenesisHash string, txId string,
	processorGetter FnGetChainProcessor) (*commonPb.TransactionInfoWithRWSet, error) {
	processor, err := processorGetter(chainGenesisHash)
	if err != nil {
		return nil, err
	}
	txStore, txStoreError := processor.GetTxWithBlockInfo(txId)
	if txStoreError != nil {
		return nil, txStoreError
	}
	if txStore == nil {
		return nil, nil
	}
	var retTransaction commonPb.TransactionInfoWithRWSet
	txRwSet, txRwSetError := processor.GetTxRWSet(txId)
	if txRwSetError != nil {
		return nil, txRwSetError
	}
	retTransaction.BlockHash = txStore.BlockHash
	retTransaction.BlockHeight = txStore.BlockHeight
	retTransaction.BlockTimestamp = txStore.BlockTimestamp
	retTransaction.Transaction = txStore.Transaction
	retTransaction.TxIndex = txStore.TxIndex
	if txRwSet != nil {
		retTransaction.RwSet = txRwSet
	}

	return &retTransaction, nil
}

// GetTxInfoOnly 获得除Tx之外的其他TxInfo信息
func (pm *ProcessorMgr) GetTxInfoOnly(chainGenesisHash string, txId string,
	processorGetter FnGetChainProcessor) (*storePb.TransactionStoreInfo, error) {
	processor, err := processorGetter(chainGenesisHash)
	if err != nil {
		return nil, err
	}
	return processor.GetTxInfoOnly(txId)
}

// GetTxHeight 根据事务id查询块高
func (pm *ProcessorMgr) GetTxHeight(chainGenesisHash string, txId string,
	processorGetter FnGetChainProcessor) (uint64, error) {
	processor, err := processorGetter(chainGenesisHash)
	if err != nil {
		return 0, err
	}
	return processor.GetTxHeight(txId)
}

// TxExists 查询事务是否存在
func (pm *ProcessorMgr) TxExists(chainGenesisHash string, txId string,
	processorGetter FnGetChainProcessor) (bool, error) {
	processor, err := processorGetter(chainGenesisHash)
	if err != nil {
		return false, err
	}
	return processor.blockDB.TxExists(txId)
}

// GetTxConfirmedTime 查询区块确认时间
func (pm *ProcessorMgr) GetTxConfirmedTime(chainGenesisHash string,
	txId string, processorGetter FnGetChainProcessor) (int64, error) {
	processor, err := processorGetter(chainGenesisHash)
	if err != nil {
		return 0, err
	}
	return processor.GetTxConfirmedTime(txId)
}

// GetLastBlock 查询最后一个区块
func (pm *ProcessorMgr) GetLastBlock(chainGenesisHash string,
	processorGetter FnGetChainProcessor) (*commonPb.Block, error) {
	processor, err := processorGetter(chainGenesisHash)
	if err != nil {
		return nil, err
	}
	return processor.GetLastBlock()
}

// GetFilteredBlock 根据高度查询序列化后的区块信息
func (pm *ProcessorMgr) GetFilteredBlock(chainGenesisHash string,
	height uint64, processorGetter FnGetChainProcessor) (*storePb.SerializedBlock, error) {
	processor, err := processorGetter(chainGenesisHash)
	if err != nil {
		return nil, err
	}
	return processor.GetFilteredBlock(height)
}

// GetLastSavepoint 返回当前最后一个归档高度
func (pm *ProcessorMgr) GetLastSavepoint(chainGenesisHash string,
	processorGetter FnGetChainProcessor) (uint64, error) {
	processor, err := processorGetter(chainGenesisHash)
	if err != nil {
		return 0, err
	}
	return processor.GetLastSavepoint()
}

// GetLastConfigBlock 查询最后一个归档区块
func (pm *ProcessorMgr) GetLastConfigBlock(chainGenesisHash string,
	processorGetter FnGetChainProcessor) (*commonPb.Block, error) {
	processor, err := processorGetter(chainGenesisHash)
	if err != nil {
		return nil, err
	}
	return processor.GetLastConfigBlock()
}

// GetLastConfigBlockHeight 查询最后一个归档区块的高度
func (pm *ProcessorMgr) GetLastConfigBlockHeight(chainGenesisHash string,
	processorGetter FnGetChainProcessor) (uint64, error) {
	processor, err := processorGetter(chainGenesisHash)
	if err != nil {
		return 0, err
	}
	return processor.GetLastConfigBlockHeight()
}

// GetBlockByTx 根据事务id查询区块信息
func (pm *ProcessorMgr) GetBlockByTx(chainGenesisHash string,
	txId string, processorGetter FnGetChainProcessor) (*commonPb.Block, error) {
	processor, err := processorGetter(chainGenesisHash)
	if err != nil {
		return nil, err
	}
	return processor.GetBlockByTx(txId)
}

// GetTxRWSet 根据事务id查询事务读写集
func (pm *ProcessorMgr) GetTxRWSet(chainGenesisHash string,
	txId string, processorGetter FnGetChainProcessor) (*commonPb.TxRWSet, error) {
	processor, err := processorGetter(chainGenesisHash)
	if err != nil {
		return nil, err
	}
	return processor.GetTxRWSet(txId)
}

// GetBlockWithRWSetByHeight 根据高度查询区块
func (pm *ProcessorMgr) GetBlockWithRWSetByHeight(chainGenesisHash string, height uint64,
	processorGetter FnGetChainProcessor) (*storePb.BlockWithRWSet, error) {
	processor, err := processorGetter(chainGenesisHash)
	if err != nil {
		return nil, err
	}
	return processor.GetBlockWithRWSetByHeight(height)
}

// GetCommonBlockInfoByHeight 适配http的query方法,根据块高查区块
func (pm *ProcessorMgr) GetCommonBlockInfoByHeight(chainGenesisHash string, height uint64,
	processorGetter FnGetChainProcessor) (*commonPb.BlockInfo, error) {
	fullBlock, err := pm.GetBlockWithRWSetByHeight(chainGenesisHash,
		height, processorGetter)
	if err != nil {
		return nil, err
	}
	var retInfo commonPb.BlockInfo
	if fullBlock != nil {
		retInfo.Block = fullBlock.Block
		retInfo.RwsetList = fullBlock.TxRWSets
		return &retInfo, nil
	}
	return nil, nil

}

// GetCommonBlockInfoByHash 适配http的query方法,根据hash查区块
func (pm *ProcessorMgr) GetCommonBlockInfoByHash(chainGenesisHash string,
	blockHash []byte, processorGetter FnGetChainProcessor) (*commonPb.BlockInfo, error) {
	height, err := pm.GetHeightByHash(chainGenesisHash, blockHash, processorGetter)
	if err == archive_utils.ErrValueNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return pm.GetCommonBlockInfoByHeight(chainGenesisHash, height, processorGetter)
}

// GetCommonBlockInfoByTxId ,根据txid查询区块
func (pm *ProcessorMgr) GetCommonBlockInfoByTxId(chainGenesisHash string,
	txId string, processorGetter FnGetChainProcessor) (*commonPb.BlockInfo, error) {
	height, err := pm.GetTxHeight(chainGenesisHash, txId, processorGetter)
	if err == archive_utils.ErrValueNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return pm.GetCommonBlockInfoByHeight(chainGenesisHash, height, processorGetter)
}

// GetChainConfigByHeight 根据高度查询配置区块
func (pm *ProcessorMgr) GetChainConfigByHeight(chainGenesisHash string,
	blockHeight uint64, processorGetter FnGetChainProcessor) (*configPb.ChainConfig, error) {
	processor, err := processorGetter(chainGenesisHash)
	if err != nil {
		return nil, err
	}
	return processor.GetChainConfigByBlockHeight(blockHeight)
}

// CompressUnderHeight 压缩指定高度下的区块
func (pm *ProcessorMgr) CompressUnderHeight(chainGenesisHash string,
	height uint64, processorGetter FnGetChainProcessor) (uint64, uint64, error) {
	processor, err := processorGetter(chainGenesisHash)
	if err != nil {
		return 0, 0, err
	}
	return processor.CompressUnderHeight(height)
}

// GetCompressStatus 查询链归档中心状态
func (pm *ProcessorMgr) GetCompressStatus(chainGenesisHash string,
	processorGetter FnGetChainProcessor) (uint64, bool, error) {
	processor, err := processorGetter(chainGenesisHash)
	if err != nil {
		return 0, false, err
	}
	return processor.GetChainCompressStatus()
}

// GetMerklePathByTxId 计算txid的merkle tree 路径
func (pm *ProcessorMgr) GetMerklePathByTxId(chainGenesisHash string,
	txId string, processorGetter FnGetChainProcessor) ([]byte, error) {
	processor, err := processorGetter(chainGenesisHash)
	if err != nil {
		return nil, err
	}
	return processor.GetMerklePathByTxId(txId)
}

// GetTxByTxIdTruncate 根据TxId获得Transaction对象，并根据参数进行截断
func (pm *ProcessorMgr) GetTxByTxIdTruncate(chainGenesisHash string,
	txId string, processorGetter FnGetChainProcessor,
	withRWSet bool, truncateLength int,
	truncateModel string) (*commonPb.TransactionInfoWithRWSet, error) {
	processor, err := processorGetter(chainGenesisHash)
	if err != nil {
		return nil, err
	}
	return processor.GetTxByTxIdTruncate(txId, withRWSet, truncateLength, truncateModel)
}

// GetBlockByHeightTruncate 根据区块高度获得区块,根据参数进行截断
func (pm *ProcessorMgr) GetBlockByHeightTruncate(chainGenesisHash string,
	processorGetter FnGetChainProcessor, blockHeight uint64,
	withRWSet bool, truncateLength int,
	truncateModel string) (*commonPb.BlockInfo, error) {
	processor, err := processorGetter(chainGenesisHash)
	if err != nil {
		return nil, err
	}
	return processor.GetBlockByHeightTruncate(blockHeight,
		withRWSet, truncateLength, truncateModel)
}

// LoadAllRegisterChainGenesisFromDB load chain info from db
// and init chain processor
func (pm *ProcessorMgr) LoadAllRegisterChainGenesisFromDB(globalLogCFG *serverconf.LogConfig) {
	listIter, listIterError := pm.systemDB.NewIteratorWithPrefix([]byte(
		keyMapGenesisHashToChainIdPrefix))
	if listIterError != nil {
		panic(fmt.Sprintf("init iterator error(%s)", listIterError.Error()))
	}
	defer listIter.Release()
	var genesisArrays []string
	var chainIdArrays []string
	var blockArrays []*storePb.BlockWithRWSet
	var configArrays []*configPb.ChainConfig
	for listIter.Next() {
		// genesis-block-hash
		tempGenesisString := strings.TrimPrefix(string(listIter.Key()), keyMapGenesisHashToChainIdPrefix)
		_, parseErr := archive_utils.DecodeStringToGenesisHash(tempGenesisString)
		if parseErr != nil {
			panic(fmt.Sprintf("parse (%s) to genesis hash error (%s)", tempGenesisString, parseErr.Error()))
		}
		// chain-id
		tempChainId := string(listIter.Value())
		genesisArrays = append(genesisArrays, tempGenesisString)
		chainIdArrays = append(chainIdArrays, tempChainId)
	}
	// 查block,查config
	for i := 0; i < len(genesisArrays); i++ {
		blockBytes, blockGetError := pm.systemDB.Get([]byte(keyMapGenesisHashToGenesisBlockPrefix + genesisArrays[i]))
		if blockGetError != nil {
			panic(fmt.Sprintf("get blockbytes(%s) from db error (%s)", genesisArrays[i], blockGetError.Error()))
		}
		tempBlock := &storePb.BlockWithRWSet{}
		blockUnMarshalError := proto.Unmarshal(blockBytes, tempBlock)
		if blockUnMarshalError != nil {
			panic(fmt.Sprintf("restore block(%s) error (%s)", genesisArrays[i], blockUnMarshalError.Error()))
		}
		blockArrays = append(blockArrays, tempBlock)
		// 查config
		configBytes, configGetError := pm.systemDB.Get([]byte(keyMapGenesisHashToConfigPrefix + genesisArrays[i]))
		if configGetError != nil {
			panic(fmt.Sprintf("get configbytes(%s) from db error(%s) ", genesisArrays[i], configGetError.Error()))
		}
		tempConfig := &configPb.ChainConfig{}
		configUnmarshalError := proto.Unmarshal(configBytes, tempConfig)
		if configUnmarshalError != nil {
			panic(fmt.Sprintf("restore config(%s) error (%s)", genesisArrays[i], configUnmarshalError.Error()))
		}
		configArrays = append(configArrays, tempConfig)
	}
	// 将加载到的信息加载到内存中
	for i := 0; i < len(genesisArrays); i++ {
		pm.addChainInfo(genesisArrays[i], &GenesisInfo{
			ChainId:       chainIdArrays[i],
			GenesisConfig: configArrays[i],
			GenesisBlock:  blockArrays[i].Block,
		})
		tempProcessor, tempProcessorError := pm.constructChainProcessor(genesisArrays[i], chainIdArrays[i],
			globalLogCFG,
			configArrays[i], blockArrays[i], false)
		if tempProcessorError != nil {
			panic(fmt.Sprintf("init chain(%s) meet error", chainIdArrays[i]))
		}
		// save chain processor
		pm.processorMp.Store(genesisArrays[i], tempProcessor)
		// mark chain has registered ,can process data
		pm.genesisChainRegisteryMp.Store(genesisArrays[i], registerProcessed)
		pm.logger.Infof("LoadAllRegisterChainGenesisFromDB [%d] , chainId [%s], chainConfig [%+v] , block [%+v]",
			i, chainIdArrays[i], *configArrays[i], *blockArrays[i])
	}
}

// ChainStatus 链状态
type ChainStatus struct {
	ChainId        string `json:"chainId"`
	GenesisHashStr string `json:"genesisHashStr"`
	GenesisHashHex string `json:"genesisHashHex"`
	ArchivedHeight uint64 `json:"archivedHeight"`
	InArchive      bool   `json:"inArchive"`
}

// GetAllChainInfo 获取归档中心所有链的状态
func (pm *ProcessorMgr) GetAllChainInfo() []ChainStatus {
	var statusArrs []ChainStatus
	pm.genesisInfoMp.Range(func(key, value interface{}) bool {
		tempGenesis, _ := key.(string)
		tempInfo, _ := value.(*GenesisInfo)
		// pm.logger.Infof("tempGenesis [%s], tempInfo : %+v", tempGenesis, *tempInfo)
		var tempStatus ChainStatus
		tempStatus.ChainId = tempInfo.ChainId
		tempStatus.GenesisHashHex = tempGenesis
		tempStatus.GenesisHashStr = base64.StdEncoding.EncodeToString(tempInfo.GenesisBlock.Hash())
		height, status, err := pm.GetInArchiveAndArchivedHeight(tempGenesis, pm.GetChainProcessorByHash)
		tempStatus.ArchivedHeight = height
		tempStatus.InArchive = status
		if err != nil {
			pm.logger.Errorf("GetAllChainInfo %s got error %s", tempGenesis, err.Error())
		} else {
			statusArrs = append(statusArrs, tempStatus)
		}
		return true
	})
	return statusArrs
}

func (pm *ProcessorMgr) addChainInfo(genesisHash string, info *GenesisInfo) {
	pm.genesisInfoMp.Store(genesisHash, info)
}

func (pm *ProcessorMgr) constructChainProcessor(genesisHash, chainId string,
	globalLogCFG *serverconf.LogConfig,
	genesisConfig *configPb.ChainConfig,
	genesisBlock *storePb.BlockWithRWSet, isFirst bool) (*ChainProcessor, error) {
	var storeCFG levelkv.LevelDbConfig
	decodeError := mapstructure.Decode(pm.mgrConfig, &storeCFG)
	if decodeError != nil {
		panic(fmt.Sprintf("decode level config error(%s)", decodeError.Error()))
	}
	levelLog := logger.NewLogger(genesisHash, &serverconf.LogConfig{
		LogPath:      fmt.Sprintf("%s/%s", globalLogCFG.LogPath, genesisHash),
		LogLevel:     globalLogCFG.LogLevel,
		LogInConsole: globalLogCFG.LogInConsole,
		ShowColor:    globalLogCFG.ShowColor,
		MaxSize:      globalLogCFG.MaxSize,
		MaxBackups:   globalLogCFG.MaxBackups,
		MaxAge:       globalLogCFG.MaxAge,
		Compress:     globalLogCFG.Compress,
	})
	// build levelkv
	kvdb := levelkv.NewLevelDBHandle(&levelkv.NewLevelDBOptions{
		Config:           &storeCFG,
		ChainGenesisHash: genesisHash,
		PathPrefix:       pm.mgrConfig.IndexPath,
		Logger:           levelLog, // 单独的初始化日志
	})
	// build filestore
	fileOpts := filestore.GetDefaultOptions()
	if pm.mgrConfig.LogDBSegmentAsync {
		fileOpts.NoSync = true
	}
	if pm.mgrConfig.LogDBSegmentSize > 0 {
		fileOpts.SegmentSize = pm.mgrConfig.LogDBSegmentSize * 1024 * 1024
	}
	if pm.mgrConfig.SegmentCacheSize > 0 {
		fileOpts.SegmentCacheSize = pm.mgrConfig.SegmentCacheSize
	}
	if !pm.mgrConfig.UseMmap {
		fileOpts.UseMmap = false
	}
	if pm.mgrConfig.RetainSeconds > 0 {
		fileOpts.DecompressFileRetainSeconds = int64(pm.mgrConfig.RetainSeconds) //设置一下最大保留时间
	}
	if pm.mgrConfig.CompressSeconds > 0 {
		fileOpts.MaxCompressTimeInSeconds = pm.mgrConfig.CompressSeconds //设置最大超时时长
	}
	if pm.mgrConfig.ScanIntervalSeconds <= 0 {
		pm.mgrConfig.ScanIntervalSeconds = 600 // default
	}
	normalPath := path.Join(pm.mgrConfig.BfdbPath, genesisHash)
	compressPath := path.Join(pm.mgrConfig.CompressPath, genesisHash)
	deCompressPath := path.Join(pm.mgrConfig.DecompressPath, genesisHash)
	blockStore, blockStoreError := filestore.Open(normalPath, compressPath, deCompressPath, fileOpts, levelLog)
	if blockStoreError != nil {
		return nil, blockStoreError
	}
	// build blockdb
	blockStoreDB := filestore.NewBlockFileDB(genesisHash, kvdb, levelLog, blockStore)
	return NewChainProcessor(genesisHash, genesisConfig.ChainId,
		genesisConfig, genesisBlock, isFirst,
		blockStoreDB, blockStore, pm.mgrConfig, levelLog)
}

// LoadCAFromKV 扫描 加锁操作，从kv中扫描出CA列表,
// 如果内存中存在，从内存中返回; 否则，将数据加载到内存中（重新覆盖）
func (pm *ProcessorMgr) LoadCAFromKV() ([][]byte, error) {
	pm.registerLock.Lock()
	defer pm.registerLock.Unlock()
	if len(pm.caArrays) > 0 {
		return pm.caArrays, nil
	}
	listIter, listIterError := pm.systemDB.NewIteratorWithPrefix([]byte(
		keyCAPrefix))
	if listIterError != nil {
		return nil, listIterError
	}
	defer listIter.Release()
	var retCAList [][]byte
	for listIter.Next() {
		tempCA := listIter.Value()
		if len(tempCA) > 0 {
			retCAList = append(retCAList, tempCA)
		} else {
			// 不再继续扫描
			break
		}
	}
	pm.caArrays = retCAList
	return retCAList, nil
}

// SaveCAInKV 加锁操作，将CA保存到kv
func (pm *ProcessorMgr) SaveCAInKV(caBytes []byte) error {
	pm.registerLock.Lock()
	defer pm.registerLock.Unlock()
	caKeyIndex := len(pm.caArrays) + 1
	cakey := keyCAPrefix + strconv.Itoa(caKeyIndex)
	// 保存到kv
	saveError := pm.systemDB.Put([]byte(cakey), caBytes)
	if saveError != nil {
		return saveError
	}
	// 将ca加到内存中
	pm.caArrays = append(pm.caArrays, caBytes)

	return nil
}

// BatchSaveCAInKV 加锁操作，批量写入ca信息
func (pm *ProcessorMgr) BatchSaveCAInKV(caArrays [][]byte) error {
	if len(caArrays) == 0 {
		return nil
	}
	pm.registerLock.Lock()
	defer pm.registerLock.Unlock()
	beginIndex := len(pm.caArrays) + 1
	batcher := archive_utils.NewUpdateBatch()
	for i := 0; i < len(caArrays); i++ {
		cakey := keyCAPrefix + strconv.Itoa(beginIndex+i)
		batcher.Put([]byte(cakey), caArrays[i])
	}
	dbSaveError := pm.systemDB.WriteBatch(batcher, true)
	if dbSaveError != nil {
		return dbSaveError
	}
	pm.caArrays = append(pm.caArrays, caArrays...)
	return nil
}

// Close 关闭所有链的处理器
func (pm *ProcessorMgr) Close() {
	// archive_utils.GlobalServerLatch.Add(1)
	// defer archive_utils.GlobalServerLatch.Done()
	pm.systemDB.Close()
	pm.processorMp.Range(func(key, value interface{}) bool {
		tempProcess, _ := value.(*ChainProcessor)
		tempProcess.Close()
		genesisHash, _ := key.(string)
		pm.logger.Infof("chain %s has shutdown", genesisHash)
		return true
	})
}
