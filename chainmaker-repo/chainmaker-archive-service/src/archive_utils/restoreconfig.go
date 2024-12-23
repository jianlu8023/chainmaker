/*
Copyright (C) BABEC. All rights reserved.


SPDX-License-Identifier: Apache-2.0
*/

// Package archive_utils define common utils
package archive_utils

import (
	"errors"

	commonPb "chainmaker.org/chainmaker/pb-go/v2/common"
	configPb "chainmaker.org/chainmaker/pb-go/v2/config"
	"chainmaker.org/chainmaker/utils/v2"
	"github.com/gogo/protobuf/proto"
)

var (
	// ErrorNoConfigBlock 非配置区块错误
	ErrorNoConfigBlock = errors.New("block not config")
	// ErrorCodeNoConfigBlock 非配置区块错误码字
	ErrorCodeNoConfigBlock = 50002
)

// RestoreChainConfigFromBlock 从配置区块中解析区块配置信息
func RestoreChainConfigFromBlock(block *commonPb.Block) (*configPb.ChainConfig, error) {
	if block == nil ||
		!utils.IsConfBlock(block) {
		return nil, ErrorNoConfigBlock
	}
	var retConfig configPb.ChainConfig
	unMarshalError := proto.Unmarshal(block.Txs[0].Result.ContractResult.Result, &retConfig)
	if unMarshalError != nil {
		return nil, unMarshalError
	}
	return &retConfig, nil
}
