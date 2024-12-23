/*
Copyright (C) BABEC. All rights reserved.


SPDX-License-Identifier: Apache-2.0
*/

// Package process define process logic
package process

import (
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"chainmaker.org/chainmaker-archive-service/src/archive_utils"
	"chainmaker.org/chainmaker-archive-service/src/logger"
	"chainmaker.org/chainmaker-archive-service/src/serverconf"
	"chainmaker.org/chainmaker-archive-service/src/store/filestore"
	configPb "chainmaker.org/chainmaker/pb-go/v2/config"
	storePb "chainmaker.org/chainmaker/pb-go/v2/store"
	"github.com/gogo/protobuf/proto"
)

// ChainProcessor 链处理器
type ChainProcessor struct {
	blockDB          *filestore.BlockFileDB
	blockFile        *filestore.BlockFile
	chainGenesisHash string // 这个可以由mgr初始化
	chainId          string // 这个也可以由mgr初始化
	InArchive        int32
	ArchivedHeight   uint64 // 这个可以调用blockDB。GetLastSavepoint
	protoBufferPool  sync.Pool
	// 从genesis block中解析出来的chainconfig，这个得落盘（todo）
	genesisChainConfig *configPb.ChainConfig
	// 在接收到归档请求的时候才有用，实时计算，
	// 初始值为lastconfig,后续传过来的区块有配置则更新
	currentChainConfig *configPb.ChainConfig
	// 这个调用blockDB。GetLastConfig,取出高度，根据高度获得区块获得配置
	// ================================
	inCompress          uint32 // 1 正在压缩，0 表示未在压缩
	logger              *logger.WrappedLogger
	scanIntervalSeconds int64 // 扫描间隔
	retainSeconds       int64 // 保留文件时间
}

// NewChainProcessor 构造对象，从kv中构造currentChainConfig，Archivedheight，
func NewChainProcessor(genesisHash, chainId string,
	genesisConfig *configPb.ChainConfig,
	genesisBlock *storePb.BlockWithRWSet, isFirst bool,
	blockdb *filestore.BlockFileDB, blockFile *filestore.BlockFile,
	storageCFG *serverconf.StorageConfig, log *logger.WrappedLogger) (*ChainProcessor, error) {
	retProcessor := &ChainProcessor{
		chainGenesisHash:    genesisHash,
		chainId:             chainId,
		genesisChainConfig:  genesisConfig,
		blockDB:             blockdb, //
		blockFile:           blockFile,
		logger:              log,
		scanIntervalSeconds: int64(storageCFG.ScanIntervalSeconds),
		retainSeconds:       int64(storageCFG.RetainSeconds),
	}
	retProcessor.protoBufferPool.New = func() interface{} {
		return proto.NewBuffer(nil)
	}
	// 调用一下复原函数
	recoverErr := retProcessor.Recover()
	if recoverErr != nil {
		retProcessor.logger.Errorf("chain genesishash %s , chainid %s recover got error %s",
			genesisHash, chainId, recoverErr.Error())
		panic(fmt.Sprintf("chain genesishash %s , chainid %s recover got error %s",
			genesisHash, chainId, recoverErr.Error()))
	}
	var archivedHeight uint64
	currentCFG := genesisConfig
	if isFirst {
		//  new db,new blockfile, append genesisBlock
		_ = retProcessor.appendBlock(genesisBlock)
	} else {
		// ,查一下 归档高度，查一下最近的配置区块
		tempConfigBlock, tempError := retProcessor.GetLastConfigBlock()
		if tempError != nil {
			return nil, tempError
		}
		lastCFG, lastError := archive_utils.RestoreChainConfigFromBlock(tempConfigBlock)
		if lastError != nil {
			return nil, lastError
		}
		currentCFG = lastCFG
		//  查一下最近的归档高度
		saveHeight, saveError := retProcessor.GetLastSavepoint()
		if saveError != nil {
			return nil, saveError
		}
		archivedHeight = saveHeight
	}
	atomic.StoreUint64(&retProcessor.ArchivedHeight, archivedHeight)
	retProcessor.currentChainConfig = currentCFG
	go retProcessor.scanAndPruneCompressedFiles()
	go retProcessor.scanAndPruneDecompressFiles()
	return retProcessor, nil
}

// scanAndPruneCompressedFiles 扫描已经压缩过的，且未用到的文件
func (cp *ChainProcessor) scanAndPruneCompressedFiles() {
	ticker := time.NewTicker(time.Second * time.Duration(cp.scanIntervalSeconds))
	cp.logger.Infof("scanAndPruneCompressedFiles begin")
	for range ticker.C {
		cp.blockDB.ReapFiles(filestore.CompressKVPrefix, false, cp.retainSeconds)

	}
}

// scanAndPruneDecompressFiles 扫描已经解压缩的，且未用到的文件
func (cp *ChainProcessor) scanAndPruneDecompressFiles() {
	ticker := time.NewTicker(time.Second * time.Duration(cp.scanIntervalSeconds))
	cp.logger.Infof("scanAndPruneDecompressFiles begin ")
	for range ticker.C {
		cp.blockDB.ReapFiles(filestore.DecompressKVPrefix, true, cp.retainSeconds)
	}
}

// appendBlock 添加区块block
func (cp *ChainProcessor) appendBlock(block *storePb.BlockWithRWSet) error {
	buf, ok := cp.protoBufferPool.Get().(*proto.Buffer)
	if !ok {
		return archive_utils.ErrorGetBufPool
	}
	buf.Reset()
	blockBytes, blockWithSerializedInfo, err := filestore.SerializeBlock(block)
	if err != nil {
		return err
	}
	//1. 写入区块数据到文件
	blockIndex, start, end, needRecordDb, err := cp.writeBlockToFile(block.Block.Header.BlockHeight, blockBytes)
	blockWithSerializedInfo.Index = blockIndex
	cp.protoBufferPool.Put(buf)
	if err != nil {
		return err
	}
	// wrap一下
	wrapData := &filestore.StartEndWrapBlock{
		BlockSerializedInfo: blockWithSerializedInfo,
		StartHeight:         start,
		EndHeight:           end,
		NeedRecord:          needRecordDb,
	}
	// 2. 写入Db，todo 写一半断电
	cacheError := cp.writeDb(wrapData)
	if cacheError != nil {
		return cacheError
	}
	return nil
}

// writeBlockToFile 将数据写入到文件中
func (cp *ChainProcessor) writeBlockToFile(blockHeight uint64,
	blockBytes []byte) (*storePb.StoreInfo, uint64, uint64, bool, error) {
	fileName, offset, bytesLen, start, end, needRecordDb, err := cp.blockFile.Write(blockHeight+1, blockBytes)
	if err != nil {
		return nil, 0, 0, false, err
	}
	return &storePb.StoreInfo{
		FileName: fileName,
		Offset:   offset,
		ByteLen:  bytesLen,
	}, start, end, needRecordDb, nil
}

// writeDb 索引数据写入到kv
func (cp *ChainProcessor) writeDb(blockWithSerializedInfo *filestore.StartEndWrapBlock) error {
	cacheErr := cp.blockDB.CommitBlock(blockWithSerializedInfo, true)
	if cacheErr != nil {
		return cacheErr
	}
	return cp.blockDB.CommitBlock(blockWithSerializedInfo, false)
}

// Close 关闭链归档处理
func (cp *ChainProcessor) Close() {
	cp.blockDB.Close()
	cp.blockFile.Close()

}
