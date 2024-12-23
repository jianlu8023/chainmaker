/*
Copyright (C) BABEC. All rights reserved.


SPDX-License-Identifier: Apache-2.0
*/

// Package levelkv define leveldb operations
package levelkv

// LevelDbConfig 配置
type LevelDbConfig struct {
	WriteBatchSize         uint64 `mapstructure:"write_batch_size"`
	WriteBufferSize        int    `mapstructure:"write_buffer_size"`
	BloomFilterBits        int    `mapstructure:"bloom_filter_bits"`
	NoSync                 bool   `mapstructure:"no_sync"`
	DisableBufferPool      bool   `mapstructure:"disable_buffer_pool"`
	Compression            uint   `mapstructure:"compression"`
	DisableBlockCache      bool   `mapstructure:"disable_block_cache"`
	BlockCacheCapacity     int    `mapstructure:"block_cache_capacity"`
	BlockSize              int    `mapstructure:"block_size"`
	CompactionTableSize    int    `mapstructure:"compaction_table_size"`
	CompactionTotalSize    int    `mapstructure:"compaction_total_size"`
	WriteL0PauseTrigger    int    `mapstructure:"write_l0_pause_trigger"`
	WriteL0SlowdownTrigger int    `mapstructure:"write_l0_slowdown_trigger"`
	CompactionL0Trigger    int    `mapstructure:"compaction_l0_trigger"`
}
