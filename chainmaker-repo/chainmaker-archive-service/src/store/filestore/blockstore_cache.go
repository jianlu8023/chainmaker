/*
Copyright (C) BABEC. All rights reserved.


SPDX-License-Identifier: Apache-2.0
*/

// Package filestore define file operation
package filestore

import (
	"errors"
	"sync"

	"chainmaker.org/chainmaker-archive-service/src/archive_utils"
	"chainmaker.org/chainmaker/protocol/v2"
)

const defaultMaxBlockSize = 10

// StoreCacheMgr provide handle to cache instances
type StoreCacheMgr struct {
	sync.RWMutex
	pendingBlockUpdates map[uint64]protocol.StoreBatcher // block height -> StoreBatcher
	cacheSize           int                              //block size in cache, if cache size <= 0, use defalut size = 10

	logger protocol.Logger
}

// NewStoreCacheMgr construct a new `StoreCacheMgr` with given chainId
func NewStoreCacheMgr(genesisHash string, blockWriteBufferSize int, logger protocol.Logger) *StoreCacheMgr {
	if blockWriteBufferSize <= 0 {
		blockWriteBufferSize = defaultMaxBlockSize
	}
	storeCacheMgr := &StoreCacheMgr{
		pendingBlockUpdates: make(map[uint64]protocol.StoreBatcher),
		cacheSize:           blockWriteBufferSize,
		logger:              logger,
	}
	return storeCacheMgr
}

// AddBlock cache a block with given block height and update batch
// 上锁，顺序操作
// @receiver mgr
// @param blockHeight
// @param updateBatch
// @return
func (mgr *StoreCacheMgr) AddBlock(blockHeight uint64, updateBatch protocol.StoreBatcher) {

	mgr.Lock()
	defer mgr.Unlock()
	mgr.pendingBlockUpdates[blockHeight] = updateBatch

	//不需要更新 treeMap了
	if mgr.logger != nil {
		mgr.logger.Debugf("add block[%d] to cache, block size:%d", blockHeight, mgr.getPendingBlockSize())
	}
	//更新 最后一次写的block 的高度，到 StoreCacheMgr 中
}

// DelBlock delete block for the given block height
// @receiver mgr
// @param blockHeight
// @return
func (mgr *StoreCacheMgr) DelBlock(blockHeight uint64) {

	mgr.Lock()
	defer mgr.Unlock()

	delete(mgr.pendingBlockUpdates, blockHeight)
	if mgr.logger != nil {
		mgr.logger.Debugf("del block[%d] from cache, block size:%d", blockHeight, mgr.getPendingBlockSize())
	}
}

// Get returns value if the key in cache, or returns nil if none exists.
// @receiver mgr
// @param key
// @return []byte
// @return bool
func (mgr *StoreCacheMgr) Get(key string) ([]byte, bool) {
	mgr.RLock()
	defer mgr.RUnlock()
	//pendingBlockUpdates，按照块高，从大到小，获得对应map，在map中查找key，如果key存在，则返回
	keysArr := []uint64{}
	for key := range mgr.pendingBlockUpdates {
		keysArr = append(keysArr, key)
	}
	//快高从大到小排序
	keysArr = QuickSort(keysArr)
	for i := 0; i < len(keysArr); i++ {
		if storageCache, exists := mgr.pendingBlockUpdates[keysArr[i]]; exists {
			v, err := storageCache.Get([]byte(key))
			//如果没找到，从cache中找下一个块高对应的key/value
			if err != nil {
				if err.Error() == archive_utils.ErrValueNotFound.Error() {
					continue
				}
			}
			if err != nil {
				return nil, false
			}
			return v, true
		}
	}
	//如果每个块高，对应的map,中都不存在对应的key，则返回nil,false
	return nil, false

}

// Has returns true if the key in cache, or returns false if none exists.
// return isDelete,isExist
// 如果这个key 对应value 是 nil，说明这个key被删除了
// 所以查找到第一个key，要判断 这个key 是否是被删除的
// @receiver mgr
// @param key
// @return bool
// @return bool
func (mgr *StoreCacheMgr) Has(key string) (bool, bool) {
	mgr.RLock()
	defer mgr.RUnlock()
	//pendingBlockUpdates，按照块高，从大到小，获得对应map，在map中查找key，如果key存在，则返回
	keysArr := []uint64{}
	for key := range mgr.pendingBlockUpdates {
		keysArr = append(keysArr, key)
	}
	//快高从大到小排序
	keysArr = QuickSort(keysArr)
	for i := 0; i < len(keysArr); i++ {
		if _, exists := mgr.pendingBlockUpdates[keysArr[i]]; exists {
			v, err := mgr.pendingBlockUpdates[keysArr[i]].Get([]byte(key))
			//如果没找到，从cache中找下一个块高对应的key/value
			if err != nil {
				if err.Error() == archive_utils.ErrValueNotFound.Error() {
					continue
				}
			}
			//如果出现其他错误，直接报错
			if err != nil {
				panic(err)
			}
			//未删除，存在
			if v != nil {
				return false, true

			}
			//已删除，存在
			return true, true
		}
	}
	//未删除，不存在
	return false, false

}

// Clear 清除缓存，目前未做任何清除操作
func (mgr *StoreCacheMgr) Clear() {
	if mgr.logger != nil {
		mgr.logger.Infof("Clear")
	}

}

// LockForFlush used to lock cache until all cache item be flushed to db
// 目前未做任何清除操作
func (mgr *StoreCacheMgr) LockForFlush() {
	if mgr.logger != nil {
		mgr.logger.Infof("LockForFlush")
	}

}

// UnLockFlush used to unlock cache by release all semaphore
// 目前未做任何清除操作
func (mgr *StoreCacheMgr) UnLockFlush() {
	if mgr.logger != nil {
		mgr.logger.Infof("UnLockFlush")
	}

}

// getPendingBlockSize 返回当前缓存的区块高度
// @receiver mgr
// @return int
func (mgr *StoreCacheMgr) getPendingBlockSize() int {
	return len(mgr.pendingBlockUpdates)
}

// KVRange get data from mgr , [startKey,endKey)
// @receiver mgr
// @param startKey
// @param endKey
// @return map[string][]byte
// @return error
func (mgr *StoreCacheMgr) KVRange(startKey []byte, endKey []byte) (map[string][]byte, error) {
	keyMap := make(map[string][]byte)
	//得到[startKey,endKey) 的 keys
	for _, batch := range mgr.pendingBlockUpdates {
		//keys := make(map[string][]byte)
		batchKV := batch.KVs()
		for k := range batchKV {
			if k >= string(startKey) && k < string(endKey) {
				keyMap[k] = nil
			}
		}
	}
	//得到对应 value
	for k := range keyMap {
		if getV, exist := mgr.Get(k); exist {
			keyMap[k] = getV
		} else {
			delete(keyMap, k)
		}

	}

	return keyMap, nil
}

// GetBatch 根据块高，返回 块对应的cache
// @receiver mgr
// @param height
// @return StoreBatcher
// @return error
func (mgr *StoreCacheMgr) GetBatch(height uint64) (protocol.StoreBatcher, error) {
	mgr.RLock()
	defer mgr.RUnlock()
	if v, exists := mgr.pendingBlockUpdates[height]; exists {
		return v, nil
	}

	return nil, errors.New("not found")
}

// QuickSort 快排，获得从小到大排序后的结果，返回
// @param arr
// @return []uint64
func QuickSort(arr []uint64) []uint64 {
	if len(arr) <= 1 {
		return arr
	}
	if len(arr) == 2 {
		if arr[0] < arr[1] {
			return append([]uint64{}, arr[1], arr[0])
		}
	}
	var (
		left, middle, right []uint64
	)
	left, middle, right = []uint64{}, []uint64{}, []uint64{}
	mid := arr[len(arr)/2]
	for i := 0; i < len(arr); i++ {
		if arr[i] > mid {
			left = append(left, arr[i])
		}
		if arr[i] == mid {
			middle = append(middle, arr[i])

		}
		if arr[i] < mid {
			right = append(right, arr[i])
		}
	}
	return append(append(QuickSort(left), middle...), QuickSort(right)...)
}
