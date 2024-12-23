/*
Copyright (C) BABEC. All rights reserved.


SPDX-License-Identifier: Apache-2.0
*/

// Package filestore define file operation
package filestore

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
)

// Compress7z 7z 压缩
type Compress7z struct {
	suffix  string // 7z
	maxTime int    // 秒级别
}

// CompressFile 压缩指定文件到压缩文件夹下
// @receiver cmg
// @param fileDir
// @param fileName
// @param compressDir
// @return error
func (cmg *Compress7z) CompressFile(fileDir, fileName, compressDir string) error {
	fullFileName := fmt.Sprintf("%s/%s", fileDir, fileName)
	zipFilePath := fmt.Sprintf("%s/%s.%s", compressDir, fileName, cmg.suffix)
	deadlineContext, cancel := context.WithTimeout(context.Background(), time.Duration(cmg.maxTime)*time.Second)
	defer cancel()
	zipCommand := exec.CommandContext(deadlineContext, "7z", "a", zipFilePath, fullFileName)
	zipError := zipCommand.Run()
	if zipError != nil {
		os.Remove(zipFilePath)
		return zipError
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
func (cmg *Compress7z) DeCompressFile(compressDir, compressFileName, unCompressDir string) (string, error) {
	dotSuffix := fmt.Sprintf(".%s", cmg.suffix)
	if !strings.HasSuffix(compressFileName, dotSuffix) {
		return "", errors.New("compressFileName not end with " + dotSuffix)
	}
	outputFileName := fmt.Sprintf("%s/%s", unCompressDir, strings.TrimSuffix(compressFileName, dotSuffix))
	compressFullName := fmt.Sprintf("%s/%s", compressDir, compressFileName)
	deadlineContext, cancel := context.WithTimeout(context.Background(), time.Duration(cmg.maxTime)*time.Second)
	defer cancel()
	unzipCommand := exec.CommandContext(deadlineContext, "7z", "e", compressFullName, fmt.Sprintf("-o%s", unCompressDir))
	unzipError := unzipCommand.Run()
	if unzipError != nil {
		os.Remove(outputFileName)
		return "", unzipError
	}
	return outputFileName, nil
}
