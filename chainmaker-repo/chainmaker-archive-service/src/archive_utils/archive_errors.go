/*
Copyright (C) BABEC. All rights reserved.


SPDX-License-Identifier: Apache-2.0
*/

// Package archive_utils define common utils
package archive_utils

import "errors"

var (
	// MsgRegisterBlockNil register error message
	MsgRegisterBlockNil = "register request is null"
	// CodeRegisterBlockNil 注册区块非法
	CodeRegisterBlockNil = 2
	// CodeRegisterNoConfig 无法从区块中解析出配置信息
	CodeRegisterNoConfig = 3
	// CodeBlockHashVerifyFail 区块hash验证失败
	CodeBlockHashVerifyFail = 4
	// ErrorBlockHashVerifyFailed 区块hash验证失败
	ErrorBlockHashVerifyFailed = errors.New("block hash not equal block.hash")
	// MsgRegisterConflict 同时注册链信息错误
	MsgRegisterConflict = "other chain node is registering ,try later"
	// CodeSaveRegisterInfoToDBFail 保存注册信息到数据库失败
	CodeSaveRegisterInfoToDBFail = 5
	// ErrorSaveRegisterInfoToDBFail 保存注册信息到数据库失败
	ErrorSaveRegisterInfoToDBFail = errors.New("save register info to db error")

	// CodeArchiveRecvError archive blocks error message
	CodeArchiveRecvError = 6
	// MsgArchiveBlockNil 区块非法
	MsgArchiveBlockNil = "illegal block"
	// CodeArchiveBlockNil 区块非法
	CodeArchiveBlockNil = 7
	// CodeArchiveBlockError 归档错误
	CodeArchiveBlockError = 8
	// ErrorArchiveHashIllegal hash不匹配
	ErrorArchiveHashIllegal = errors.New("previous block hash not equal last block hash")

	// CodeGetArchiveStatusFail get archived status error
	CodeGetArchiveStatusFail = 9

	// CodeGetArchivedHeightFail get range blocks
	CodeGetArchivedHeightFail = 10

	// CodeBlockExistsFailed get block by hash error
	CodeBlockExistsFailed = 11
	// CodeBlockByHashFailed 查询失败
	CodeBlockByHashFailed = 12
	// CodeHeightByHashFailed 查询失败
	CodeHeightByHashFailed = 13
	// CodeHashInvalidParam 参数错误
	CodeHashInvalidParam = 14

	// CodeHeaderByHeightFailed get block by height error
	CodeHeaderByHeightFailed = 15
	// CodeBlockByHeightFailed 参数错误
	CodeBlockByHeightFailed = 16
	// CodeHeightInvalidParam 参数错误
	CodeHeightInvalidParam = 17

	// CodeBlockByTxIdFailed get block by txid error
	CodeBlockByTxIdFailed = 18
	// ErrorBlockByTxId get block by txid error
	ErrorBlockByTxId = errors.New("got no block by txid")
	// CodeTxRWSetByTxIdFailed get tx rwset by txid error
	CodeTxRWSetByTxIdFailed = 19

	// CodeTxByTxIdFailed  get tx by txid error
	CodeTxByTxIdFailed = 20

	// CodeLastConfigFailed get last config block
	CodeLastConfigFailed = 21

	// CodeTxExistsByTxIdFailed get tx detail by txid error
	CodeTxExistsByTxIdFailed = 22
	// CodeTxHeightByTxIdFailed 根据txid查询height错误
	CodeTxHeightByTxIdFailed = 23
	// CodeTxConfirmedByTxIdFailed 交易确认时间错误
	CodeTxConfirmedByTxIdFailed = 24
	// CodeTxInvalidParam 参数错误
	CodeTxInvalidParam = 25

	// CodeHttpInvalidParameter http parameter invalid
	CodeHttpInvalidParameter = 26
	// MsgHttpInvalidParameter 参数错误
	MsgHttpInvalidParameter = "invalid parameter"

	// CodeHttpGetRangeBlocksFailed http GetRangeBlocks error
	CodeHttpGetRangeBlocksFailed = 27

	// CodeGetLastBlockFailed get last block error
	CodeGetLastBlockFailed = 28

	// CodeGetLastSavePointFailed get last save point error
	CodeGetLastSavePointFailed = 29

	// CodeCompressUnderHeightFailed compress under height error
	CodeCompressUnderHeightFailed = 30

	// CodeGetCompressStatusFailed get compress status error
	CodeGetCompressStatusFailed = 31

	// CodeCAFileFailed addca status error
	CodeCAFileFailed = 32

	// CodeGetConfigByHeightFailed 根据块高查询链配置信息错误
	CodeGetConfigByHeightFailed = 33

	// CodeHexDecodeFailed 无法根据hex解码出byte字符串
	CodeHexDecodeFailed = 34

	// ErrorGetBufPool 无法取到缓存了
	ErrorGetBufPool = errors.New("get object from pool error")
	// ErrorChainInCompress 当前链正在进行压缩
	ErrorChainInCompress = errors.New("chain is compressing,retry later")

	// CodeRangeTooLarge 范围过大
	CodeRangeTooLarge = 35
	// ErrorRangeTooLarge 范围过大
	ErrorRangeTooLarge = "block range must smaller than 6"

	// ErrorMerklePath 计算merklepath错误
	ErrorMerklePath = 36
)
