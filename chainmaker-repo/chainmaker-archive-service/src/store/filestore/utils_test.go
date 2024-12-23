/*
Copyright (C) BABEC. All rights reserved.


SPDX-License-Identifier: Apache-2.0
*/

// Package filestore define file operation
package filestore

import (
	"testing"

	storePb "chainmaker.org/chainmaker/pb-go/v2/store"
	"github.com/stretchr/testify/assert"
)

func TestPathExists(t *testing.T) {
	exists, existsErr := PathExists("./utils.go")
	assert.Nil(t, existsErr)
	assert.True(t, exists)
}

func TestConstructDBIndexInfo(t *testing.T) {
	storeVar := storePb.StoreInfo{
		StoreType: storePb.DataStoreType_FILE_STORE,
		FileName:  "00001.fdb",
		Offset:    200,
	}
	dataBytes := ConstructDBIndexInfo(&storeVar, 100, 1000)
	assert.NotNil(t, dataBytes)
}

func TestDecodeValueToIndex(t *testing.T) {
	storeVar := storePb.StoreInfo{
		StoreType: storePb.DataStoreType_FILE_STORE,
		FileName:  "00001.fdb",
		Offset:    200,
	}
	dataBytes := ConstructDBIndexInfo(&storeVar, 100, 1000)
	assert.NotNil(t, dataBytes)
	decodeStore, decodeErr := DecodeValueToIndex(dataBytes)
	assert.Nil(t, decodeErr)
	assert.Equal(t, decodeStore.FileName, storeVar.FileName)
	assert.Equal(t, decodeStore.ByteLen, uint64(1000))
	assert.Equal(t, decodeStore.Offset, 100+storeVar.Offset)
}
