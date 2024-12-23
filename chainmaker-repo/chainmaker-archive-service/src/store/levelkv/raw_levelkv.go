/*
Copyright (C) BABEC. All rights reserved.


SPDX-License-Identifier: Apache-2.0
*/

// Package levelkv define leveldb operations
package levelkv

import (
	"os"

	"github.com/syndtr/goleveldb/leveldb/filter"
	"github.com/syndtr/goleveldb/leveldb/opt"
)

func setupOptions(lcfg *LevelDbConfig) *opt.Options {
	dbOpts := &opt.Options{
		WriteBuffer: defaultWriteBufferSize,
	}
	if lcfg.WriteBufferSize > 0 {
		dbOpts.WriteBuffer = lcfg.WriteBufferSize * opt.MiB
	}
	bloomFilterBits := lcfg.BloomFilterBits
	if bloomFilterBits <= 0 {
		bloomFilterBits = defaultBloomFilterBits
	}
	dbOpts.Filter = filter.NewBloomFilter(bloomFilterBits)

	if lcfg.NoSync {
		dbOpts.NoSync = lcfg.NoSync
	}

	if lcfg.DisableBufferPool {
		dbOpts.DisableBufferPool = lcfg.DisableBufferPool
	}

	if lcfg.Compression > 0 {
		dbOpts.Compression = opt.Compression(lcfg.Compression)
	}

	if lcfg.DisableBlockCache {
		dbOpts.DisableBlockCache = lcfg.DisableBlockCache
	}

	if lcfg.BlockCacheCapacity > 0 {
		dbOpts.BlockCacheCapacity = lcfg.BlockCacheCapacity * opt.MiB
	}

	if lcfg.BlockSize > 0 {
		dbOpts.BlockSize = lcfg.BlockSize * opt.KiB
	}

	if lcfg.CompactionTableSize > 0 {
		dbOpts.CompactionTableSize = lcfg.CompactionTableSize * opt.MiB
	}

	if lcfg.CompactionTotalSize > 0 {
		dbOpts.CompactionTotalSize = lcfg.CompactionTotalSize * opt.MiB
	}

	if lcfg.WriteL0PauseTrigger > 0 {
		dbOpts.WriteL0PauseTrigger = lcfg.WriteL0PauseTrigger
	}

	if lcfg.WriteL0SlowdownTrigger > 0 {
		dbOpts.WriteL0SlowdownTrigger = lcfg.WriteL0SlowdownTrigger
	}

	if lcfg.CompactionL0Trigger > 0 {
		dbOpts.CompactionL0Trigger = lcfg.CompactionL0Trigger
	}

	return dbOpts
}

func createDirIfNotExist(path string) error {
	_, err := os.Stat(path)
	if err == nil {
		return nil
	}
	if os.IsNotExist(err) {
		// 创建文件夹
		err := os.MkdirAll(path, os.ModePerm)
		if err != nil {
			return err
		}
	}
	return nil
}
