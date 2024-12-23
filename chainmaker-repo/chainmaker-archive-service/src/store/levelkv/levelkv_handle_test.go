/*
Copyright (C) BABEC. All rights reserved.


SPDX-License-Identifier: Apache-2.0
*/

// Package levelkv define leveldb operations
package levelkv

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"chainmaker.org/chainmaker/protocol/v2"
	"chainmaker.org/chainmaker/protocol/v2/test"
	"github.com/stretchr/testify/assert"
)

// UpdateBatch encloses the details of multiple `updates`
type UpdateBatch struct {
	kvs map[string][]byte
}

func (batch *UpdateBatch) Get(key []byte) ([]byte, error) {
	//TODO implement me
	panic("implement me")
}

func (batch *UpdateBatch) Has(key []byte) bool {
	//TODO implement me
	panic("implement me")
}

func (batch *UpdateBatch) SplitBatch(batchCnt uint64) []protocol.StoreBatcher {
	panic("implement me")
}

// NewUpdateBatch constructs an instance of a Batch
func NewUpdateBatch() protocol.StoreBatcher {
	return &UpdateBatch{kvs: make(map[string][]byte)}
}

// KVs return map
func (batch *UpdateBatch) KVs() map[string][]byte {
	return batch.kvs
}

// Put adds a KV
func (batch *UpdateBatch) Put(key []byte, value []byte) {
	if value == nil {
		panic("Nil value not allowed")
	}
	batch.kvs[string(key)] = value
}

// Delete deletes a Key and associated value
func (batch *UpdateBatch) Delete(key []byte) {
	batch.kvs[string(key)] = nil
}

// Len returns the number of entries in the batch
func (batch *UpdateBatch) Len() int {
	return len(batch.kvs)
}

// Merge merges other kvs to this updateBatch
func (batch *UpdateBatch) Merge(u protocol.StoreBatcher) {
	for key, value := range u.KVs() {
		batch.kvs[key] = value
	}
}

var log = &test.GoLogger{}
var dbConfig = &LevelDbConfig{
	BloomFilterBits: 10,
}

func TestDBHandle_NewLevelDBHandle(t *testing.T) {
	defer func() {
		err := recover()
		//assert.Equal(t, strings.Contains(err.(string), "Error create dir"), true)
		fmt.Println(err)
	}()
	dbConfigTest := &LevelDbConfig{
		BloomFilterBits: 10,
	}
	op := &NewLevelDBOptions{ChainGenesisHash: "x8lise1", PathPrefix: "../../index_data", Config: dbConfigTest, Logger: log}
	dbHandle1 := NewLevelDBHandle(op)
	dbHandle1.Close()

	dbConfigTest = &LevelDbConfig{
		BloomFilterBits: 10,
	}
	dbHandle2 := NewLevelDBHandle(&NewLevelDBOptions{ChainGenesisHash: "x8lise2", PathPrefix: "../../index_data", Config: dbConfigTest, Logger: log})
	dbHandle2.Close()

	dbConfigTest = &LevelDbConfig{
		BloomFilterBits: 10,
	}
	dbHandle3 := NewLevelDBHandle(&NewLevelDBOptions{ChainGenesisHash: "x8lise3", PathPrefix: "../../index_data", Config: dbConfigTest, Logger: log})
	dbHandle3.Close()
	defer os.RemoveAll("../../index_data")
}

func TestDBHandle_Put(t *testing.T) {
	dbHandle := NewLevelDBHandle(&NewLevelDBOptions{ChainGenesisHash: "x8lise4", PathPrefix: "../../index_data", Config: dbConfig, Logger: log}) //dbPath：db文件的存储路径
	defer dbHandle.Close()
	defer os.RemoveAll("../../index_data")
	key1 := []byte("key1")
	value1 := []byte("value1")
	err := dbHandle.Put(key1, value1)
	assert.Nil(t, err)

	value, err := dbHandle.Get(key1)
	assert.Nil(t, err)
	assert.Equal(t, value1, value)
	value, err = dbHandle.Get([]byte("another key"))
	assert.Nil(t, err)
	assert.Nil(t, value)

	defer dbHandle.Close()

	_, err = dbHandle.Get(key1)
	assert.Nil(t, err)
	//assert.NotNil(t, err)
	//assert.Equal(t, strings.Contains(err.Error(), "getting leveldbprovider key"), true)

	err = dbHandle.Put(key1, nil)
	assert.NotNil(t, err)
	assert.Equal(t, strings.Contains(err.Error(), "writing leveldbprovider with nil value"), true)

	key2 := []byte("key2")
	value2 := []byte("value2")
	err = dbHandle.Put(key2, value2)
	assert.Nil(t, err)
	//	assert.NotNil(t, err)
	//	assert.Equal(t, strings.Contains(err.Error(), "writing leveldbprovider key"), true)
}

func TestDBHandle_Delete(t *testing.T) {
	dbHandle := NewLevelDBHandle(&NewLevelDBOptions{ChainGenesisHash: "x8lise", PathPrefix: "../../index_data", Config: dbConfig, Logger: log}) //dbPath：db文件的存储路径
	defer dbHandle.Close()
	defer os.RemoveAll("../../index_data")
	key1 := []byte("key1")
	value1 := []byte("value1")
	err := dbHandle.Put(key1, value1)
	assert.Nil(t, err)

	exist, err := dbHandle.Has(key1)
	assert.Nil(t, err)
	assert.True(t, exist)

	err = dbHandle.Delete(key1)
	assert.Nil(t, err)

	exist, err = dbHandle.Has(key1)
	assert.Nil(t, err)
	assert.False(t, exist)

	defer dbHandle.Close()

	err = dbHandle.Delete(key1)
	assert.Nil(t, err)
	//assert.NotNil(t, err)
	//assert.Equal(t, strings.Contains(err.Error(), "deleting leveldbprovider key"), true)

	_, err = dbHandle.Has(key1)
	assert.Nil(t, err)
	//assert.NotNil(t, err)
	//assert.Equal(t, strings.Contains(err.Error(), "getting leveldbprovider key"), true)
}

func TestDBHandle_WriteBatch(t *testing.T) {
	dbHandle := NewLevelDBHandle(&NewLevelDBOptions{ChainGenesisHash: "x8lise", PathPrefix: "../../index_data", Config: dbConfig, Logger: log}) //dbPath：db文件的存储路径
	defer dbHandle.Close()
	defer os.RemoveAll("../../index_data")
	batch := NewUpdateBatch()

	err := dbHandle.WriteBatch(batch, true)
	assert.Nil(t, err)

	key1 := []byte("key1")
	value1 := []byte("value1")
	key2 := []byte("key2")
	value2 := []byte("value2")
	batch.Put(key1, value1)
	batch.Put(key2, value2)
	err = dbHandle.WriteBatch(batch, true)
	assert.Nil(t, err)

	key3 := []byte("key3")
	value3 := []byte("")
	batch.Put(key3, value3)
	err = dbHandle.WriteBatch(batch, true)
	assert.Nil(t, err)

	value, err := dbHandle.Get(key2)
	assert.Nil(t, err)
	assert.Equal(t, value2, value)

	defer dbHandle.Close()

	err = dbHandle.WriteBatch(batch, true)
	assert.Nil(t, err)
	//assert.NotNil(t, err)
	//assert.Equal(t, strings.Contains(err.Error(), "error writing batch to leveldb provider"), true)
}

func TestDBHandle_CompactRange(t *testing.T) {
	dbHandle := NewLevelDBHandle(&NewLevelDBOptions{ChainGenesisHash: "x8lise", PathPrefix: "../../index_data", Config: dbConfig, Logger: log}) //dbPath：db文件的存储路径
	defer dbHandle.Close()
	defer os.RemoveAll("../../index_data")
	batch := NewUpdateBatch()
	key1 := []byte("key1")
	value1 := []byte("value1")
	key2 := []byte("key2")
	value2 := []byte("value2")
	batch.Put(key1, value1)
	batch.Put(key2, value2)
	err := dbHandle.WriteBatch(batch, true)
	assert.Nil(t, err)

	err = dbHandle.CompactRange(key1, key2)
	assert.Nil(t, err)
}

func TestDBHandle_NewIteratorWithRange(t *testing.T) {
	dbHandle := NewLevelDBHandle(&NewLevelDBOptions{ChainGenesisHash: "x8lise", PathPrefix: "../../index_data", Config: dbConfig, Logger: log}) //dbPath：db文件的存储路径
	defer dbHandle.Close()
	defer os.RemoveAll("../../index_data")
	batch := NewUpdateBatch()
	key1 := []byte("key1")
	value1 := []byte("value1")
	key2 := []byte("key2")
	value2 := []byte("value2")
	batch.Put(key1, value1)
	batch.Put(key2, value2)
	err := dbHandle.WriteBatch(batch, true)
	assert.Nil(t, err)

	iter, err := dbHandle.NewIteratorWithRange(key1, []byte("key3"))
	assert.Nil(t, err)
	defer iter.Release()
	var count int
	for iter.Next() {
		count++
	}
	assert.Equal(t, 2, count)

	_, err = dbHandle.NewIteratorWithRange([]byte(""), []byte(""))
	assert.NotNil(t, err)
	assert.Equal(t, strings.Contains(err.Error(), "iterator range should not start"), true)
}

func TestDBHandle_NewIteratorWithPrefix(t *testing.T) {
	dbHandle := NewLevelDBHandle(&NewLevelDBOptions{ChainGenesisHash: "x8lise", PathPrefix: "../../index_data", Config: dbConfig, Logger: log}) //dbPath：db文件的存储路径
	defer dbHandle.Close()
	defer os.RemoveAll("../../index_data")
	batch := NewUpdateBatch()

	batch.Put([]byte("pkey1"), []byte("value1"))
	batch.Put([]byte("pkey2"), []byte("value2"))
	batch.Put([]byte("pkey3"), []byte("value3"))
	batch.Put([]byte("pkey4"), []byte("value4"))
	batch.Put([]byte("pkeyx"), []byte("value5"))
	batch.Put([]byte("pkey-+"), []byte("-value="))
	batch.Put([]byte("hkey"), []byte("hkey"))
	batch.Put([]byte("12key"), []byte("12value"))

	err := dbHandle.WriteBatch(batch, true)
	assert.Equal(t, nil, err)

	iter, err := dbHandle.NewIteratorWithPrefix([]byte("pkey"))
	assert.Nil(t, err)
	defer iter.Release()
	var count int
	for iter.Next() {
		count++
		fmt.Printf("key: %s, value: %s \n", string(iter.Key()), string(iter.Value()))
		//key := string(iter.Key())
		//fmt.Println(fmt.Sprintf("key: %s", key))
	}
	assert.Equal(t, 6, count)

	_, err = dbHandle.NewIteratorWithPrefix([]byte(""))
	assert.NotNil(t, err)
	assert.Equal(t, strings.Contains(err.Error(), "iterator prefix should not be empty key"), true)
}

func TestTempFolder(t *testing.T) {
	t.Log(os.TempDir())
}
func TestDBHandle_NewIteratorWithPrefix_SM4Encryptor(t *testing.T) {

	dbHandle := NewLevelDBHandle(&NewLevelDBOptions{ChainGenesisHash: "x8lise", PathPrefix: "../../index_data", Config: dbConfig, Logger: log}) //dbPath：db文件的存储路径
	defer dbHandle.Close()
	defer os.RemoveAll("../../index_data")
	batch := NewUpdateBatch()

	batch.Put([]byte("key1"), []byte("value1"))
	batch.Put([]byte("key2"), []byte("value2"))
	batch.Put([]byte("key3"), []byte("value3"))
	batch.Put([]byte("key4"), []byte("value4"))
	batch.Put([]byte("keyx"), []byte("value5"))
	batch.Put([]byte("keynull"), []byte("will delete"))
	batch.Delete([]byte("keynull"))
	err := dbHandle.WriteBatch(batch, true)
	assert.Equal(t, nil, err)

	iter, err := dbHandle.NewIteratorWithPrefix([]byte("key"))
	assert.Nil(t, err)
	defer iter.Release()
	var count int
	for iter.Next() {
		count++
		key := string(iter.Key())
		t.Logf("key: %s,value:%s", key, iter.Value())
	}
	assert.Equal(t, 5, count)

	_, err = dbHandle.NewIteratorWithPrefix([]byte(""))
	assert.NotNil(t, err)
	assert.Equal(t, strings.Contains(err.Error(), "iterator prefix should not be empty key"), true)
}
func TestLevelDBHandle_Get(t *testing.T) {

	dbHandle := NewLevelDBHandle(&NewLevelDBOptions{ChainGenesisHash: "x8lise", PathPrefix: "../../index_data", Config: dbConfig, Logger: log}) //dbPath：db文件的存储路径
	defer dbHandle.Close()
	defer os.RemoveAll("../../index_data")
	err := dbHandle.Put([]byte("key1"), []byte{})
	assert.Nil(t, err)
	_, err = dbHandle.Get([]byte("key1"))
	assert.Nil(t, err)
	_, err = dbHandle.Get([]byte("key2"))
	assert.Nil(t, err)
}

func TestLevelDBHandle_Gets(t *testing.T) {
	dbHandle := NewLevelDBHandle(&NewLevelDBOptions{ChainGenesisHash: "x8lise", PathPrefix: "../../index_data", Config: dbConfig, Logger: log}) //dbPath：db文件的存储路径
	defer dbHandle.Close()
	defer os.RemoveAll("../../index_data")
	n := 3
	keys := make([][]byte, 0, n+1)
	values := make([][]byte, 0, n+1)
	batch := NewUpdateBatch()
	for i := 0; i < n; i++ {
		keyi := []byte(fmt.Sprintf("key%d", i))
		valuei := []byte(fmt.Sprintf("value%d", i))
		keys = append(keys, keyi)
		values = append(values, valuei)

		batch.Put(keyi, valuei)
	}

	err := dbHandle.WriteBatch(batch, true)
	assert.Nil(t, err)

	keys = append(keys, nil)
	values = append(values, nil)
	valuesR, err1 := dbHandle.GetKeys(keys)
	assert.Nil(t, err1)
	for i := 0; i < len(keys); i++ {
		assert.Equal(t, values[i], valuesR[i])
	}
}
