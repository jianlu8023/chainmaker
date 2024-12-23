/*
Copyright (C) BABEC. All rights reserved.


SPDX-License-Identifier: Apache-2.0
*/

// Package interfaces define file interface
package interfaces

import (
	"errors"
	"fmt"
	"os"
	"strconv"

	storePb "chainmaker.org/chainmaker/pb-go/v2/store"
	"chainmaker.org/chainmaker/protocol/v2"
)

// MemCompressor 内存压缩模拟器
type MemCompressor struct {
	compressRecord map[string]string
}

// CompressFile 压缩文件
func (c *MemCompressor) CompressFile(fileDir, fileName,
	compressDir string) error {
	//originalFile := fmt.Sprintf("%s/%s",fileDir,fileName)
	compressedPath := fmt.Sprintf("%s/%s", compressDir, fileName)

	c.compressRecord[fileName] = compressedPath
	return nil
}

// DeCompressFile 解压文件
func (c *MemCompressor) DeCompressFile(compressDir string, compressFileName string,
	deCompressDir string) (string, error) {
	decompressPath := fmt.Sprintf("%s/%s", deCompressDir, compressFileName)
	return decompressPath, nil
}

// MemBinlog 内存文件模拟器
type MemBinlog struct {
	mem           map[uint64][]byte
	last          uint64
	log           protocol.Logger
	compressDir   string
	decompressDir string
	path          string
	compressor    MemCompressor
}

// ReadFileSection 根据索引，读取对应数据
func (l *MemBinlog) ReadFileSection(isDeCompressed bool, fiIndex *storePb.StoreInfo) ([]byte, error) {
	i, _ := strconv.Atoi(fiIndex.FileName)
	data, found := l.mem[uint64(i-1)]
	if !found {
		return nil, errors.New("not found")
	}
	return data[fiIndex.Offset : fiIndex.Offset+fiIndex.ByteLen], nil
}

// NewMemBinlog 创建一个内存binlog
func NewMemBinlog(log protocol.Logger) *MemBinlog {
	memCompressor := MemCompressor{
		compressRecord: make(map[string]string),
	}
	_ = os.Mkdir("./test_datas", os.ModePerm)
	_ = os.Mkdir("./test_datas/c", os.ModePerm)
	_ = os.Mkdir("./test_datas/d", os.ModePerm)
	return &MemBinlog{
		mem:           make(map[uint64][]byte),
		last:          0,
		log:           log,
		compressor:    memCompressor,
		path:          "./test_datas",
		compressDir:   "./test_datas/c",
		decompressDir: "./test_datas/d",
	}
}

// GetCanCompressHeight 根据文件计算可压缩的最大高度
func (l *MemBinlog) GetCanCompressHeight() uint64 {
	if l.last <= 1 {
		return 0
	}
	return l.last - 2
	//return 0
}

// Close 关闭
func (l *MemBinlog) Close() error {
	l.mem = make(map[uint64][]byte)
	l.last = 0
	_ = os.RemoveAll("./test_datas") // 清理数据
	return nil
}

// TruncateFront 指定index 清空
func (l *MemBinlog) TruncateFront(index uint64) error {
	return nil
}

// ReadLastSegSection 读取最后的segment
func (l *MemBinlog) ReadLastSegSection(index uint64) ([]byte, string, uint64, uint64, error) {
	return l.mem[index-1], fmt.Sprintf("%d", index), 0, uint64(len(l.mem[index])), nil
}

// LastIndex 返回最后写入的索引
func (l *MemBinlog) LastIndex() (uint64, error) {
	l.log.Debugf("get last index %d", l.last)
	return l.last, nil
}

// Write 写一个区块到内存中
func (l *MemBinlog) Write(index uint64, data []byte) (fileName string,
	offset, blkLen uint64, start, end uint64, storeDB bool, err error) {
	if index != l.last+1 {
		return "", 0, 0, 0, 0, false, errors.New("binlog out of order")
	}
	l.mem[index-1] = data
	l.last = index
	l.log.Debugf("write binlog index=%d,offset=%d,len=%d", index, 0, len(data))
	return fmt.Sprintf("%d", index), 0, uint64(len(data)), index, index, true, nil
}

// DeCompressFile 解压缩文件
func (l *MemBinlog) DeCompressFile(compressFileName string) (string, error) {
	return l.compressor.DeCompressFile(l.compressDir, compressFileName, l.decompressDir)
}

// CheckDecompressFileExist 检查解压缩文件是否存在
func (l *MemBinlog) CheckDecompressFileExist(filePath string) (bool, int64, error) {
	return true, 0, nil
}

// CompressFileByStartHeight 压缩指定高度下文件
func (l *MemBinlog) CompressFileByStartHeight(startHeight uint64) (string, error) {
	fileName := fmt.Sprintf("%d", startHeight+1)

	return fileName, l.compressor.CompressFile(l.path, fileName, l.compressDir)
}

// TryRemoveFile 删除文件
func (l *MemBinlog) TryRemoveFile(fileName string, isDeCompressed bool) (bool, error) {
	return true, nil
}
