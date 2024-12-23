/*
Copyright (C) BABEC. All rights reserved.


SPDX-License-Identifier: Apache-2.0
*/

// Package filestore define file operation
package filestore

import (
	"os"
	"reflect"
	"testing"
	"time"

	"chainmaker.org/chainmaker/pb-go/v2/common"
	"chainmaker.org/chainmaker/pb-go/v2/store"
	storePb "chainmaker.org/chainmaker/pb-go/v2/store"
	"chainmaker.org/chainmaker/protocol/v2/test"
	"github.com/stretchr/testify/assert"
)

func initOpenFile() (*BlockFile, error) {
	logger := &test.GoLogger{}
	os.Mkdir("./test_dir", os.ModePerm)
	os.Mkdir("./test_dir/c", os.ModePerm)
	os.Mkdir("./test_dir/d", os.ModePerm)
	return Open("./test_dir",
		"./test_dir/c", "./test_dir/d",
		GetDefaultOptions(), logger)
}

func closeOpenFile(fileO *BlockFile) {
	fileO.Close()
	os.RemoveAll("./test_dir")
}

var testFileChainId = "testchainid_1"
var block0File = createConfigBlock(testFileChainId, 0, 0)

// var block1File, _ = createBlockAndRWSets4(testFileChainId, 1, 10, 0)
// var block2File, _ = createBlockAndRWSets4(testFileChainId, 2, 2, 0)
// var block3File, _ = createBlockAndRWSets4(testFileChainId, 3, 2, 0)
// var configBlock4File = createConfigBlock(testFileChainId, 4, 4)
// var block5File, _ = createBlockAndRWSets4(testFileChainId, 5, 3, 4)

func writeBlockToFile(fileO *BlockFile, block *common.Block) (*StartEndWrapBlock, error) {
	blockBytes, blockWithSerializedInfo, err := SerializeBlock(&store.BlockWithRWSet{
		Block: block,
	})
	if err != nil {
		return nil, err
	}
	fileName, offset, bytesLen, start, end, needRecordDb, writeErr := fileO.Write(block.Header.BlockHeight+1, blockBytes)
	if writeErr != nil {
		return nil, writeErr
	}
	blockWithSerializedInfo.Index = &storePb.StoreInfo{
		FileName: fileName,
		Offset:   offset,
		ByteLen:  bytesLen,
	}
	ret := &StartEndWrapBlock{
		BlockSerializedInfo: blockWithSerializedInfo,
		StartHeight:         start,
		EndHeight:           end,
		NeedRecord:          needRecordDb,
	}
	return ret, nil
}

func TestOpen(t *testing.T) {
	blockFile, blockFileErr := initOpenFile()
	assert.Nil(t, blockFileErr)
	defer closeOpenFile(blockFile)
}

func TestBlockFile_Write(t *testing.T) {
	blockFile, blockFileErr := initOpenFile()
	assert.Nil(t, blockFileErr)
	defer closeOpenFile(blockFile)
	info, infoErr := writeBlockToFile(blockFile, block0File)
	assert.Nil(t, infoErr)
	t.Logf("store info %+v \n", *info)
}

func TestBlockFile_ReadFileSection(t *testing.T) {
	blockFile, blockFileErr := initOpenFile()
	assert.Nil(t, blockFileErr)
	defer closeOpenFile(blockFile)
	info, infoErr := writeBlockToFile(blockFile, block0File)
	assert.Nil(t, infoErr)
	t.Logf("store info %+v \n", *info)
	assert.NotNil(t, info)
	assert.NotNil(t, info.BlockSerializedInfo)
	blockBytes, blockBytesErr := blockFile.ReadFileSection(false,
		info.BlockSerializedInfo.Index)
	assert.Nil(t, blockBytesErr)
	deserialBlock, deserialErr := DeserializeBlock(blockBytes)
	assert.Nil(t, deserialErr)
	assert.NotNil(t, deserialBlock)
	isSame := reflect.DeepEqual(*deserialBlock.Block, *block0File)
	assert.True(t, isSame)
	index, indexErr := blockFile.LastIndex()
	assert.Nil(t, indexErr)
	assert.Equal(t, index, uint64(1))

}

func TestBlockFile_ReadLastSegSection(t *testing.T) {
	blockFile, blockFileErr := initOpenFile()
	assert.Nil(t, blockFileErr)
	defer closeOpenFile(blockFile)
	info, infoErr := writeBlockToFile(blockFile, block0File)
	assert.Nil(t, infoErr)
	t.Logf("store info %+v \n", *info)
	assert.NotNil(t, info)
	assert.NotNil(t, info.BlockSerializedInfo)
	_, fileName, _, _, readErr := blockFile.ReadLastSegSection(1)
	assert.Nil(t, readErr)
	assert.Equal(t, info.BlockSerializedInfo.Index.FileName, fileName)
	height := blockFile.GetCanCompressHeight()
	assert.Equal(t, height, uint64(0))
}

func TestBlockFile_CheckDecompressFileExist(t *testing.T) {
	blockFile, blockFileErr := initOpenFile()
	assert.Nil(t, blockFileErr)
	defer closeOpenFile(blockFile)
	info, infoErr := writeBlockToFile(blockFile, block0File)
	assert.Nil(t, infoErr)
	t.Logf("store info %+v \n", *info)
	exists, _, existErr := blockFile.CheckDecompressFileExist(info.BlockSerializedInfo.Index.FileName)
	assert.Nil(t, existErr)
	assert.False(t, exists)
}

func TestBlockFile_TryRemoveFile(t *testing.T) {
	blockFile, blockFileErr := initOpenFile()
	assert.Nil(t, blockFileErr)
	defer closeOpenFile(blockFile)
	filePath := blockFile.segmentPath(false, "0002.fdb")
	fileOpt, fileErr := os.Create(filePath)
	assert.Nil(t, fileErr)
	defer fileOpt.Close()
	fileOpt.WriteString("block 0")
	time.Sleep(2 * time.Second)
	removed, removedErr := blockFile.TryRemoveFile("0002.fdb", false)
	assert.Nil(t, removedErr)
	assert.False(t, removed)
}
