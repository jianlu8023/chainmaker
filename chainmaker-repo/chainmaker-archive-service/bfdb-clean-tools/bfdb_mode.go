/*
Copyright (C) BABEC. All rights reserved.


SPDX-License-Identifier: Apache-2.0
*/

package main

import (
	"bytes"
	"context"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"time"

	"chainmaker.org/chainmaker/pb-go/v2/store"
)

// BfdbFileInfo bfdb文件信息
type BfdbFileInfo struct {
	FileName      string
	BeginHeight   uint64
	EndHeight     uint64
	CanClean      string
	ConfigHeights []uint64
}

var (
	headers = []string{"fileName", "beginHeight", "endHeight", "canClean", "configHeights"}
)

// createCSVFile 创建csv文件
// @param fileName
// @return File
// @return Writer
func createCSVFile(fileName string) (*os.File, *csv.Writer) {
	csvF, err := os.Create(fileName)
	if err != nil {
		panic("create error " + err.Error())
	}
	csvW := csv.NewWriter(csvF)
	headerErr := csvW.Write(headers)
	if headerErr != nil {
		csvF.Close()
		panic("write header error " + headerErr.Error())
	}
	return csvF, csvW
}

// openCSVFile 打开csv文件
// @param fileName
// @return File
// @return Reader
func openCSVFile(fileName string) (*os.File, *csv.Reader) {
	_, err := os.Stat(fileName)
	if err != nil {
		panic(err.Error())
	}
	csvF, csvE := os.Open(fileName)
	if csvE != nil {
		panic("open csv error " + csvE.Error())
	}
	csvR := csv.NewReader(csvF)
	_, readErr := csvR.Read()
	if readErr != nil {
		csvF.Close()
		panic("read csv error " + readErr.Error())
	}
	return csvF, csvR
}

// writeFileRecord 写入一条记录到csv
// @param record
// @param Writer
func writeFileRecord(record *BfdbFileInfo, writer *csv.Writer) {
	var configs []string
	for _, height := range record.ConfigHeights {
		configs = append(configs, strconv.FormatUint(height, 10))
	}
	var configStr string
	if len(configs) > 0 {
		configStr = strings.Join(configs, ";")
	}
	writeErr := writer.Write([]string{record.FileName, strconv.FormatUint(record.BeginHeight, 10),
		strconv.FormatUint(record.EndHeight, 10), record.CanClean, configStr})
	if writeErr != nil {
		GlobalLogger.Errorf("writeFileRecord record %+v got error %s",
			*record, writeErr.Error())
		panic("writeFileRecord error : " + writeErr.Error())
	}
	writer.Flush()
}

// readFileRecord 从csv中读取一条记录
// @param reader
// @return BfdbFileInfo
func readFileRecord(reader *csv.Reader) *BfdbFileInfo {
	records, recordsErr := reader.Read()
	if recordsErr != nil {
		if recordsErr == io.EOF {
			return nil
		}
		panic("readFileRecord got error : " + recordsErr.Error())
	}
	var ret BfdbFileInfo
	if len(records) != 5 {
		panic("readFileRecord records length failed , " + strings.Join(records, "/"))
	}
	ret.FileName = records[0]
	ret.BeginHeight, _ = strconv.ParseUint(records[1], 10, 64)
	ret.EndHeight, _ = strconv.ParseUint(records[2], 10, 64)
	ret.CanClean = records[3]
	heightArrs := strings.Split(records[4], ";")
	for _, heightStr := range heightArrs {
		height, _ := strconv.ParseUint(heightStr, 10, 64)
		ret.ConfigHeights = append(ret.ConfigHeights, height)
	}
	return &ret
}

// ArchiveCenterResponse 定义了归档中心通用返回头信息
type ArchiveCenterResponse struct {
	Code     int    `json:"code"` // 错误码,0代表成功.其余代表失败
	ErrorMsg string `json:"errorMsg"`
	//Data     interface{} `json:"data"`
}

// ArchiveCenterResponseBlockWithRwSet 定义带读写集区块
type ArchiveCenterResponseBlockWithRwSet struct {
	ArchiveCenterResponse
	Data *store.BlockWithRWSet `json:"data"`
}

// ArchiveCenterQueryParam 定义归档中心通用请求头
type ArchiveCenterQueryParam struct {
	ChainGenesisHash string `json:"chain_genesis_hash,omitempty"`
	Start            uint64 `json:"start,omitempty"`
	End              uint64 `json:"end,omitempty"`
	BlockHash        string `json:"block_hash,omitempty"`
	Height           uint64 `json:"height,omitempty"`
	TxId             string `json:"tx_id,omitempty"`
	WithRwSet        bool   `json:"with_rwset,omitempty"`
	TruncateLength   int    `json:"truncate_length,omitempty"`
	TruncateModel    string `json:"truncate_model,omitempty"`
}

var (
	// archiveCenterApiGetFullBlockByHeight 根据块高查全区块
	archiveCenterApiGetFullBlockByHeight = "get_full_block_by_height"
	archiveCenterApiGetArchiveStatus     = "get_archive_status"
)

// ArchiveCenterArchiveStatus query archive status response
type ArchiveCenterArchiveStatus struct {
	ArchiveCenterResponse
	Data map[string]interface{} `json:"data"`
}

func httpQueryArchiveCenter(apiUri string,
	queryParam *ArchiveCenterQueryParam,
	responseT reflect.Type) (interface{}, error) {
	cfg := &GlobalServerCFG.ArchiveCFG
	if cfg == nil || len(cfg.ArchiveCenterHttpUrl) == 0 {
		return nil, nil
	}
	queryParam.ChainGenesisHash = cfg.ChainGenesisHash // 这里设置一下genesisBlockHash
	// 构造请求body
	reqBody, _ := json.Marshal(queryParam)
	requestUri := getApiUrl(apiUri, cfg.ArchiveCenterHttpUrl)
	ctx, ctxCancel := context.WithTimeout(context.Background(),
		time.Duration(cfg.ReqeustSecondLimit)*time.Second)
	defer ctxCancel()
	// 构造http请求
	req, reqErr := http.NewRequestWithContext(ctx,
		http.MethodPost, requestUri, bytes.NewReader(reqBody))
	if reqErr != nil {
		GlobalLogger.Errorf("httpQueryArchiveCenter api[%s] NewRequest got error [%s]",
			apiUri, reqErr.Error())
		return nil, reqErr
	}
	// 请求http接口
	resp, respErr := http.DefaultClient.Do(req)
	if respErr != nil {
		GlobalLogger.Errorf("httpQueryArchiveCenter api[%s] do request error [%s]",
			apiUri, respErr.Error())
		return nil, respErr
	}
	if resp == nil {
		GlobalLogger.Warnf("httpQueryArchiveCenter api[%s] got nothing", apiUri)
		return nil, nil
	}
	defer resp.Body.Close()
	// 读取接口返回信息
	respBytes, respBytesErr := ioutil.ReadAll(resp.Body)
	if respBytesErr != nil {
		GlobalLogger.Errorf("httpQueryArchiveCenter api[%s] read resp body error [%s]",
			apiUri, respBytesErr.Error())
		return nil, respBytesErr
	}
	responseV := reflect.New(responseT)
	responseP := responseV.Interface()
	unMarshalErr := json.Unmarshal(respBytes, responseP)
	if unMarshalErr != nil {
		GlobalLogger.Errorf("httpQueryArchiveCenter api[%s] unMarshal error [%s], origin resp [%s]",
			apiUri, unMarshalErr.Error(), string(respBytes))
		return nil, unMarshalErr
	}
	// 解析接口返回数据
	code := responseV.Elem().FieldByName("Code").Int()
	// 读取错误信息
	msg := responseV.Elem().FieldByName("ErrorMsg").String()
	// 读取业务数据
	data := responseV.Elem().FieldByName("Data").Interface()
	// 判断是否有有效数据
	dataIsNil := responseV.Elem().FieldByName("Data").IsNil()
	// 查询错误
	if code > 0 {
		GlobalLogger.Warnf("httpQueryArchiveCenter api[%s] query got code[%d] error[%s] ",
			apiUri, code, msg)
		return nil, fmt.Errorf("code[%d] msg[%s]", code, msg)
	}
	// 无有效数据
	if dataIsNil {
		GlobalLogger.Warnf("httpQueryArchiveCenter api[%s] query got nothing", apiUri)
		return nil, nil
	}
	return data, nil
}

func getApiUrl(apiUrl, baseUrl string) string {
	return fmt.Sprintf("%s/%s", baseUrl, apiUrl)
}

func getArchiveStatus() (uint64, error) {
	httpResp, httpRespErr := httpQueryArchiveCenter(archiveCenterApiGetArchiveStatus,
		&ArchiveCenterQueryParam{}, reflect.TypeOf(ArchiveCenterArchiveStatus{}))
	if httpRespErr != nil {
		GlobalLogger.Infof("getArchiveStatus got error %s",
			httpRespErr.Error())
		return 0, httpRespErr
	}
	if httpResp == nil {
		GlobalLogger.Infof("getArchiveStatus got nothing")
		return 0, nil
	}

	blockHeightT, blockOK := httpResp.(map[string]interface{})
	// 归档中心查到有效数据
	if blockOK && blockHeightT != nil {
		GlobalLogger.Infof("archiveStatus [%+v]", blockHeightT)
		height := blockHeightT["height"]

		return uint64(height.(float64)), nil
	}
	return 0, nil
}

func getBlockWithRWSetByHeight(blockHeight uint64) (*store.BlockWithRWSet, error) {
	httpResp, httpRespErr := httpQueryArchiveCenter(archiveCenterApiGetFullBlockByHeight,
		&ArchiveCenterQueryParam{
			Height: blockHeight,
		}, reflect.TypeOf(ArchiveCenterResponseBlockWithRwSet{}))
	if httpRespErr != nil {
		GlobalLogger.Infof("getBlockWithRWSetByHeight height [%d] got error %s",
			blockHeight, httpRespErr.Error())
		return nil, httpRespErr
	}
	if httpResp == nil {
		GlobalLogger.Infof("getBlockWithRWSetByHeight height [%d] got nothing")
		return nil, nil
	}

	block, blockOK := httpResp.(*store.BlockWithRWSet)
	// 归档中心查到有效数据
	if blockOK {
		return block, nil
	}
	return nil, nil
}

func getAllConfigsUnderHeight(blockHeight uint64) ([]uint64, error) {
	block, blockError := getBlockWithRWSetByHeight(blockHeight)
	if blockError != nil {
		return nil, blockError
	}
	if block == nil {
		return nil, fmt.Errorf("getAllConfigsUnderHeight got height [%d] nothing",
			blockHeight)
	}
	var retHeights []uint64
	if IsConfBlock(block.Block) {
		retHeights = append(retHeights, blockHeight)
	}
	nextHeight := block.Block.Header.PreConfHeight
	for nextHeight > 0 {
		tempBlock, tempBlockError := getBlockWithRWSetByHeight(nextHeight)
		if tempBlockError != nil {
			GlobalLogger.Errorf("getAllConfigsUnderHeight height [%d] error %s",
				nextHeight, tempBlockError.Error())
			return retHeights, tempBlockError
		}
		if tempBlock == nil {
			return nil, fmt.Errorf("getAllConfigsUnderHeight got height [%d] nothing",
				blockHeight)
		}
		retHeights = append(retHeights, nextHeight)
		nextHeight = tempBlock.Block.Header.PreConfHeight
	}
	retHeights = append(retHeights, nextHeight)
	fmt.Printf("getAllConfigsUnderHeight under height [%d],got confs %+v \n",
		blockHeight, retHeights)
	GlobalLogger.Infof("getAllConfigsUnderHeight under height [%d],got confs %+v",
		blockHeight, retHeights)
	return retHeights, nil
}

// scanBfdbPathDir 扫描文件夹下的所有文件,输出
func scanBfdbPathDir(pathDir string) (map[uint64]uint64,
	[]uint64, uint64, error) {
	dires, diresErr := os.ReadDir(pathDir)
	if diresErr != nil {
		GlobalLogger.Errorf("scanBfdbPathDir path [%s] error %s",
			pathDir, diresErr.Error())
		return nil, nil, 0,
			errors.New("runScanFiles ReadDir error " + diresErr.Error())
	}
	var endBFDBName uint64
	nameEndMp := make(map[uint64]uint64)
	nameArrays := make([]uint64, 0, len(dires))
	for _, dir := range dires {
		tempName := dir.Name()
		GlobalLogger.Debugf("scanBfdbPathDir dirName %s ", tempName)
		if strings.HasSuffix(tempName, ".fdb") {
			nameId, nameIdErr := strconv.ParseUint(strings.TrimSuffix(tempName, ".fdb"), 10, 64)
			if nameIdErr != nil {
				GlobalLogger.Errorf("scanBfdbPathDir [%s] parseUint error %s",
					tempName, nameIdErr.Error())
				return nameEndMp, nameArrays, endBFDBName,
					errors.New("scan dir name " + tempName + nameIdErr.Error())
			}
			nameArrays = append(nameArrays, nameId)
		} else if strings.HasSuffix(tempName, ".fdb.END") {
			nameId, nameIdErr := strconv.ParseUint(strings.TrimSuffix(tempName, ".fdb.END"), 10, 64)
			if nameIdErr != nil {
				GlobalLogger.Errorf("scanBfdbPathDir [%s] parseUint error %s",
					tempName, nameIdErr.Error())
				return nameEndMp, nameArrays, endBFDBName,
					errors.New("scan dir name " + tempName + nameIdErr.Error())
			}
			nameArrays = append(nameArrays, nameId)
			endBFDBName = nameId
		} else {
			continue
		}
	}
	sort.Slice(nameArrays, func(i, j int) bool {
		return nameArrays[i] < nameArrays[j]
	})
	for k := 0; k < len(nameArrays)-1; k++ {
		nameEndMp[nameArrays[k]] = nameArrays[k+1] - 2
	}
	nameArrays = nameArrays[:len(nameArrays)-1]
	//GlobalLogger.Debugf("nameArrays [%+v],endName %d,mp [%+v]",
	//	nameArrays, endBFDBName, nameEndMp)
	return nameEndMp, nameArrays, endBFDBName, nil
}
