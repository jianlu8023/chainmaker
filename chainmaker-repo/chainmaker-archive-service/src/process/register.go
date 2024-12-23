/*
Copyright (C) BABEC. All rights reserved.


SPDX-License-Identifier: Apache-2.0
*/

// Package process define process logic
package process

import (
	"errors"
	"fmt"

	"chainmaker.org/chainmaker-archive-service/src/archive_utils"
	"chainmaker.org/chainmaker-archive-service/src/serverconf"
	archivePb "chainmaker.org/chainmaker/pb-go/v2/archivecenter"
	configPb "chainmaker.org/chainmaker/pb-go/v2/config"
	storePb "chainmaker.org/chainmaker/pb-go/v2/store"
	"github.com/gogo/protobuf/proto"
)

// RegisterChainByGenesisBlock 注册创世块到服务
// @receiver pm
// @param genesisBlock
// @return uint32
// @return error
func (pm *ProcessorMgr) RegisterChainByGenesisBlock(genesisBlock *storePb.BlockWithRWSet,
	globalLogCFG *serverconf.LogConfig) (uint32, error) {
	if genesisBlock == nil || genesisBlock.Block == nil {
		return uint32(archive_utils.CodeRegisterBlockNil), errors.New(archive_utils.MsgRegisterBlockNil)
	}
	block := genesisBlock.Block
	// 1. 从配置区块中计算出配置信息
	genesisConfig, genesisError := archive_utils.RestoreChainConfigFromBlock(block)
	if genesisError != nil {
		pm.logger.Errorf("RegisterChainByGenesisBlock genesis %+v  get config error %s ", *genesisBlock, genesisError.Error())
		return uint32(archive_utils.CodeRegisterNoConfig), genesisError
	}
	// 2. 根据配置区块校验一下block是否正确，
	verifyOk, _ := archive_utils.VerifyBlockHash(genesisConfig.Crypto.GetHash(), block)
	if !verifyOk {
		return uint32(archive_utils.CodeBlockHashVerifyFail), archive_utils.ErrorBlockHashVerifyFailed
	}
	// 3. 锁检测 ，
	existStatus, loaded := pm.genesisChainRegisteryMp.LoadOrStore(block.GetBlockHashStr(), registerInProcess)
	if loaded {
		status, _ := existStatus.(int8)
		if status == registerInProcess {
			// 有同一条链的别的节点正在注册
			return uint32(archivePb.RegisterStatus_RegisterStatusConflict), errors.New(archive_utils.MsgRegisterConflict)
		}
		// 已经注册过了，可以做交互了
		return uint32(archivePb.RegisterStatus_RegisterStatusSuccess), nil
	}
	// 4. marshal 保存到kv
	dbSaveError := pm.saveGenesisInfoInDB(block.GetBlockHashStr(), genesisConfig, genesisBlock)
	if dbSaveError != nil {
		// 锁释放了
		pm.genesisChainRegisteryMp.Delete(block.GetBlockHashStr())
		pm.logger.Errorf("RegisterChainByGenesisBlock save genesis config %+v ,block %+v , error %s ",
			*genesisConfig, *genesisBlock, dbSaveError.Error())
		return uint32(archive_utils.CodeSaveRegisterInfoToDBFail), archive_utils.ErrorSaveRegisterInfoToDBFail
	}
	// 5
	pm.addChainInfo(block.GetBlockHashStr(), &GenesisInfo{
		ChainId:       genesisConfig.ChainId,
		GenesisConfig: genesisConfig,
		GenesisBlock:  block,
	})
	// 6. 初始化链处理
	processor, processorError := pm.constructChainProcessor(block.GetBlockHashStr(), genesisConfig.ChainId,
		globalLogCFG, genesisConfig, genesisBlock, true)
	if processorError != nil {
		pm.logger.Errorf("RegisterChainByGenesisBlock new chain(chain-id: %s) processor error %s ",
			genesisConfig.ChainId, processorError.Error())
		panic(fmt.Sprintf("new chain(chain-id: %s) processor error(%s)",
			genesisConfig.ChainId, processorError.Error()))
	}
	pm.processorMp.Store(block.GetBlockHashStr(), processor)
	// 7. 更新一下注册信息
	pm.genesisChainRegisteryMp.Store(block.GetBlockHashStr(), registerProcessed)
	return uint32(archivePb.RegisterStatus_RegisterStatusSuccess), nil
}

// saveGenesisInfoInDB 保存创世区块信息到leveldb
// @receiver pm
// @param genesisHash
// @param genesisConfig
// @param genesisBlock
// @return error
func (pm *ProcessorMgr) saveGenesisInfoInDB(genesisHash string,
	genesisConfig *configPb.ChainConfig, genesisBlock *storePb.BlockWithRWSet) error {
	updateBatcher := archive_utils.NewUpdateBatch()
	blockBytes, blockErr := proto.Marshal(genesisBlock)
	configBytes, configErr := proto.Marshal(genesisConfig)
	if blockErr != nil {
		return blockErr
	}
	if configErr != nil {
		return configErr
	}
	// 1. genesishash -> chainid
	updateBatcher.Put([]byte(keyMapGenesisHashToChainIdPrefix+genesisHash), []byte(genesisConfig.ChainId))
	// 2 genesishash -> config
	updateBatcher.Put([]byte(keyMapGenesisHashToConfigPrefix+genesisHash), configBytes)
	// 3. genesishash -> block
	updateBatcher.Put([]byte(keyMapGenesisHashToGenesisBlockPrefix+genesisHash), blockBytes)
	return pm.systemDB.WriteBatch(updateBatcher, true)
}
