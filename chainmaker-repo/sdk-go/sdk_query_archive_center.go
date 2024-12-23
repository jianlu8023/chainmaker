/*
Copyright (C) BABEC. All rights reserved.
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package chainmaker_sdk_go

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"reflect"
	"time"

	"chainmaker.org/chainmaker/pb-go/v2/common"
	"chainmaker.org/chainmaker/pb-go/v2/config"
	"chainmaker.org/chainmaker/pb-go/v2/store"
)

// ArchiveCenterResponse 定义了归档中心通用返回头信息
type ArchiveCenterResponse struct {
	Code     int    `json:"code"` // 错误码,0代表成功.其余代表失败
	ErrorMsg string `json:"errorMsg"`
	//Data     interface{} `json:"data"`
}

// ArchiveCenterResponseTransaction 定义交易返回结构
type ArchiveCenterResponseTransaction struct {
	ArchiveCenterResponse
	Data *common.TransactionInfo `json:"data"`
}

// ArchiveCenterResponseTransactionWithRwSet 定义读写集交易
type ArchiveCenterResponseTransactionWithRwSet struct {
	ArchiveCenterResponse
	Data *common.TransactionInfoWithRWSet `json:"data"`
}

// ArchiveCenterResponseBlockInfo 定义区块返回结构
type ArchiveCenterResponseBlockInfo struct {
	ArchiveCenterResponse
	Data *common.BlockInfo `json:"data"`
}

// ArchiveCenterResponseBlockWithRwSet 定义带读写集区块
type ArchiveCenterResponseBlockWithRwSet struct {
	ArchiveCenterResponse
	Data *store.BlockWithRWSet `json:"data"`
}

// ArchiveCenterResponseHeight 定义块高返回结构
type ArchiveCenterResponseHeight struct {
	ArchiveCenterResponse
	Data *uint64 `json:"data"`
}

// ArchiveCenterResponseBlockHeader 定义区块头返回结构
type ArchiveCenterResponseBlockHeader struct {
	ArchiveCenterResponse
	Data *common.BlockHeader `json:"data"`
}

// ArchiveCenterResponseChainConfig 定义链配置返回结构
type ArchiveCenterResponseChainConfig struct {
	ArchiveCenterResponse
	Data *config.ChainConfig `json:"data"`
}

// ArchiveCenterResponseMerklePath 定义merkle path 返回结构
type ArchiveCenterResponseMerklePath struct {
	ArchiveCenterResponse
	Data []byte `json:"data"`
}

// ArchiveCenterHttpConfig 定义归档中心配置
type ArchiveCenterHttpConfig struct {
	ChainGenesisHash     string //
	ArchiveCenterHttpUrl string
	ReqeustSecondLimit   int // http请求的超时间隔,默认5秒
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
	// httpRequestDuration 默认http请求超时秒数
	httpRequestDuration = 5
)

var (
	// archiveCenterApiGetBlockHeaderByHeight 根据高度查区块头接口
	archiveCenterApiGetBlockHeaderByHeight = "get_block_header_by_height"
	// archiveCenterApiGetBlockHeightByHash 根据区块hash查块高接口
	archiveCenterApiGetBlockHeightByHash = "get_height_by_hash"
	// archiveCenterApiGetBlockHeightByTxId 根据txid查询块高接口
	archiveCenterApiGetBlockHeightByTxId = "get_height_by_tx_id"
	// archiveCenterApiGetFullBlockByHeight 根据块高查全区块
	archiveCenterApiGetFullBlockByHeight = "get_full_block_by_height"
	// archiveCenterApiGetBlockWithTxRWSetByHash 根据hash查读写集区块接口
	archiveCenterApiGetBlockWithTxRWSetByHash = "get_block_info_by_hash"
	// archiveCenterApiGetBlockWithTxRWSetByTxId 根据txid查读写集区块接口
	archiveCenterApiGetBlockWithTxRWSetByTxId = "get_block_info_by_txid"
	// archiveCenterApiGetBlockWithTxRWSetByHeight 根据高度查读写集区块接口
	archiveCenterApiGetBlockWithTxRWSetByHeight = "get_block_info_by_height"
	// archiveCenterApiGetCommonTransactionByTxId 根据txid查交易接口
	archiveCenterApiGetCommonTransactionByTxId = "get_transaction_info_by_txid"
	// archiveCenterApiGetCommonTransactionWithRWSetByTxId 根据txid查读写集交易接口
	archiveCenterApiGetCommonTransactionWithRWSetByTxId = "get_full_transaction_info_by_txid"
	// archiveCenterApiGetConfigByHeight 根据块高查询最近链配置信息接口
	archiveCenterApiGetConfigByHeight = "get_chainconfig_by_height"
	// archiveCenterApiGetMerklePathByTxId 根据txid查询merkle path
	archiveCenterApiGetMerklePathByTxId = "get_merklepath_by_txid"
	// archiveCenterApiGetTruncateTxByTxId 根据txid查询截断交易信息
	archiveCenterApiGetTruncateTxByTxId = "get_truncate_tx_by_txid"
	// archiveCenterApiGetTruncateBlockByHeight 根据高度查询截断区块信息
	archiveCenterApiGetTruncateBlockByHeight = "get_truncate_block_by_height"
)

func (cc *ChainClient) httpQueryArchiveCenter(apiUri string,
	queryParam *ArchiveCenterQueryParam,
	responseT reflect.Type) (interface{}, error) {
	cfg := cc.ArchiveCenterConfig()
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
		cc.logger.Errorf("httpQueryArchiveCenter api[%s] NewRequest got error [%s]",
			apiUri, reqErr.Error())
		return nil, reqErr
	}
	// 请求http接口
	resp, respErr := http.DefaultClient.Do(req)
	if respErr != nil {
		cc.logger.Errorf("httpQueryArchiveCenter api[%s] do request error [%s]",
			apiUri, respErr.Error())
		return nil, respErr
	}
	if resp == nil {
		cc.logger.Warnf("httpQueryArchiveCenter api[%s] got nothing", apiUri)
		return nil, nil
	}
	defer resp.Body.Close()
	// 读取接口返回信息
	respBytes, respBytesErr := ioutil.ReadAll(resp.Body)
	if respBytesErr != nil {
		cc.logger.Errorf("httpQueryArchiveCenter api[%s] read resp body error [%s]",
			apiUri, respBytesErr.Error())
		return nil, respBytesErr
	}
	responseV := reflect.New(responseT)
	responseP := responseV.Interface()
	unMarshalErr := json.Unmarshal(respBytes, responseP)
	if unMarshalErr != nil {
		cc.logger.Errorf("httpQueryArchiveCenter api[%s] unMarshal error [%s], origin resp [%s]",
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
		cc.logger.Warnf("httpQueryArchiveCenter api[%s] query got code[%d] error[%s] ",
			apiUri, code, msg)
		return nil, fmt.Errorf("code[%d] msg[%s]", code, msg)
	}
	// 无有效数据
	if dataIsNil {
		cc.logger.Warnf("httpQueryArchiveCenter api[%s] query got nothing", apiUri)
		return nil, nil
	}
	return data, nil
}

func getApiUrl(apiUrl, baseUrl string) string {
	return fmt.Sprintf("%s/%s", baseUrl, apiUrl)
}
