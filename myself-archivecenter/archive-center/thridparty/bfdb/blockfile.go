/*
Copyright (C) BABEC. All rights reserved.


SPDX-License-Identifier: Apache-2.0
*/

package bfdb

import (
	commonPb "chainmaker.org/chainmaker/pb-go/v2/common"
	"chainmaker.org/chainmaker/pb-go/v2/syscontract"
)

// IsConfigTx 配置信息
// @param tx
// @return bool
func IsConfigTx(tx *commonPb.Transaction) bool {
	if tx == nil || tx.Payload == nil {
		return false
	}
	return tx.Payload.ContractName == syscontract.SystemContract_CHAIN_CONFIG.String()
}

// IsValidConfigTx 是否为配置交易
// @param tx
// @return bool
func IsValidConfigTx(tx *commonPb.Transaction) bool {
	if !IsConfigTx(tx) {
		return false
	}

	if tx.Result == nil || tx.Result.Code != commonPb.TxStatusCode_SUCCESS ||
		tx.Result.ContractResult == nil || tx.Result.ContractResult.Result == nil {
		return false
	}

	return true
}

// IsConfBlock 是否为配置区块
// @param block
// @return bool
func IsConfBlock(block *commonPb.Block) bool {
	if block == nil || len(block.Txs) == 0 {
		return false
	}
	tx := block.Txs[0]
	return IsValidConfigTx(tx)
}
