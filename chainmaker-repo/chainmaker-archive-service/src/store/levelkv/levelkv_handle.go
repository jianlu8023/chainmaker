/*
Copyright (C) BABEC. All rights reserved.


SPDX-License-Identifier: Apache-2.0
*/

// Package levelkv define leveldb operations
package levelkv

import (
	"fmt"
	"path/filepath"
	"sync"
	"time"

	"chainmaker.org/chainmaker/protocol/v2"
	"github.com/pkg/errors"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/opt"
	"github.com/syndtr/goleveldb/leveldb/util"
)

// DbType_Leveldb 数据库类型
var DbType_Leveldb = "leveldb"
var _ protocol.DBHandle = (*LevelDBHandle)(nil)

const (
	defaultBloomFilterBits = 10

	defaultWriteBufferSize = 4 * opt.MiB
)

// LevelDBHandle encapsulated handle to leveldb
type LevelDBHandle struct {
	writeLock      sync.Mutex
	db             *leveldb.DB
	logger         protocol.Logger
	writeBatchSize uint64
}

// GetWriteBatchSize 返回写批量大小
func (h *LevelDBHandle) GetWriteBatchSize() uint64 {
	return h.writeBatchSize
}

// NewLevelDBOptions 配置
type NewLevelDBOptions struct {
	Config           *LevelDbConfig
	Logger           protocol.Logger
	ChainGenesisHash string
	PathPrefix       string
}

func createLevelDBHandleByPath(path string, dbConfig *LevelDbConfig, logger protocol.Logger) *leveldb.DB {
	dbPath, _ := filepath.Abs(path)
	err := createDirIfNotExist(dbPath)
	if err != nil {
		panic(fmt.Sprintf("Error create dir %s by leveldbprovider: %s", dbPath, err))
	}
	db, err := leveldb.OpenFile(dbPath, setupOptions(dbConfig))
	if err != nil {
		panic(fmt.Sprintf("Error opening %s by leveldbprovider: %s", dbPath, err))
	}
	logger.Debugf("open leveldb:%s", dbPath)
	return db
}

// NewLevelDBHandle 初始化leveldb
func NewLevelDBHandle(options *NewLevelDBOptions) *LevelDBHandle {
	dbConfig := options.Config
	logger := options.Logger
	var dbPath string
	if len(options.ChainGenesisHash) > 0 {
		// 保存链的索引的文件夹
		dbPath = filepath.Join(options.PathPrefix, options.ChainGenesisHash)
	} else {
		// 保存的为系统信息
		dbPath = options.PathPrefix
	}
	db := createLevelDBHandleByPath(dbPath, dbConfig, logger)
	return &LevelDBHandle{
		db:     db,
		logger: logger,
	}
}

// GetDbType returns db type
func (h *LevelDBHandle) GetDbType() string {
	return DbType_Leveldb
}

// Get returns the value for the given key, or returns nil if none exists
func (h *LevelDBHandle) Get(key []byte) ([]byte, error) {
	value, err := h.db.Get(key, nil)
	if err == leveldb.ErrNotFound {
		value = nil
		err = nil
	}
	if err != nil {
		h.logger.Errorf("getting leveldbprovider key [%s], err:%s", key, err.Error())
		return nil, errors.Wrapf(err, "error getting leveldbprovider key [%s]", key)
	}
	return value, nil
}

// GetKeys returns the value for the given key
func (h *LevelDBHandle) GetKeys(keys [][]byte) ([][]byte, error) {
	if len(keys) == 0 {
		return nil, nil
	}

	wg := sync.WaitGroup{}
	errsChan := make(chan error, len(keys))
	wg.Add(len(keys))
	values := make([][]byte, len(keys))
	for index, k := range keys {
		go func(i int, key []byte) {
			defer wg.Done()

			if len(key) == 0 {
				values[i] = nil
				return
			}

			value, err := h.db.Get(key, nil)
			if err == leveldb.ErrNotFound {
				value = nil
				err = nil
			}
			values[i] = value
			if err != nil {
				h.logger.Errorf("getting leveldbprovider key [%s], err:%s", key, err.Error())
				errsChan <- errors.Wrapf(err, "error getting leveldbprovider key [%s]", key)
			}
		}(index, k)
	}

	wg.Wait()

	if len(errsChan) > 0 {
		return nil, <-errsChan
	}

	return values, nil
}

// Put saves the key-values
func (h *LevelDBHandle) Put(key []byte, value []byte) error {
	if value == nil {
		h.logger.Warn("writing leveldbprovider key [%s] with nil value", key)
		return errors.New("error writing leveldbprovider with nil value")
	}

	err := h.db.Put(key, value, &opt.WriteOptions{Sync: true})
	if err != nil {
		h.logger.Errorf("writing leveldbprovider key [%s]", key)
		return errors.Wrapf(err, "error writing leveldbprovider key [%s]", key)
	}
	return err
}

// Has return true if the given key exist, or return false if none exists
func (h *LevelDBHandle) Has(key []byte) (bool, error) {
	exist, err := h.db.Has(key, nil)
	if err != nil {
		h.logger.Errorf("getting leveldbprovider key [%s], err:%s", key, err.Error())
		return false, errors.Wrapf(err, "error getting leveldbprovider key [%s]", key)
	}
	return exist, nil
}

// Delete deletes the given key
func (h *LevelDBHandle) Delete(key []byte) error {
	wo := &opt.WriteOptions{Sync: true}
	err := h.db.Delete(key, wo)
	if err != nil {
		h.logger.Errorf("deleting leveldbprovider key [%s]", key)
		return errors.Wrapf(err, "error deleting leveldbprovider key [%s]", key)
	}
	return err
}

// WriteBatch writes a batch in an atomic operation
func (h *LevelDBHandle) WriteBatch(batch protocol.StoreBatcher, sync bool) error {
	start := time.Now()
	if batch.Len() == 0 {
		return nil
	}
	h.writeLock.Lock()
	defer h.writeLock.Unlock()
	levelBatch := &leveldb.Batch{}
	for k, v := range batch.KVs() {
		key := []byte(k)
		if v == nil {
			levelBatch.Delete(key)
		} else {
			levelBatch.Put(key, v)
		}
	}

	batchFilterDur := time.Since(start)
	wo := &opt.WriteOptions{Sync: sync}
	if err := h.db.Write(levelBatch, wo); err != nil {
		h.logger.Errorf("write batch to leveldb provider failed")
		return errors.Wrap(err, "error writing batch to leveldb provider")
	}
	writeDur := time.Since(start)
	h.logger.Debugf("leveldb write batch[%d] sync: %v, time used: (filter:%d, write:%d, total:%d)",
		batch.Len(), sync, batchFilterDur.Milliseconds(), (writeDur - batchFilterDur).Milliseconds(),
		time.Since(start).Milliseconds())
	return nil
}

// CompactRange compacts the underlying DB for the given key range.
func (h *LevelDBHandle) CompactRange(start, limit []byte) error {
	return h.db.CompactRange(util.Range{
		Start: start,
		Limit: limit,
	})
}

// NewIteratorWithRange returns an iterator that contains all the key-values between given key ranges
// start is included in the results and limit is excluded.
func (h *LevelDBHandle) NewIteratorWithRange(startKey []byte, limitKey []byte) (protocol.Iterator, error) {
	if len(startKey) == 0 || len(limitKey) == 0 {
		return nil, fmt.Errorf("iterator range should not start(%s) or limit(%s) with empty key",
			string(startKey), string(limitKey))
	}
	keyRange := &util.Range{Start: startKey, Limit: limitKey}
	iter := h.db.NewIterator(keyRange, nil)
	return iter, nil
}

// NewIteratorWithPrefix returns an iterator that contains all the key-values with given prefix
func (h *LevelDBHandle) NewIteratorWithPrefix(prefix []byte) (protocol.Iterator, error) {
	if len(prefix) == 0 {
		return nil, fmt.Errorf("iterator prefix should not be empty key")
	}
	r := util.BytesPrefix(prefix)
	return h.NewIteratorWithRange(r.Start, r.Limit)
}

// Close closes the leveldb
func (h *LevelDBHandle) Close() error {
	h.writeLock.Lock()
	defer h.writeLock.Unlock()
	return h.db.Close()
}
