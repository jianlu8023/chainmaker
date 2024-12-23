/*
Copyright (C) BABEC. All rights reserved.


SPDX-License-Identifier: Apache-2.0
*/

// Package filestore define file operation
package filestore

import (
	"encoding/json"
	"errors"
	"math"
	"strconv"
	"sync"
	"time"

	"chainmaker.org/chainmaker-archive-service/src/archive_utils"
	"chainmaker.org/chainmaker-archive-service/src/interfaces"
	bytesconv "chainmaker.org/chainmaker/common/v2/bytehelper"
	commonPb "chainmaker.org/chainmaker/pb-go/v2/common"
	storePb "chainmaker.org/chainmaker/pb-go/v2/store"
	"chainmaker.org/chainmaker/protocol/v2"
	"chainmaker.org/chainmaker/utils/v2"
	"github.com/gogo/protobuf/proto"
)

const (
	// blockNumIdxKeyPrefix  = 'n'
	blockHashIdxKeyPrefix = 'h'

	blockTxIDIdxKeyPrefix   = 'b'
	blockIndexKeyPrefix     = "ib"
	blockMetaIndexKeyPrefix = "im"
	lastBlockNumKeyStr      = "lastBlockNumKey"
	lastConfigBlockNumKey   = "lastConfigBlockNumKey"
	// archivedPivotKey        = "archivedPivotKey"
	txRWSetIndexKeyPrefix = "ri"
	// resultDBSavepointKey    = "resultSavepointKey"

	startEndMapPrefix = "sem"
)

var (
	errGetBatchPool = errors.New("get updatebatch error")
)

// BlockFileDB provider a implementation of `blockdb.BlockDB`
// This implementation provides a key-value based data model
type BlockFileDB struct {
	dbHandle protocol.DBHandle
	cache    *StoreCacheMgr
	//archivedPivot uint64
	logger protocol.Logger
	sync.Mutex
	batchPool sync.Pool
	// storeConfig *serverconf.StorageConfig
	fileStore interfaces.BinLogger
	// 压缩
	//compressMgr *CompressMgr
}

// NewBlockFileDB 创建数据库操作对象
// @param genesisHash
// @param dbHandle
// @param logger
// @param fileStore
// @return BlockFileDB
func NewBlockFileDB(genesisHash string, dbHandle protocol.DBHandle,
	logger protocol.Logger, fileStore interfaces.BinLogger) *BlockFileDB {
	b := &BlockFileDB{
		dbHandle: dbHandle,
		cache:    NewStoreCacheMgr(genesisHash, 10, logger),
		//archivedPivot: 0,
		logger:    logger,
		batchPool: sync.Pool{},
		// storeConfig: storeConfig,
		fileStore: fileStore,
	}
	b.batchPool.New = func() interface{} {
		return archive_utils.NewUpdateBatch()
	}
	return b
}

// InitGenesis 初始化创世区块
// @receiver b
// @param genesisBlock
// @return error
func (b *BlockFileDB) InitGenesis(genesisBlock *BlockWithSerializedInfo) error {
	wrapBlock := &StartEndWrapBlock{
		BlockSerializedInfo: genesisBlock,
	}
	//update Cache
	err := b.CommitBlock(wrapBlock, true)
	if err != nil {
		return err
	}
	//update BlockFileDB
	return b.CommitBlock(wrapBlock, false)
}

// CommitBlock commits the block and the corresponding rwsets in an atomic operation
// @receiver b
// @param blockInfo
// @param isCache
// @return error
func (b *BlockFileDB) CommitBlock(blockInfo *StartEndWrapBlock, isCache bool) error {
	//原则上，写db失败，重试一定次数后，仍然失败，panic

	//如果是更新cache，则 直接更新 然后返回，不写 blockdb
	if isCache {
		return b.CommitCache(blockInfo)
	}
	//如果不是更新cache，则 直接更新 kvdb，然后返回
	return b.CommitDB(blockInfo)
}

// CommitCache 提交数据到cache
// @receiver b
// @param wrapBlock
// @return error
func (b *BlockFileDB) CommitCache(wrapBlock *StartEndWrapBlock) error {
	//如果是更新cache，则 直接更新 然后返回，不写 blockdb
	//从对象池中取一个对象,并重置
	blockInfo := wrapBlock.BlockSerializedInfo
	start := time.Now()
	batch, ok := b.batchPool.Get().(*archive_utils.UpdateBatch)
	if !ok {
		b.logger.Errorf("chain[%s]: blockInfo[%d] get updatebatch error",
			blockInfo.Block.Header.ChainId, blockInfo.Block.Header.BlockHeight)
		return errGetBatchPool
	}
	batch.ReSet()
	dbType := b.dbHandle.GetDbType()

	// 0. 保存一下开始文件-结束文件映射
	if wrapBlock.NeedRecord {
		startKey := constructBeginEndMapKey(wrapBlock.StartHeight)
		batch.Put(startKey, encodeBlockNum(wrapBlock.EndHeight))
	}

	// 1. last blockInfo height
	block := blockInfo.Block
	batch.Put([]byte(lastBlockNumKeyStr), encodeBlockNum(block.Header.BlockHeight))

	// 2. height-> blockInfo
	if blockInfo.Index == nil || blockInfo.MetaIndex == nil {
		return errors.New("blockInfo.Index and blockInfo.MetaIndex must not nil while block rfile db enabled")

	}
	blockIndexKey := constructBlockIndexKey(dbType, block.Header.BlockHeight)
	blockIndexInfo, err := proto.Marshal(blockInfo.Index)
	if err != nil {
		return err
	}
	batch.Put(blockIndexKey, blockIndexInfo)
	b.logger.Debugf("put block[%d] rfile index:%v", block.Header.BlockHeight, blockInfo.Index)

	// 3. save block meta index to db
	metaIndexKey := constructBlockMetaIndexKey(dbType, block.Header.BlockHeight)
	metaIndexInfo := ConstructDBIndexInfo(blockInfo.Index, blockInfo.MetaIndex.Offset,
		blockInfo.MetaIndex.ByteLen)
	batch.Put(metaIndexKey, metaIndexInfo)

	// 4. hash-> height
	hashKey := constructBlockHashKey(b.dbHandle.GetDbType(), block.Header.BlockHash)
	batch.Put(hashKey, encodeBlockNum(block.Header.BlockHeight))

	// 4. txid -> tx,  txid -> blockHeight
	// 5. Concurrency batch Put
	//并发更新cache,ConcurrencyMap,提升并发更新能力
	//groups := len(blockInfo.SerializedContractEvents)
	wg := &sync.WaitGroup{}
	wg.Add(len(blockInfo.SerializedTxs))

	for index, txBytes := range blockInfo.SerializedTxs {
		go func(index int, txBytes []byte, batch protocol.StoreBatcher, wg *sync.WaitGroup) {
			defer wg.Done()
			tx := blockInfo.Block.Txs[index]

			// if block rfile db disable, save tx data to db
			txFileIndex := blockInfo.TxsIndex[index]

			// 把tx的地址写入数据库
			blockTxIdKey := constructBlockTxIDKey(tx.Payload.TxId)
			txBlockInf := constructTxIDBlockInfo(block.Header.BlockHeight, block.Header.BlockHash, uint32(index),
				block.Header.BlockTimestamp, blockInfo.Index, txFileIndex)

			batch.Put(blockTxIdKey, txBlockInf)
			//b.logger.Debugf("put tx[%s] rfile index:%v", tx.Payload.TxId, txFileIndex)
		}(index, txBytes, batch, wg)
	}

	wg.Wait()

	// 这里写入TXRWSET的数据
	txRWSets := blockInfo.TxRWSets
	for index, txRWSet := range txRWSets {
		rwSetIndexKey := constructTxRWSetIndexKey(txRWSet.TxId)
		rwIndex := blockInfo.RWSetsIndex[index]
		rwIndexInfo := ConstructDBIndexInfo(blockInfo.Index, rwIndex.Offset, rwIndex.ByteLen)
		batch.Put(rwSetIndexKey, rwIndexInfo)
	}
	//  写入TXRWSET索引数据完毕

	// last configBlock height
	if utils.IsConfBlock(block) || block.Header.BlockHeight == 0 {
		batch.Put([]byte(lastConfigBlockNumKey), encodeBlockNum(block.Header.BlockHeight))
		b.logger.Infof("chain[%s]: commit config blockInfo[%d]", block.Header.ChainId, block.Header.BlockHeight)
	}

	// 6. 增加cache,注意这个batch放到cache中了，正在使用，不能放到batchPool中
	b.cache.AddBlock(block.Header.BlockHeight, batch)

	b.logger.Debugf("chain[%s]: commit cache block[%d] blockfiledb, batch[%d], time used: %d",
		block.Header.ChainId, block.Header.BlockHeight, batch.Len(),
		time.Since(start).Milliseconds())
	return nil
}

// CommitDB 提交数据到kvdb
// @receiver b
// @param blockInfo
// @return error
func (b *BlockFileDB) CommitDB(blockInfo *StartEndWrapBlock) error {
	//1. 从缓存中取 batch
	start := time.Now()
	block := blockInfo.BlockSerializedInfo.Block
	cacheBatch, err := b.cache.GetBatch(block.Header.BlockHeight)
	if err != nil {
		b.logger.Errorf("chain[%s]: commit blockInfo[%d] delete cacheBatch error",
			block.Header.ChainId, block.Header.BlockHeight)
		panic(err)
	}

	batchDur := time.Since(start)
	//2. write blockDb
	err = b.writeBatch(block.Header.BlockHeight, cacheBatch)
	if err != nil {
		return err
	}

	//3. Delete block from Cache, put batch to batchPool
	//把cacheBatch 从Cache 中删除
	b.cache.DelBlock(block.Header.BlockHeight)

	//再把cacheBatch放回到batchPool 中 (注意这两步前后不能反了)
	b.batchPool.Put(cacheBatch)
	writeDur := time.Since(start)

	b.logger.Debugf("chain[%s]: commit block[%d] kv blockfiledb, time used (batch[%d]:%d, "+
		"write:%d, total:%d)", block.Header.ChainId, block.Header.BlockHeight, cacheBatch.Len(),
		batchDur.Milliseconds(), (writeDur - batchDur).Milliseconds(), time.Since(start).Milliseconds())
	return nil
}

// ShrinkBlocks remove ranged txid--SerializedTx from kvdb
// 不支持
func (b *BlockFileDB) ShrinkBlocks(startHeight uint64, endHeight uint64) (map[uint64][]string, error) {
	panic("block filedb not implement shrink")
}

// RestoreBlocks restore block data from outside to kvdb: txid--SerializedTx
// 不支持
func (b *BlockFileDB) RestoreBlocks(blockInfos []*BlockWithSerializedInfo) error {
	panic("block filedb not implement restore blocks")
}

// BlockExists returns true if the block hash exist, or returns false if none exists.
// @receiver b
// @param blockHash
// @return bool
// @return error
func (b *BlockFileDB) BlockExists(blockHash []byte) (bool, error) {
	hashKey := constructBlockHashKey(b.dbHandle.GetDbType(), blockHash)
	return b.has(hashKey)
}

// GetBlockByHash returns a block given it's hash, or returns nil if none exists.
// @receiver b
// @param blockHash
// @return Block
// @return error
func (b *BlockFileDB) GetBlockByHash(blockHash []byte) (*commonPb.Block, error) {
	hashKey := constructBlockHashKey(b.dbHandle.GetDbType(), blockHash)
	heightBytes, err := b.get(hashKey)
	if err != nil {
		return nil, err
	}
	if len(heightBytes) == 0 {
		return nil, nil
	}
	height := decodeBlockNum(heightBytes)
	return b.GetBlock(height)
}

// GetHeightByHash returns a block height given it's hash, or returns nil if none exists.
// @receiver b
// @param blockHash
// @return uint64
// @return error
func (b *BlockFileDB) GetHeightByHash(blockHash []byte) (uint64, error) {
	hashKey := constructBlockHashKey(b.dbHandle.GetDbType(), blockHash)
	heightBytes, err := b.get(hashKey)
	if err != nil {
		return 0, err
	}

	if heightBytes == nil {
		return 0, archive_utils.ErrValueNotFound
	}

	return decodeBlockNum(heightBytes), nil
}

// GetBlockHeaderByHeight returns a block header by given it's height, or returns nil if none exists.
// @receiver b
// @param height
// @return BlockHeader
// @return error
func (b *BlockFileDB) GetBlockHeaderByHeight(height uint64) (*commonPb.BlockHeader, error) {
	storeInfo, err := b.GetBlockMetaIndex(height)
	if err != nil {
		return nil, err
	}
	if storeInfo == nil {
		return nil, nil
	}
	// 根据storeinfo中的文件名称解析出区块的起始点，比较一下压缩点
	_, decompressed, decompressedError := b.FindOrDecompressStoreInfo(storeInfo)
	if decompressedError != nil {
		return nil, decompressedError
	}
	vBytes, err := b.fileStore.ReadFileSection(decompressed, storeInfo)
	if err != nil {
		return nil, err
	}
	var blockStoreInfo storePb.SerializedBlock
	err = proto.Unmarshal(vBytes, &blockStoreInfo)
	if err != nil {
		return nil, err
	}

	return blockStoreInfo.Header, nil
}

// GetBlock returns a block given it's block height, or returns nil if none exists.
// @receiver b
// @param height
// @return Block
// @return error
func (b *BlockFileDB) GetBlock(height uint64) (*commonPb.Block, error) {

	blockWithRWSet, err := b.GetBlockWithRWSetByHeight(height)
	if err != nil {
		return nil, err
	}
	if blockWithRWSet == nil {
		return nil, nil
	}
	return blockWithRWSet.Block, nil
}

// GetBlockWithRWSetByHeight 根据区块高度查询带有读写集的区块数据
// @receiver b
// @param height
// @return BlockWithRWSet
// @return error
func (b *BlockFileDB) GetBlockWithRWSetByHeight(height uint64) (*storePb.BlockWithRWSet, error) {
	// if block rfile db is enable and db is kvdb, we will get data from block rfile db
	index, err := b.GetBlockIndex(height)
	if err != nil {
		return nil, err
	}
	if index == nil {
		return nil, nil
	}
	// 根据storeinfo中的文件名称解析出区块的起始点，比较一下压缩点
	_, decompressed, decompressedError := b.FindOrDecompressStoreInfo(index)
	if decompressedError != nil {
		return nil, decompressedError
	}
	data, err := b.fileStore.ReadFileSection(decompressed, index)
	if err != nil {
		return nil, err
	}
	brw, err := DeserializeBlock(data)
	if err != nil {
		return nil, err
	}
	return brw, nil
}

// GetLastBlock returns the last block.
// @receiver b
// @return Block
// @return error
func (b *BlockFileDB) GetLastBlock() (*commonPb.Block, error) {
	num, err := b.GetLastSavepoint()
	b.logger.Debugf("GetLastBlock %d ", num)
	if err != nil {
		return nil, err
	}
	return b.GetBlock(num)
}

// GetLastConfigBlock returns the last config block.
// @receiver b
// @return Block
// @return error
func (b *BlockFileDB) GetLastConfigBlock() (*commonPb.Block, error) {
	height, err := b.GetLastConfigBlockHeight()
	if err != nil {
		return nil, err
	}
	b.logger.Debugf("configBlock height:%v", height)
	return b.GetBlock(height)
}

// GetLastConfigBlockHeight returns the last config block height.
// @receiver b
// @return uint64
// @return error
func (b *BlockFileDB) GetLastConfigBlockHeight() (uint64, error) {
	heightBytes, err := b.get([]byte(lastConfigBlockNumKey))
	if err != nil {
		return math.MaxUint64, err
	}
	b.logger.Debugf("configBlock height:%v", heightBytes)
	return decodeBlockNum(heightBytes), nil
}

// GetFilteredBlock returns a filtered block given it's block height, or return nil if none exists.
// @receiver b
// @param height
// @return SerializedBlock
// @return error
func (b *BlockFileDB) GetFilteredBlock(height uint64) (*storePb.SerializedBlock, error) {
	storeInfo, err := b.GetBlockMetaIndex(height)
	if err != nil {
		return nil, err
	}
	if storeInfo == nil {
		return nil, nil
	}
	// 根据storeinfo中的文件名称解析出区块的起始点，比较一下压缩点
	_, decompressed, decompressedError := b.FindOrDecompressStoreInfo(storeInfo)
	if decompressedError != nil {
		return nil, decompressedError
	}

	vBytes, err := b.fileStore.ReadFileSection(decompressed, storeInfo)
	if err != nil {
		return nil, err
	}
	var blockStoreInfo storePb.SerializedBlock
	err = proto.Unmarshal(vBytes, &blockStoreInfo)
	if err != nil {
		return nil, err
	}
	return &blockStoreInfo, nil

}

// GetLastSavepoint reurns the last block height
// @receiver b
// @return uint64
// @return error
func (b *BlockFileDB) GetLastSavepoint() (uint64, error) {
	bytes, err := b.get([]byte(lastBlockNumKeyStr))
	if err != nil {
		return 0, err
	} else if bytes == nil {
		return 0, nil
	}

	return decodeBlockNum(bytes), nil
}

// GetBlockByTx returns a block which contains a tx.
// @receiver b
// @param txId
// @return Block
// @return error
func (b *BlockFileDB) GetBlockByTx(txId string) (*commonPb.Block, error) {
	txInfo, err := b.getTxInfoOnly(txId)
	if err != nil {
		return nil, err
	}
	if txInfo == nil {
		return nil, nil
	}
	return b.GetBlock(txInfo.BlockHeight)
}

// GetTxHeight retrieves a transaction height by txid, or returns nil if none exists.
// @receiver b
// @param txId
// @return uint64
// @return error
func (b *BlockFileDB) GetTxHeight(txId string) (uint64, error) {
	blockTxIdKey := constructBlockTxIDKey(txId)
	txIdBlockInfoBytes, err := b.get(blockTxIdKey)
	if err != nil {
		return 0, err
	}

	if txIdBlockInfoBytes == nil {
		return 0, archive_utils.ErrorBlockByTxId
	}
	height, _, _, _, _, err := parseTxIdBlockInfo(txIdBlockInfoBytes)
	return height, err
}

// GetTx retrieves a transaction by txid, or returns nil if none exists.
// @receiver b
// @param txId
// @return Transaction
// @return error
func (b *BlockFileDB) GetTx(txId string) (*commonPb.Transaction, error) {

	// get tx from block rfile db
	index, err := b.GetTxIndex(txId)
	if err != nil {
		return nil, err
	}
	if index == nil {
		return nil, nil
	}
	// 根据storeinfo中的文件名称解析出区块的起始点，比较一下压缩点
	_, decompressed, decompressedError := b.FindOrDecompressStoreInfo(index)
	if decompressedError != nil {
		return nil, decompressedError
	}
	data, err := b.fileStore.ReadFileSection(decompressed, index)
	if err != nil {
		return nil, err
	}

	tx := &commonPb.Transaction{}
	if err = proto.Unmarshal(data, tx); err != nil {
		return nil, err
	}
	return tx, nil
}

// GetTxWithBlockInfo 根据txid获取交易数据
// @receiver b
// @param txId
// @return TransactionStoreInfo
// @return error
func (b *BlockFileDB) GetTxWithBlockInfo(txId string) (*storePb.TransactionStoreInfo, error) {
	// 先查一下索引
	txInfo, err := b.getTxInfoOnly(txId)
	if err != nil {
		return nil, err
	}
	if txInfo == nil { //查不到对应的Tx，返回nil,nil
		return nil, nil
	}
	// 根据txid查询交易数据
	tx, err := b.GetTx(txId)
	if err != nil {
		return nil, err
	}
	txInfo.Transaction = tx
	return txInfo, nil
}

// getTxInfoOnly 获得除Tx之外的其他TxInfo信息
// @receiver b
// @param txId
// @return TransactionStoreInfo
// @return error
func (b *BlockFileDB) getTxInfoOnly(txId string) (*storePb.TransactionStoreInfo, error) {
	txIDBlockInfoBytes, err := b.get(constructBlockTxIDKey(txId))
	if err != nil {
		return nil, err
	}
	if txIDBlockInfoBytes == nil {
		return nil, nil
	}
	txi := &storePb.TransactionStoreInfo{}
	err = txi.Unmarshal(txIDBlockInfoBytes)
	return txi, err
}

// GetTxInfoOnly 获得除Tx之外的其他TxInfo信息
// @receiver b
// @param txId
// @return TransactionStoreInfo
// @return error
func (b *BlockFileDB) GetTxInfoOnly(txId string) (*storePb.TransactionStoreInfo, error) {
	return b.getTxInfoOnly(txId)
}

// TxExists returns true if the tx exist, or returns false if none exists.
// @receiver b
// @param txId
// @return bool
// @return error
func (b *BlockFileDB) TxExists(txId string) (bool, error) {
	txHashKey := constructBlockTxIDKey(txId)
	exist, err := b.has(txHashKey)
	if err != nil {
		return false, err
	}
	return exist, nil
}

// GetTxConfirmedTime returns the confirmed time of a given tx
// @receiver b
// @param txId
// @return int64
// @return error
func (b *BlockFileDB) GetTxConfirmedTime(txId string) (int64, error) {
	txInfo, err := b.getTxInfoOnly(txId)
	if err != nil {
		return 0, err
	}
	if txInfo == nil {
		return 0, nil
	}
	//如果是新版本，有BlockTimestamp。直接返回
	if txInfo.BlockTimestamp > 0 {
		return txInfo.BlockTimestamp, nil
	}
	//从TxInfo拿不到Timestamp，那么就从Block去拿
	block, err := b.GetBlockMeta(txInfo.BlockHeight)
	if err != nil {
		return 0, err
	}
	return block.Header.BlockTimestamp, nil
}

// GetBlockIndex reurns the offset of the block in the rfile
// @receiver b
// @param height
// @return StoreInfo
// @return error
func (b *BlockFileDB) GetBlockIndex(height uint64) (*storePb.StoreInfo, error) {
	blockIndexByte, err := b.get(constructBlockIndexKey(b.dbHandle.GetDbType(), height))
	if err != nil {
		return nil, err
	}
	if blockIndexByte == nil {
		return nil, nil
	}
	return DecodeValueToIndex(blockIndexByte)
}

// GetBlockMeta 根据高度获取区块元信息
// @receiver b
// @param height
// @return SerializedBlock
// @return error
func (b *BlockFileDB) GetBlockMeta(height uint64) (*storePb.SerializedBlock, error) {
	si, err := b.GetBlockMetaIndex(height)
	if err != nil {
		return nil, err
	}
	if si == nil {
		return nil, nil
	}

	// 根据storeinfo中的文件名称解析出区块的起始点，比较一下压缩点
	_, decompressed, decompressedError := b.FindOrDecompressStoreInfo(si)
	if decompressedError != nil {
		return nil, decompressedError
	}
	data, err := b.fileStore.ReadFileSection(decompressed, si)
	if err != nil {
		return nil, err
	}
	bm := &storePb.SerializedBlock{}
	err = bm.Unmarshal(data)
	if err != nil {
		return nil, err
	}
	return bm, nil
}

// GetBlockMetaIndex returns the offset of the block in the rfile
// @receiver b
// @param height
// @param StoreInfo
// @return error
func (b *BlockFileDB) GetBlockMetaIndex(height uint64) (*storePb.StoreInfo, error) {
	index, err := b.get(constructBlockMetaIndexKey(b.dbHandle.GetDbType(), height))
	if err != nil {
		return nil, err
	}
	if index == nil {
		return nil, nil
	}
	return DecodeValueToIndex(index)
}

// GetTxIndex returns the offset of the transaction in the rfile
// @receiver b
// @param txId
// @return StoreInfo
// @return error
func (b *BlockFileDB) GetTxIndex(txId string) (*storePb.StoreInfo, error) {
	txIDBlockInfoBytes, err := b.get(constructBlockTxIDKey(txId))
	if err != nil {
		return nil, err
	}
	if len(txIDBlockInfoBytes) == 0 {
		return nil, nil //not found
	}
	_, _, _, _, txIndex, err := parseTxIdBlockInfo(txIDBlockInfoBytes)
	b.logger.Debugf("read tx[%s] rfile index get:%v", txId, txIndex)
	return txIndex, err
}

// Close is used to close database
func (b *BlockFileDB) Close() {
	b.logger.Info("close block rfile db")
	b.dbHandle.Close()
	b.cache.Clear()
}

// writeBatch 调用底层kvdb将batch数据写入（batch为一个block的索引）
// @receiver b
// @param blockHeight
// @param batch
// @return error
func (b *BlockFileDB) writeBatch(blockHeight uint64, batch protocol.StoreBatcher) error {

	startWriteBatchTime := utils.CurrentTimeMillisSeconds()
	err := b.dbHandle.WriteBatch(batch, true)
	endWriteBatchTime := utils.CurrentTimeMillisSeconds()
	b.logger.Debugf("write block db, block[%d], current batch cnt: %d, time used: %d",
		blockHeight, batch.Len(), endWriteBatchTime-startWriteBatchTime)
	return err
}

// get 获取key
// @receiver b
// @param key
// @return []byte
// @return error
func (b *BlockFileDB) get(key []byte) ([]byte, error) {
	//get from cache
	value, exist := b.cache.Get(string(key))
	if exist {
		b.logger.Debugf("get content: [%x] by [%s] in cache", value, string(key))
		if len(value) == 0 {
			b.logger.Debugf("get value []byte is empty in cache, key[%s], value: %s, ", key, value)
		}
		return value, nil
	}
	//如果从db 中未找到，会返回 (val,err) 为 (nil,nil)
	//由调用get的上层函数再做判断
	//因为存储系统，对一个key/value 做删除操作，是 直接把 value改为 nil,不做物理删除
	//get from database
	val, err := b.dbHandle.Get(key)
	b.logger.Debugf("get content: [%x] by [%s] in database", val, string(key))
	if err != nil {
		if len(val) == 0 {
			b.logger.Debugf("get value []byte is empty in database, key[%s], value: %s, ", key, value)
		}
		//
	}
	if err == nil {
		if len(val) == 0 {
			b.logger.Debugf("get value []byte is empty in database,err == nil, key[%s], value: %s, ", key, value)
		}
	}
	return val, err
}

// has 是否有key
// @receiver b
// @param key
// @return bool
// @return error
func (b *BlockFileDB) has(key []byte) (bool, error) {
	//check has from cache
	isDelete, exist := b.cache.Has(string(key))
	if exist {
		return !isDelete, nil
	}
	return b.dbHandle.Has(key)
}

// func constructBlockNumKey(dbType string, blockNum uint64) []byte {
// 	blkNumBytes := encodeBlockNum(blockNum)
// 	return append([]byte{blockNumIdxKeyPrefix}, blkNumBytes...)
// }

func constructBlockMetaIndexKey(dbType string, blockNum uint64) []byte {
	blkNumBytes := encodeBlockNum(blockNum)
	key := append([]byte{}, blockMetaIndexKeyPrefix...)
	return append(key, blkNumBytes...)
}

func constructBlockIndexKey(dbType string, blockNum uint64) []byte {
	blkNumBytes := encodeBlockNum(blockNum)
	key := append([]byte{}, blockIndexKeyPrefix...)
	return append(key, blkNumBytes...)
}

func constructBlockHashKey(dbType string, blockHash []byte) []byte {
	return append([]byte{blockHashIdxKeyPrefix}, blockHash...)
}

func constructBlockTxIDKey(txID string) []byte {
	return append([]byte{blockTxIDIdxKeyPrefix}, bytesconv.StringToBytes(txID)...)
}

func encodeBlockNum(blockNum uint64) []byte {
	return proto.EncodeVarint(blockNum)
}

// func decodeBlockNumKey(dbType string, blkNumBytes []byte) uint64 {
// 	blkNumBytes = blkNumBytes[len([]byte{blockNumIdxKeyPrefix}):]
// 	return decodeBlockNum(blkNumBytes)
// }

func decodeBlockNum(blockNumBytes []byte) uint64 {
	blockNum, _ := proto.DecodeVarint(blockNumBytes)
	return blockNum
}

func constructTxIDBlockInfo(height uint64, blockHash []byte, txIndex uint32, timestamp int64,
	blockFileIndex, txFileIndex *storePb.StoreInfo) []byte {
	var transactionFileIndex *storePb.StoreInfo
	if txFileIndex != nil {
		transactionFileIndex = &storePb.StoreInfo{
			FileName: blockFileIndex.FileName,
			Offset:   blockFileIndex.Offset + txFileIndex.Offset,
			ByteLen:  txFileIndex.ByteLen,
		}
	}
	txInf := &storePb.TransactionStoreInfo{
		BlockHeight:          height,
		BlockHash:            nil, //for performance, set nil
		TxIndex:              txIndex,
		BlockTimestamp:       timestamp,
		TransactionStoreInfo: transactionFileIndex,
	}
	data, _ := txInf.Marshal()
	return data
}

func parseTxIdBlockInfo(value []byte) (height uint64, blockHash []byte, txIndex uint32, timestamp int64,
	txFIleIndex *storePb.StoreInfo, err error) {
	if len(value) == 0 {
		err = errors.New("input is empty,in parseTxIdBlockInfo")
		return
	}
	//新版本使用TransactionInfo，是因为经过BenchmarkTest速度会快很多
	var txInfo storePb.TransactionStoreInfo
	err = txInfo.Unmarshal(value)
	if err != nil {
		return
	}
	height = txInfo.BlockHeight
	blockHash = txInfo.BlockHash
	txIndex = txInfo.TxIndex
	timestamp = txInfo.BlockTimestamp
	txFIleIndex = txInfo.TransactionStoreInfo
	err = nil
	return
}

func constructTxRWSetIndexKey(txId string) []byte {
	key := append([]byte{}, txRWSetIndexKeyPrefix...)
	return append(key, txId...)
}

func constructBeginEndMapKey(start uint64) []byte {
	key := append([]byte{}, []byte(startEndMapPrefix)...)
	return append(key, encodeBlockNum(start)...)
}

// GetTxRWSet returns an txRWSet for given txId, or returns nil if none exists.
// @receiver b
// @param txId
// @return TxRWSet
// @return error
func (b *BlockFileDB) GetTxRWSet(txId string) (*commonPb.TxRWSet, error) {
	sinfo, err := b.GetRWSetIndex(txId)
	if err != nil {
		return nil, err
	}
	if sinfo == nil {
		return nil, nil
	}

	// 根据storeinfo中的文件名称解析出区块的起始点，比较一下压缩点
	_, decompressed, decompressedError := b.FindOrDecompressStoreInfo(sinfo)
	if decompressedError != nil {
		return nil, decompressedError
	}
	data, err := b.fileStore.ReadFileSection(decompressed, sinfo)
	if err != nil {
		return nil, err
	}
	rwset := &commonPb.TxRWSet{}
	err = rwset.Unmarshal(data)
	if err != nil {
		return nil, err
	}
	return rwset, nil
}

// GetRWSetIndex returns the offset of the block in the file
// @receiver b
// @param txId
// @return StoreInfo
// @return error
func (b *BlockFileDB) GetRWSetIndex(txId string) (*storePb.StoreInfo, error) {
	// GetRWSetIndex returns the offset of the block in the file
	index, err := b.get(constructTxRWSetIndexKey(txId))
	if err != nil {
		return nil, err
	}

	return DecodeValueToIndex(index)
}

// FindOrDecompressStoreInfo 返回是否是解压缩文件
// @receiver b
// @param storeInfo
// @return storeInfo
// @return bool
// @return error
func (b *BlockFileDB) FindOrDecompressStoreInfo(storeInfo *storePb.StoreInfo) (*storePb.StoreInfo, bool, error) {
	// 这个地方最好用下缓存
	compressedHeight, compressedError := b.GetCompressedHeight()
	if compressedError != nil {
		return nil, false, compressedError
	}
	beginHeight, _ := strconv.ParseUint(storeInfo.FileName, 10, 64)
	// 说明未被压缩，直接返回
	if compressedHeight < beginHeight {
		return storeInfo, false, nil
	}
	// 先找一下是否已经解压过，解压过了直接返回，
	// 避免并发解压缩同一个文件多次
	b.Lock()
	defer b.Unlock()
	exist, _, existError := b.fileStore.CheckDecompressFileExist(storeInfo.FileName)
	if existError != nil {
		return nil, false, existError
	}
	if !exist {
		// 如果不存在解压的文件，那么把该文件解压一下
		_, deCompressError := b.fileStore.DeCompressFile(storeInfo.FileName)
		if deCompressError != nil {
			return nil, false, deCompressError
		}
		// 存到db中，
		tempInfo := DecompressFileInfo{FileName: storeInfo.FileName, CreatedAt: time.Now().Unix()}
		tempBytes, _ := json.Marshal(tempInfo)
		saveDecompressError := b.dbHandle.Put([]byte(DecompressKVPrefix+storeInfo.FileName), tempBytes)
		if saveDecompressError != nil {
			b.logger.Errorf("save file %s decompress info got error %s ", storeInfo.FileName, saveDecompressError.Error())
		}
	}
	return storeInfo, true, nil
}

// GetCompressedHeight 数据库中查询压缩区块高度
// @receiver b
// @return uint64
// @return error
func (b *BlockFileDB) GetCompressedHeight() (uint64, error) {
	heightBytes, err := b.dbHandle.Get([]byte(compressedHeightKey))
	if err != nil {
		return 0, err
	}
	if len(heightBytes) == 0 {
		return 0, nil //尚未保存过
	}
	compressedHeight := decodeBlockNum(heightBytes)
	return compressedHeight, nil
}

// SaveCompressedHeightIntoDB 保存压缩高度到kv中
// @receiver b
// @param height
// @return error
func (b *BlockFileDB) SaveCompressedHeightIntoDB(height uint64) error {
	heightBytes := encodeBlockNum(height)
	saveError := b.dbHandle.Put([]byte(compressedHeightKey), heightBytes)
	if saveError != nil {
		return saveError
	}
	return nil
}

// GetFileEndHeightByBeginHeight 根据起始高度查询文件的结束高度
// @receiver b
// @param beginHeight
// @return uint64
// @return error
func (b *BlockFileDB) GetFileEndHeightByBeginHeight(beginHeight uint64) (uint64, error) {
	startKey := constructBeginEndMapKey(beginHeight)
	value, valueError := b.dbHandle.Get(startKey)
	if valueError != nil {
		return 0, valueError
	}
	endHeight := decodeBlockNum(value)
	return endHeight, nil
}

// CompressUnderHeight 压缩指定高度
// @receiver b
// @param height
// @return uint64
// @return uint64
// @return error
func (b *BlockFileDB) CompressUnderHeight(height uint64) (uint64, uint64, error) {
	compressedHeight, verifyError := b.verifyCompressedHeight(height)
	if verifyError != nil {
		return 0, 0, verifyError
	}
	var startCompressHeight, endCompressHeight uint64
	startCompressHeight = compressedHeight + 1
	if compressedHeight == 0 {
		startCompressHeight = 0
	}
	// 获取当前可以压缩的最大高度,从文件中读取
	maxHeight := b.fileStore.GetCanCompressHeight()
	if height >= maxHeight {
		endCompressHeight = maxHeight
	} else {
		maxHeightEnd, maxHeightEndError := b.findMaxHeightUnderHeight(height, maxHeight)
		if maxHeightEndError != nil {
			return 0, 0, maxHeightEndError
		}
		endCompressHeight = maxHeightEnd
	}
	b.logger.Infof("CompressUnderHeight height %d , startCompressHeight %d , endCompressHeight %d",
		height, startCompressHeight, endCompressHeight)
	if endCompressHeight < startCompressHeight {
		// 说明不必压缩了
		return endCompressHeight, endCompressHeight, nil
	}
	// 从前面向后面开始压缩文件
	for currentHeight := startCompressHeight; currentHeight < endCompressHeight; {
		tempStart := currentHeight
		tempEnd, tempEndError := b.GetFileEndHeightByBeginHeight(tempStart)
		if tempEndError != nil {
			return 0, 0, tempEndError
		}
		// 压缩文件
		tempCompressName, compressError := b.fileStore.CompressFileByStartHeight(tempStart)
		if compressError != nil {
			return 0, 0, compressError
		}
		// 更新一下kv保存的数据
		saveDbError := b.SaveCompressedHeightIntoDB(tempEnd)
		if saveDbError != nil {
			// 已经压缩的文件是否需要删除一下？
			return 0, 0, saveDbError
		}
		// 保存一下压缩信息到kv中
		// 存到db中，
		tempInfo := DecompressFileInfo{FileName: tempCompressName, CreatedAt: time.Now().Unix()}
		tempBytes, _ := json.Marshal(tempInfo)
		saveDecompressError := b.dbHandle.Put([]byte(CompressKVPrefix+tempCompressName), tempBytes)
		if saveDecompressError != nil {
			b.logger.Errorf("save file %s compress info got error %s ", tempCompressName, saveDecompressError.Error())
		}
		currentHeight = tempEnd + 1
		b.logger.Infof("CompressUnderHeight.loop tempStart %d , tempEnd %d , tempCompressName %s",
			tempStart, tempEnd, tempCompressName)
	}
	return startCompressHeight, endCompressHeight, nil
}

// verifyCompressedHeight 校验压缩高度
// @receiver b
// @param height
// @return uint64
// @return error
func (b *BlockFileDB) verifyCompressedHeight(height uint64) (uint64, error) {
	compressedHeight, compressedError := b.GetCompressedHeight()
	if compressedError != nil {
		return 0, compressedError
	}
	if height <= compressedHeight {
		return compressedHeight, errors.New("block height has compressed")
	}
	return compressedHeight, nil
}

// findMaxHeightUnderHeight 寻找height所在文件的最高区块高度，将这个高度和maxHeight中取一个最小值
// @receiver b
// @param height
// @param maxHeight
// @return uint64
// @return error
func (b *BlockFileDB) findMaxHeightUnderHeight(height, maxHeight uint64) (uint64, error) {
	store, storeError := b.GetBlockIndex(height)
	if storeError != nil {
		return 0, storeError
	}
	beginHeight, beginHeightError := strconv.ParseUint(store.FileName, 10, 64)
	if beginHeightError != nil {
		return 0, beginHeightError
	}
	if beginHeight == 1 {
		return 0, nil
	}
	retHeight := beginHeight - 1
	if retHeight > maxHeight {
		retHeight = maxHeight
	}
	return retHeight, nil
}

// ReapFiles 清除已经过期的解压缩文件，或者已经压缩过的文件
// @receiver b
// @param filePrefix
// @param isDecompress
// @param retainSeconds
// @return
func (b *BlockFileDB) ReapFiles(filePrefix string, isDecompress bool, retainSeconds int64) {
	archive_utils.GlobalServerLatch.Add(1)
	defer archive_utils.GlobalServerLatch.Done()
	listIter, listIterError := b.dbHandle.NewIteratorWithPrefix([]byte(filePrefix))
	if listIterError != nil {
		b.logger.Errorf("reapFiles new iterator got error %s , prefix %s", listIterError.Error(), filePrefix)
		return
	}
	defer listIter.Release()
	nowUnix := time.Now().Unix()
	b.logger.Infof("ReapFiles.begin at %d, filePrefix %s , isDecompress %+v",
		nowUnix, filePrefix, isDecompress)
	for listIter.Next() {
		prefixKey := string(listIter.Key())
		var tempFileInfo DecompressFileInfo
		unMarshalErr := json.Unmarshal(listIter.Value(), &tempFileInfo)
		if unMarshalErr != nil {
			b.logger.Errorf("reapFiles prefix %s ,file %s unmarshal error %s",
				filePrefix, prefixKey, unMarshalErr.Error())
			continue
		}
		b.logger.Infof("ReapFiles prefixKey %s ,filename %s , createdAt %d, nowUnix %d",
			prefixKey, tempFileInfo.FileName, tempFileInfo.CreatedAt, nowUnix)
		if nowUnix-tempFileInfo.CreatedAt < retainSeconds {
			continue
		}
		removeSuccess, removeError := b.fileStore.TryRemoveFile(tempFileInfo.FileName, isDecompress)
		if removeError != nil {
			b.logger.Errorf("reapFiles TryRemoveFile %s got error %s",
				tempFileInfo.FileName, removeError.Error())
		}
		if !removeSuccess {
			continue
		}
		deleteError := b.dbHandle.Delete([]byte(prefixKey))
		if deleteError != nil {
			b.logger.Errorf("reapFiles kv delete key %s got error %s ", prefixKey, deleteError.Error())
		}
	}
}
