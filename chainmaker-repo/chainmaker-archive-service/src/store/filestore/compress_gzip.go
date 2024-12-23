/*
Copyright (C) BABEC. All rights reserved.


SPDX-License-Identifier: Apache-2.0
*/

// Package filestore define file operation
package filestore

import (
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
)

// CompressGzip gzip压缩
type CompressGzip struct {
	suffix string //gzip
}

// CompressFile 压缩指定文件到压缩文件夹下
// @receiver cmg
// @param fileDir
// @param fileName
// @param compressDir
// @return error
func (cmg *CompressGzip) CompressFile(fileDir, fileName, compressDir string) error {
	fullFileName := fmt.Sprintf("%s/%s", fileDir, fileName)
	fileRead, fileReadError := os.Open(fullFileName)
	if fileReadError != nil {
		return fileReadError
	}
	zipFilePath := fmt.Sprintf("%s/%s.%s", compressDir, fileName, cmg.suffix)
	zipFile, zipFileErr := os.Create(zipFilePath)
	if zipFileErr != nil {
		return zipFileErr
	}
	defer zipFile.Close()
	zipWriter, zipWriteError := gzip.NewWriterLevel(zipFile, gzip.BestCompression)
	if zipWriteError != nil {
		return zipWriteError
	}
	defer zipWriter.Close()
	_, writeError := io.Copy(zipWriter, fileRead)
	if writeError != nil {
		return writeError
	}
	flushError := zipWriter.Flush()
	if flushError != nil {
		return flushError
	}
	return nil
}

// DeCompressFile 将压缩文件解压缩到指定文件夹下
// @receiver cmg
// @param compressDir
// @param compressFileName
// @param unCompressDir
// @return string
// @return error
func (cmg *CompressGzip) DeCompressFile(compressDir, compressFileName, unCompressDir string) (string, error) {
	dotSuffix := fmt.Sprintf(".%s", cmg.suffix)
	if !strings.HasSuffix(compressFileName, dotSuffix) {
		return "", errors.New("compressFileName not end with " + dotSuffix)
	}
	outputFileName := fmt.Sprintf("%s/%s", unCompressDir, strings.TrimSuffix(compressFileName, dotSuffix))
	outFile, outFileError := os.Create(outputFileName)
	if outFileError != nil {
		return "", outFileError
	}
	defer outFile.Close()
	compressFullName := fmt.Sprintf("%s/%s", compressDir, compressFileName)
	compressedFile, compressedError := os.Open(compressFullName)
	if compressedError != nil {
		return "", compressedError
	}
	defer compressedFile.Close()
	zipReader, zipReaderError := gzip.NewReader(compressedFile)
	if zipReaderError != nil {
		return "", zipReaderError
	}
	defer zipReader.Close()
	//nolint
	_, copyError := io.Copy(outFile, zipReader)
	if copyError != nil {
		return "", copyError
	}
	syncError := outFile.Sync()
	if syncError != nil {
		return "", syncError
	}
	return outputFileName, nil
}
