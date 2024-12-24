/*
Copyright (C) BABEC. All rights reserved.


SPDX-License-Identifier: Apache-2.0
*/

// Package httpserver define http operation
package httpserver

import (
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"net/http"

	"chainmaker.org/chainmaker-archive-service/src/archive_utils"
	"github.com/gin-gonic/gin"
)

// QueryParameter http的查询信息
type QueryParameter struct {
	ChainGenesisHash string `json:"chain_genesis_hash,omitempty"`
	Start            uint64 `json:"start,omitempty"`
	End              uint64 `json:"end,omitempty"`
	BlockHash        string `json:"block_hash,omitempty"`
	ByteBlockHash    []byte
	Height           uint64 `json:"height,omitempty"`
	TxId             string `json:"tx_id,omitempty"`
	WithRwSet        bool   `json:"with_rwset,omitempty"`
	TruncateLength   int    `json:"truncate_length,omitempty"`
	TruncateModel    string `json:"truncate_model,omitempty"`
}

// Response 通用http返回信息
type Response struct {
	Code     int         `json:"code,omitempty"` // 错误码,0代表成功.其余代表失败
	ErrorMsg string      `json:"errorMsg,omitempty"`
	Data     interface{} `json:"data,omitempty"`
}

var (
	caFileName = "ca_name"
)

// GetArchivedHeight 获取当前归档高度
func (srv *HttpSrv) GetArchivedHeight(ginContext *gin.Context) {
	var inParameter QueryParameter
	err := ginContext.BindJSON(&inParameter)
	if err != nil || len(inParameter.ChainGenesisHash) == 0 {
		ginContext.SecureJSON(http.StatusOK, Response{
			Code:     archive_utils.CodeHttpInvalidParameter,
			ErrorMsg: archive_utils.MsgHttpInvalidParameter,
		})
		return
	}
	height, heightErr := srv.ProxyProcessor.GetArchivedHeight(inParameter.ChainGenesisHash,
		srv.ProxyProcessor.GetChainProcessorById)
	if heightErr != nil {
		srv.logger.Errorf("GetArchivedHeight got error %s ", heightErr.Error())
		ginContext.SecureJSON(http.StatusOK, Response{
			Code:     archive_utils.CodeGetArchivedHeightFail,
			ErrorMsg: heightErr.Error(),
		})
		return
	}
	ginContext.SecureJSON(http.StatusOK, Response{
		Data: height,
	})

}

// GetRangeBlocks 批量查询区块接口
func (srv *HttpSrv) GetRangeBlocks(ginContext *gin.Context) {
	var inParameter QueryParameter
	err := ginContext.BindJSON(&inParameter)
	if err != nil || len(inParameter.ChainGenesisHash) == 0 || inParameter.End < inParameter.Start {
		ginContext.SecureJSON(http.StatusOK, Response{
			Code:     archive_utils.CodeHttpInvalidParameter,
			ErrorMsg: archive_utils.MsgHttpInvalidParameter,
		})
		return
	}
	if inParameter.End-inParameter.Start > 5 {
		ginContext.SecureJSON(http.StatusOK, Response{
			Code:     archive_utils.CodeRangeTooLarge,
			ErrorMsg: archive_utils.ErrorRangeTooLarge,
		})
		return
	}
	datas, datasError := srv.ProxyProcessor.GetRangeBlocks(inParameter.ChainGenesisHash,
		inParameter.Start, inParameter.End, srv.ProxyProcessor.GetChainProcessorById)
	if datasError != nil {
		srv.logger.Errorf("GetRangeBlocks start %d , end %d got error %s",
			inParameter.Start, inParameter.End, datasError.Error())
		ginContext.SecureJSON(http.StatusOK, Response{
			Code:     archive_utils.CodeHttpGetRangeBlocksFailed,
			ErrorMsg: datasError.Error(),
		})
		return
	}
	ginContext.SecureJSON(http.StatusOK, Response{
		Data: datas,
	})
}

// GetInArchiveAndArchivedHeight 查询当前归档高度，指定的链是否正在归档
func (srv *HttpSrv) GetInArchiveAndArchivedHeight(ginContext *gin.Context) {
	var inParameter QueryParameter
	err := ginContext.BindJSON(&inParameter)
	if err != nil || len(inParameter.ChainGenesisHash) == 0 {
		ginContext.SecureJSON(http.StatusOK, Response{
			Code:     archive_utils.CodeHttpInvalidParameter,
			ErrorMsg: archive_utils.MsgHttpInvalidParameter,
		})
		return
	}

	height, inArchive, rpcError := srv.ProxyProcessor.GetInArchiveAndArchivedHeight(inParameter.ChainGenesisHash,
		srv.ProxyProcessor.GetChainProcessorById)
	if rpcError != nil {
		srv.logger.Errorf("GetInArchiveAndArchivedHeight error %s ", rpcError.Error())
		ginContext.SecureJSON(http.StatusOK, Response{
			Code:     archive_utils.CodeGetArchiveStatusFail,
			ErrorMsg: rpcError.Error(),
		})
		return
	}
	ginContext.SecureJSON(http.StatusOK, Response{
		Data: map[string]interface{}{
			"height":    height,
			"inArchive": inArchive,
		},
	})
}

// transformHashToByte 优先使用hex解码,如果hex解码错误,那么使用base64进行解码
func (srv *HttpSrv) transformHashToByte(stringHash string) ([]byte, error) {
	blockHash, decodeError := hex.DecodeString(stringHash)
	if decodeError != nil {
		hash, base64DecodeErr := base64.StdEncoding.DecodeString(stringHash)
		if base64DecodeErr != nil {
			return nil, base64DecodeErr
		}
		return hash, nil
	}
	return blockHash, decodeError
}

// ComputeBlockHashhexByHashByte 根据base64编码的区块hash计算hex编码的区块hash
func (srv *HttpSrv) ComputeBlockHashhexByHashByte(ginContext *gin.Context) {
	var inParameter QueryParameter
	err := ginContext.BindJSON(&inParameter)
	if err != nil || len(inParameter.BlockHash) == 0 {
		ginContext.SecureJSON(http.StatusOK, Response{
			Code:     archive_utils.CodeHttpInvalidParameter,
			ErrorMsg: archive_utils.MsgHttpInvalidParameter,
		})
		return
	}
	srv.logger.Debugf("blockhash : %s", inParameter.BlockHash)
	hash, base64DecodeErr := base64.StdEncoding.DecodeString(inParameter.BlockHash)
	if base64DecodeErr != nil {
		ginContext.SecureJSON(http.StatusOK, Response{
			Code:     archive_utils.CodeHttpInvalidParameter,
			ErrorMsg: base64DecodeErr.Error(),
		})
		return

	}
	hexString := hex.EncodeToString(hash)
	ginContext.SecureJSON(http.StatusOK, Response{
		Data: hexString,
	})

}

// GetChainInfos 查询归档中心所有的链信息
func (srv *HttpSrv) GetChainInfos(ginContext *gin.Context) {
	chainStatuses := srv.ProxyProcessor.GetAllChainInfo()
	ginContext.JSONP(http.StatusOK, Response{
		Data: chainStatuses,
	})
}

// BlockExists 根据区块hash查询区块是否存在
func (srv *HttpSrv) BlockExists(ginContext *gin.Context) {
	var inParameter QueryParameter
	err := ginContext.BindJSON(&inParameter)
	if err != nil || len(inParameter.ChainGenesisHash) == 0 ||
		len(inParameter.BlockHash) == 0 {
		ginContext.SecureJSON(http.StatusOK, Response{
			Code:     archive_utils.CodeHttpInvalidParameter,
			ErrorMsg: archive_utils.MsgHttpInvalidParameter,
		})
		return
	}
	inParameter.ByteBlockHash, err = srv.transformHashToByte(inParameter.BlockHash)
	if err != nil {
		srv.logger.Errorf("BlockExists hash %s transform error %s",
			inParameter.BlockHash, err.Error())
		ginContext.SecureJSON(http.StatusOK, Response{
			Code:     archive_utils.CodeHttpInvalidParameter,
			ErrorMsg: archive_utils.MsgHttpInvalidParameter,
		})
		return
	}
	exist, existErr := srv.ProxyProcessor.BlockExists(inParameter.ChainGenesisHash,
		inParameter.ByteBlockHash, srv.ProxyProcessor.GetChainProcessorById)
	if existErr != nil {
		srv.logger.Errorf("BlockExists blockHash %s got error %s",
			inParameter.BlockHash, existErr.Error())
		ginContext.SecureJSON(http.StatusOK, Response{
			Code:     archive_utils.CodeBlockExistsFailed,
			ErrorMsg: existErr.Error(),
		})
		return
	}
	ginContext.SecureJSON(http.StatusOK, Response{
		Data: exist,
	})
}

// GetBlockByHash 根据区块hash查询区块
func (srv *HttpSrv) GetBlockByHash(ginContext *gin.Context) {
	var inParameter QueryParameter
	err := ginContext.BindJSON(&inParameter)
	if err != nil || len(inParameter.ChainGenesisHash) == 0 ||
		len(inParameter.BlockHash) == 0 {
		ginContext.SecureJSON(http.StatusOK, Response{
			Code:     archive_utils.CodeHttpInvalidParameter,
			ErrorMsg: archive_utils.MsgHttpInvalidParameter,
		})
		return
	}
	inParameter.ByteBlockHash, err = srv.transformHashToByte(inParameter.BlockHash)
	if err != nil {
		srv.logger.Errorf("GetBlockByHash hash %s transform error %s ",
			inParameter.BlockHash, err.Error())
		ginContext.SecureJSON(http.StatusOK, Response{
			ErrorMsg: archive_utils.MsgHttpInvalidParameter,
			Code:     archive_utils.CodeHttpInvalidParameter,
		})
		return
	}
	block, blockError := srv.ProxyProcessor.GetBlockByHash(inParameter.ChainGenesisHash,
		inParameter.ByteBlockHash, srv.ProxyProcessor.GetChainProcessorById)
	if blockError != nil {
		srv.logger.Errorf("GetBlockByHash hash %s got error %s",
			inParameter.BlockHash, blockError.Error())
		ginContext.SecureJSON(http.StatusOK, Response{
			ErrorMsg: blockError.Error(),
			Code:     archive_utils.CodeBlockByHashFailed,
		})
		return
	}
	ginContext.SecureJSON(http.StatusOK, Response{
		Data: block,
	})
}

// GetHeightByHash 根据hash查询块高度
func (srv *HttpSrv) GetHeightByHash(ginContext *gin.Context) {
	var inParameter QueryParameter
	err := ginContext.BindJSON(&inParameter)
	if err != nil || len(inParameter.ChainGenesisHash) == 0 ||
		len(inParameter.BlockHash) == 0 {
		ginContext.SecureJSON(http.StatusOK, Response{
			ErrorMsg: archive_utils.MsgHttpInvalidParameter,
			Code:     archive_utils.CodeHttpInvalidParameter,
		})
		return
	}
	inParameter.ByteBlockHash, err = srv.transformHashToByte(inParameter.BlockHash)
	if err != nil {
		srv.logger.Errorf("GetHeightByHash hash %s transform error %s",
			inParameter.BlockHash, err.Error())
		ginContext.SecureJSON(http.StatusOK, Response{
			ErrorMsg: archive_utils.MsgHttpInvalidParameter,
			Code:     archive_utils.CodeHttpInvalidParameter,
		})
		return
	}
	height, heightError := srv.ProxyProcessor.GetHeightByHash(inParameter.ChainGenesisHash,
		inParameter.ByteBlockHash, srv.ProxyProcessor.GetChainProcessorById)
	if heightError != nil {
		srv.logger.Errorf("GetHeightByHash hash %s got error %s ",
			inParameter.BlockHash, heightError.Error())
		ginContext.SecureJSON(http.StatusOK, Response{
			ErrorMsg: heightError.Error(),
			Code:     archive_utils.CodeHeightByHashFailed,
		})
		return
	}
	ginContext.SecureJSON(http.StatusOK, Response{
		Data: height,
	})
}

// GetBlockHeaderByHeight 根据块高查询区块头
func (srv *HttpSrv) GetBlockHeaderByHeight(ginContext *gin.Context) {
	var inParameter QueryParameter
	err := ginContext.BindJSON(&inParameter)
	if err != nil || len(inParameter.ChainGenesisHash) == 0 {
		ginContext.SecureJSON(http.StatusOK, Response{
			ErrorMsg: archive_utils.MsgHttpInvalidParameter,
			Code:     archive_utils.CodeHttpInvalidParameter,
		})
		return
	}

	header, headerError := srv.ProxyProcessor.GetBlockHeaderByHeight(inParameter.ChainGenesisHash,
		inParameter.Height, srv.ProxyProcessor.GetChainProcessorById)
	if headerError != nil {
		srv.logger.Errorf("GetBlockHeaderByHeight height %d got error %s",
			inParameter.Height, headerError.Error())
		ginContext.SecureJSON(http.StatusOK, Response{
			ErrorMsg: headerError.Error(),
			Code:     archive_utils.CodeHeaderByHeightFailed,
		})
		return
	}
	ginContext.SecureJSON(http.StatusOK, Response{
		Data: header,
	})
}

// GetBlock 根据块高查询区块
func (srv *HttpSrv) GetBlock(ginContext *gin.Context) {
	var inParameter QueryParameter
	err := ginContext.BindJSON(&inParameter)
	if err != nil || len(inParameter.ChainGenesisHash) == 0 {
		ginContext.SecureJSON(http.StatusOK, Response{
			ErrorMsg: archive_utils.MsgHttpInvalidParameter,
			Code:     archive_utils.CodeHttpInvalidParameter,
		})
		return
	}
	block, blockError := srv.ProxyProcessor.GetBlock(inParameter.ChainGenesisHash,
		inParameter.Height, srv.ProxyProcessor.GetChainProcessorById)
	if blockError != nil {
		srv.logger.Errorf("GetBlock height %d got error %s",
			inParameter.Height, blockError.Error())
		ginContext.SecureJSON(http.StatusOK, Response{
			ErrorMsg: blockError.Error(),
			Code:     archive_utils.CodeBlockByHeightFailed,
		})
		return
	}
	ginContext.SecureJSON(http.StatusOK, Response{
		Data: block,
	})
}

// GetFullBlockWithRWSetByHeight 根据块高查询全块信息
func (srv *HttpSrv) GetFullBlockWithRWSetByHeight(ginContext *gin.Context) {
	var inParameter QueryParameter
	err := ginContext.BindJSON(&inParameter)
	if err != nil || len(inParameter.ChainGenesisHash) == 0 {
		ginContext.SecureJSON(http.StatusOK, Response{
			ErrorMsg: archive_utils.MsgHttpInvalidParameter,
			Code:     archive_utils.CodeHttpInvalidParameter,
		})
		return
	}
	block, blockError := srv.ProxyProcessor.GetBlockWithRWSetByHeight(inParameter.ChainGenesisHash,
		inParameter.Height, srv.ProxyProcessor.GetChainProcessorById)
	if blockError != nil {
		srv.logger.Errorf("GetFullBlockWithRWSetByHeight height %d got error %s",
			inParameter.Height, blockError.Error())
		ginContext.SecureJSON(http.StatusOK, Response{
			ErrorMsg: blockError.Error(),
			Code:     archive_utils.CodeBlockByHeightFailed,
		})
		return
	}
	ginContext.SecureJSON(http.StatusOK, Response{
		Data: block,
	})
}

// GetChainConfigByHeight 根据高度查询该高度前最近的配置区块信息
func (srv *HttpSrv) GetChainConfigByHeight(ginContext *gin.Context) {
	var inParameter QueryParameter
	err := ginContext.BindJSON(&inParameter)
	if err != nil || len(inParameter.ChainGenesisHash) == 0 {
		ginContext.SecureJSON(http.StatusOK, Response{
			ErrorMsg: archive_utils.MsgHttpInvalidParameter,
			Code:     archive_utils.CodeHttpInvalidParameter,
		})
		return
	}
	config, configError := srv.ProxyProcessor.GetChainConfigByHeight(inParameter.ChainGenesisHash,
		inParameter.Height, srv.ProxyProcessor.GetChainProcessorById)
	if configError != nil {
		srv.logger.Errorf("GetChainConfigByHeight height %d got error %s ",
			inParameter.Height, configError.Error())
		ginContext.SecureJSON(http.StatusOK, Response{
			ErrorMsg: configError.Error(),
			Code:     archive_utils.CodeGetConfigByHeightFailed,
		})
		return
	}
	ginContext.SecureJSON(http.StatusOK, Response{
		Data: config,
	})
}

// GetBlockInfoByHeight 根据高度查询block信息
func (srv *HttpSrv) GetBlockInfoByHeight(ginContext *gin.Context) {
	var inParameter QueryParameter
	err := ginContext.BindJSON(&inParameter)
	if err != nil || len(inParameter.ChainGenesisHash) == 0 {
		ginContext.SecureJSON(http.StatusOK, Response{
			ErrorMsg: archive_utils.MsgHttpInvalidParameter,
			Code:     archive_utils.CodeHttpInvalidParameter,
		})
		return
	}
	block, blockError := srv.ProxyProcessor.GetCommonBlockInfoByHeight(inParameter.ChainGenesisHash,
		inParameter.Height, srv.ProxyProcessor.GetChainProcessorById)
	if blockError != nil {
		srv.logger.Errorf("GetBlockInfoByHeight height %d got error %s",
			inParameter.Height, blockError.Error())
		ginContext.SecureJSON(http.StatusOK, Response{
			ErrorMsg: blockError.Error(),
			Code:     archive_utils.CodeBlockByHeightFailed,
		})
		return
	}
	ginContext.SecureJSON(http.StatusOK, Response{
		Data: block,
	})
}

// GetBlockInfoByHash 根据hash查询block信息
func (srv *HttpSrv) GetBlockInfoByHash(ginContext *gin.Context) {
	var inParameter QueryParameter
	err := ginContext.BindJSON(&inParameter)
	if err != nil || len(inParameter.ChainGenesisHash) == 0 ||
		len(inParameter.BlockHash) == 0 {
		ginContext.SecureJSON(http.StatusOK, Response{
			Code:     archive_utils.CodeHttpInvalidParameter,
			ErrorMsg: archive_utils.MsgHttpInvalidParameter,
		})
		return
	}
	inParameter.ByteBlockHash, err = srv.transformHashToByte(inParameter.BlockHash)
	if err != nil {
		srv.logger.Errorf("GetBlockInfoByHash hash %s transform error %s ",
			inParameter.BlockHash, err.Error())
		ginContext.SecureJSON(http.StatusOK, Response{
			ErrorMsg: archive_utils.MsgHttpInvalidParameter,
			Code:     archive_utils.CodeHttpInvalidParameter,
		})
		return
	}
	block, blockError := srv.ProxyProcessor.GetCommonBlockInfoByHash(inParameter.ChainGenesisHash,
		inParameter.ByteBlockHash, srv.ProxyProcessor.GetChainProcessorById)
	if blockError != nil {
		srv.logger.Errorf("GetBlockInfoByHash hash %s got error %s",
			inParameter.BlockHash, blockError.Error())
		ginContext.SecureJSON(http.StatusOK, Response{
			ErrorMsg: blockError.Error(),
			Code:     archive_utils.CodeBlockByHashFailed,
		})
		return
	}
	ginContext.SecureJSON(http.StatusOK, Response{
		Data: block,
	})
}

// GetBlockInfoByTxId 根据txid查询block信息
func (srv *HttpSrv) GetBlockInfoByTxId(ginContext *gin.Context) {
	var inParameter QueryParameter
	err := ginContext.BindJSON(&inParameter)
	if err != nil || len(inParameter.ChainGenesisHash) == 0 ||
		len(inParameter.TxId) == 0 {
		ginContext.SecureJSON(http.StatusOK, Response{
			ErrorMsg: archive_utils.MsgHttpInvalidParameter,
			Code:     archive_utils.CodeHttpInvalidParameter,
		})
		return
	}
	block, blockError := srv.ProxyProcessor.GetCommonBlockInfoByTxId(inParameter.ChainGenesisHash,
		inParameter.TxId, srv.ProxyProcessor.GetChainProcessorById)
	if blockError != nil {
		srv.logger.Errorf("GetBlockInfoByTxId hash %s got error %s",
			inParameter.BlockHash, blockError.Error())
		ginContext.SecureJSON(http.StatusOK, Response{
			ErrorMsg: blockError.Error(),
			Code:     archive_utils.CodeBlockByHashFailed,
		})
		return
	}
	ginContext.SecureJSON(http.StatusOK, Response{
		Data: block,
	})
}

// GetMerklePathByTxId 根据txid 计算merklepath
func (srv *HttpSrv) GetMerklePathByTxId(ginContext *gin.Context) {
	var inParameter QueryParameter
	err := ginContext.BindJSON(&inParameter)
	if err != nil || len(inParameter.ChainGenesisHash) == 0 ||
		len(inParameter.TxId) == 0 {
		ginContext.SecureJSON(http.StatusOK, Response{
			ErrorMsg: archive_utils.MsgHttpInvalidParameter,
			Code:     archive_utils.CodeHttpInvalidParameter,
		})
		return
	}
	path, pathErr := srv.ProxyProcessor.GetMerklePathByTxId(inParameter.ChainGenesisHash,
		inParameter.TxId, srv.ProxyProcessor.GetChainProcessorById)
	if pathErr != nil {
		ginContext.SecureJSON(http.StatusOK,
			Response{
				ErrorMsg: pathErr.Error(),
				Code:     archive_utils.ErrorMerklePath,
			})
		return
	}
	ginContext.SecureJSON(http.StatusOK, Response{
		Data: path,
	})
}

// GetTruncateBlockByHeight 根据高度查询区块
func (srv *HttpSrv) GetTruncateBlockByHeight(ginContext *gin.Context) {
	var inParameter QueryParameter
	err := ginContext.BindJSON(&inParameter)
	if err != nil || len(inParameter.ChainGenesisHash) == 0 {
		ginContext.SecureJSON(http.StatusOK, Response{
			ErrorMsg: archive_utils.MsgHttpInvalidParameter,
			Code:     archive_utils.CodeHttpInvalidParameter,
		})
		return
	}
	block, blockErr :=
		srv.ProxyProcessor.GetBlockByHeightTruncate(inParameter.ChainGenesisHash,
			srv.ProxyProcessor.GetChainProcessorById, inParameter.Height,
			inParameter.WithRwSet, inParameter.TruncateLength, inParameter.TruncateModel)
	if blockErr != nil {
		ginContext.SecureJSON(http.StatusOK, Response{
			ErrorMsg: blockErr.Error(),
			Code:     archive_utils.CodeBlockByHeightFailed,
		})
		return
	}
	ginContext.SecureJSON(http.StatusOK, Response{
		Data: block,
	})
}

// GetTruncateTxByTxId 根据txid查询交易
func (srv *HttpSrv) GetTruncateTxByTxId(ginContext *gin.Context) {
	var inParameter QueryParameter
	err := ginContext.BindJSON(&inParameter)
	if err != nil || len(inParameter.ChainGenesisHash) == 0 ||
		len(inParameter.TxId) == 0 {
		ginContext.SecureJSON(http.StatusOK, Response{
			ErrorMsg: archive_utils.MsgHttpInvalidParameter,
			Code:     archive_utils.CodeHttpInvalidParameter,
		})
		return
	}
	tx, txErr :=
		srv.ProxyProcessor.GetTxByTxIdTruncate(inParameter.ChainGenesisHash, inParameter.TxId,
			srv.ProxyProcessor.GetChainProcessorById, inParameter.WithRwSet,
			inParameter.TruncateLength, inParameter.TruncateModel)
	if txErr != nil {
		ginContext.SecureJSON(http.StatusOK, Response{
			ErrorMsg: txErr.Error(),
			Code:     archive_utils.CodeTxByTxIdFailed,
		})
		return
	}
	ginContext.SecureJSON(http.StatusOK, Response{
		Data: tx,
	})
}

// GetTx 根据事务id查询事务
func (srv *HttpSrv) GetTx(ginContext *gin.Context) {
	var inParameter QueryParameter
	err := ginContext.BindJSON(&inParameter)
	if err != nil || len(inParameter.ChainGenesisHash) == 0 ||
		len(inParameter.TxId) == 0 {
		ginContext.SecureJSON(http.StatusOK, Response{
			ErrorMsg: archive_utils.MsgHttpInvalidParameter,
			Code:     archive_utils.CodeHttpInvalidParameter,
		})
		return
	}
	tx, txError := srv.ProxyProcessor.GetTx(inParameter.ChainGenesisHash,
		inParameter.TxId, srv.ProxyProcessor.GetChainProcessorById)
	if txError != nil {
		srv.logger.Errorf("GetTx by txid %s got error %s ",
			inParameter.TxId, txError.Error())
		ginContext.SecureJSON(http.StatusOK, Response{
			ErrorMsg: txError.Error(),
			Code:     archive_utils.CodeTxByTxIdFailed,
		})
		return
	}
	ginContext.SecureJSON(http.StatusOK, Response{
		Data: tx,
	})
}

// GetTxWithBlockInfo 根据事务id查询事务信息，TransactionStoreInfo为空（文件索引信息无用）
func (srv *HttpSrv) GetTxWithBlockInfo(ginContext *gin.Context) {
	var inParameter QueryParameter
	err := ginContext.BindJSON(&inParameter)
	if err != nil || len(inParameter.ChainGenesisHash) == 0 ||
		len(inParameter.TxId) == 0 {
		ginContext.SecureJSON(http.StatusOK, Response{
			ErrorMsg: archive_utils.MsgHttpInvalidParameter,
			Code:     archive_utils.CodeHttpInvalidParameter,
		})
		return
	}

	tx, txError := srv.ProxyProcessor.GetTxWithBlockInfo(inParameter.ChainGenesisHash,
		inParameter.TxId, srv.ProxyProcessor.GetChainProcessorById)
	if txError != nil {
		srv.logger.Errorf("GetTxWithBlockInfo by txid %s got error %s ",
			inParameter.TxId, txError.Error())
		ginContext.SecureJSON(http.StatusOK, Response{
			ErrorMsg: txError.Error(),
			Code:     archive_utils.CodeBlockByTxIdFailed,
		})
		return
	}
	ginContext.SecureJSON(http.StatusOK, Response{
		Data: tx,
	})
}

// GetCommonTxInfo 获取交易信息
func (srv *HttpSrv) GetCommonTxInfo(ginContext *gin.Context) {
	var inParameter QueryParameter
	err := ginContext.BindJSON(&inParameter)
	if err != nil || len(inParameter.ChainGenesisHash) == 0 ||
		len(inParameter.TxId) == 0 {
		ginContext.SecureJSON(http.StatusOK, Response{
			ErrorMsg: archive_utils.MsgHttpInvalidParameter,
			Code:     archive_utils.CodeHttpInvalidParameter,
		})
		return
	}

	tx, txError := srv.ProxyProcessor.GetCommonTransactionInfo(inParameter.ChainGenesisHash,
		inParameter.TxId, srv.ProxyProcessor.GetChainProcessorById)
	if txError != nil {
		srv.logger.Errorf("GetCommonTxInfo by txid %s got error %s ",
			inParameter.TxId, txError.Error())
		ginContext.SecureJSON(http.StatusOK, Response{
			ErrorMsg: txError.Error(),
			Code:     archive_utils.CodeBlockByTxIdFailed,
		})
		return
	}
	ginContext.SecureJSON(http.StatusOK, Response{
		Data: tx,
	})
}

// GetCommonTxInfoWithRWSet 根据txid查询区块信息
func (srv *HttpSrv) GetCommonTxInfoWithRWSet(ginContext *gin.Context) {
	var inParameter QueryParameter
	err := ginContext.BindJSON(&inParameter)
	if err != nil || len(inParameter.ChainGenesisHash) == 0 ||
		len(inParameter.TxId) == 0 {
		ginContext.SecureJSON(http.StatusOK, Response{
			ErrorMsg: archive_utils.MsgHttpInvalidParameter,
			Code:     archive_utils.CodeHttpInvalidParameter,
		})
		return
	}

	tx, txError := srv.ProxyProcessor.GetCommonTransactionWithRWSet(inParameter.ChainGenesisHash,
		inParameter.TxId, srv.ProxyProcessor.GetChainProcessorById)
	if txError != nil {
		srv.logger.Errorf("GetCommonTxInfoWithRWSet by txid %s got error %s ",
			inParameter.TxId, txError.Error())
		ginContext.SecureJSON(http.StatusOK, Response{
			ErrorMsg: txError.Error(),
			Code:     archive_utils.CodeBlockByTxIdFailed,
		})
		return
	}
	ginContext.SecureJSON(http.StatusOK, Response{
		Data: tx,
	})
}

// GetTxInfoOnly 获得除Tx之外的其他TxInfo信息
func (srv *HttpSrv) GetTxInfoOnly(ginContext *gin.Context) {
	var inParameter QueryParameter
	err := ginContext.BindJSON(&inParameter)
	if err != nil || len(inParameter.ChainGenesisHash) == 0 ||
		len(inParameter.TxId) == 0 {
		ginContext.SecureJSON(http.StatusOK, Response{
			ErrorMsg: archive_utils.MsgHttpInvalidParameter,
			Code:     archive_utils.CodeHttpInvalidParameter,
		})
		return
	}
	tx, txError := srv.ProxyProcessor.GetTxInfoOnly(inParameter.ChainGenesisHash,
		inParameter.TxId, srv.ProxyProcessor.GetChainProcessorById)
	if txError != nil {
		srv.logger.Errorf("GetTxInfoOnly txid %s got error %s",
			inParameter.TxId, txError.Error())
		ginContext.SecureJSON(http.StatusOK, Response{
			ErrorMsg: txError.Error(),
			Code:     archive_utils.CodeTxByTxIdFailed,
		})
		return
	}
	ginContext.SecureJSON(http.StatusOK, Response{
		Data: tx,
	})
}

// GetTxHeight 根据事务id查询块高
func (srv *HttpSrv) GetTxHeight(ginContext *gin.Context) {
	var inParameter QueryParameter
	err := ginContext.BindJSON(&inParameter)
	if err != nil || len(inParameter.ChainGenesisHash) == 0 ||
		len(inParameter.TxId) == 0 {
		ginContext.SecureJSON(http.StatusOK, Response{
			ErrorMsg: archive_utils.MsgHttpInvalidParameter,
			Code:     archive_utils.CodeHttpInvalidParameter,
		})
		return
	}

	height, heightError := srv.ProxyProcessor.GetTxHeight(inParameter.ChainGenesisHash,
		inParameter.TxId, srv.ProxyProcessor.GetChainProcessorById)
	if heightError != nil {
		srv.logger.Errorf("GetTxHeight by txid %s got error %s",
			inParameter.TxId, heightError.Error())
		ginContext.SecureJSON(http.StatusOK, Response{
			ErrorMsg: heightError.Error(),
			Code:     archive_utils.CodeTxHeightByTxIdFailed,
		})
		return
	}
	ginContext.SecureJSON(http.StatusOK, Response{
		Data: height,
	})
}

// TxExists 查询事务是否存在
func (srv *HttpSrv) TxExists(ginContext *gin.Context) {
	var inParameter QueryParameter
	err := ginContext.BindJSON(&inParameter)
	if err != nil || len(inParameter.ChainGenesisHash) == 0 ||
		len(inParameter.TxId) == 0 {
		ginContext.SecureJSON(http.StatusOK, Response{
			ErrorMsg: archive_utils.MsgHttpInvalidParameter,
			Code:     archive_utils.CodeHttpInvalidParameter,
		})
		return
	}
	exist, existError := srv.ProxyProcessor.TxExists(inParameter.ChainGenesisHash,
		inParameter.TxId, srv.ProxyProcessor.GetChainProcessorById)
	if existError != nil {
		srv.logger.Errorf("TxExists  by txid %s got error %s ",
			inParameter.TxId, existError.Error())
		ginContext.SecureJSON(http.StatusOK, Response{
			ErrorMsg: existError.Error(),
			Code:     archive_utils.CodeTxExistsByTxIdFailed,
		})
		return
	}
	ginContext.SecureJSON(http.StatusOK, Response{
		Data: exist,
	})
}

// GetTxConfirmedTime 查询区块确认时间
func (srv *HttpSrv) GetTxConfirmedTime(ginContext *gin.Context) {
	var inParameter QueryParameter
	err := ginContext.BindJSON(&inParameter)
	if err != nil || len(inParameter.ChainGenesisHash) == 0 ||
		len(inParameter.TxId) == 0 {
		ginContext.SecureJSON(http.StatusOK, Response{
			ErrorMsg: archive_utils.MsgHttpInvalidParameter,
			Code:     archive_utils.CodeHttpInvalidParameter,
		})
		return
	}

	confirmedTime, confirmedError := srv.ProxyProcessor.GetTxConfirmedTime(inParameter.ChainGenesisHash,
		inParameter.TxId, srv.ProxyProcessor.GetChainProcessorById)
	if confirmedError != nil {
		srv.logger.Errorf("GetTxConfirmedTime by txid %s got error %s",
			inParameter.TxId, confirmedError.Error())
		ginContext.SecureJSON(http.StatusOK, Response{
			ErrorMsg: confirmedError.Error(),
			Code:     archive_utils.CodeTxConfirmedByTxIdFailed,
		})
		return
	}
	ginContext.SecureJSON(http.StatusOK, Response{
		Data: confirmedTime,
	})
}

// GetLastBlock 查询最后一个区块
func (srv *HttpSrv) GetLastBlock(ginContext *gin.Context) {
	var inParameter QueryParameter
	err := ginContext.BindJSON(&inParameter)
	if err != nil || len(inParameter.ChainGenesisHash) == 0 {
		ginContext.SecureJSON(http.StatusOK, Response{
			ErrorMsg: archive_utils.MsgHttpInvalidParameter,
			Code:     archive_utils.CodeHttpInvalidParameter,
		})
		return
	}
	block, blockError := srv.ProxyProcessor.GetLastBlock(inParameter.ChainGenesisHash,
		srv.ProxyProcessor.GetChainProcessorById)
	if blockError != nil {
		srv.logger.Errorf("GetLastBlock got error %s", blockError.Error())
		ginContext.SecureJSON(http.StatusOK, Response{
			ErrorMsg: blockError.Error(),
			Code:     archive_utils.CodeGetLastBlockFailed,
		})
		return
	}
	ginContext.SecureJSON(http.StatusOK, Response{
		Data: block,
	})
}

// GetFilteredBlock 根据高度查询序列化后的区块信息
func (srv *HttpSrv) GetFilteredBlock(ginContext *gin.Context) {
	var inParameter QueryParameter
	err := ginContext.BindJSON(&inParameter)
	if err != nil || len(inParameter.ChainGenesisHash) == 0 {
		ginContext.SecureJSON(http.StatusOK, Response{
			ErrorMsg: archive_utils.MsgHttpInvalidParameter,
			Code:     archive_utils.CodeHttpInvalidParameter,
		})
		return
	}

	filteredBlock, filteredError := srv.ProxyProcessor.GetFilteredBlock(inParameter.ChainGenesisHash,
		inParameter.Height, srv.ProxyProcessor.GetChainProcessorById)
	if filteredError != nil {
		srv.logger.Errorf("GetFilteredBlock height %d got error %s",
			inParameter.Height, filteredError.Error())
		ginContext.SecureJSON(http.StatusOK, Response{
			ErrorMsg: filteredError.Error(),
			Code:     archive_utils.CodeBlockByHeightFailed,
		})
		return
	}
	ginContext.SecureJSON(http.StatusOK, Response{
		Data: filteredBlock,
	})
}

// GetLastSavepoint 返回当前最后一个归档高度
func (srv *HttpSrv) GetLastSavepoint(ginContext *gin.Context) {
	var inParameter QueryParameter
	err := ginContext.BindJSON(&inParameter)
	if err != nil || len(inParameter.ChainGenesisHash) == 0 {
		ginContext.SecureJSON(http.StatusOK, Response{
			ErrorMsg: archive_utils.MsgHttpInvalidParameter,
			Code:     archive_utils.CodeHttpInvalidParameter,
		})
		return
	}
	height, heightError := srv.ProxyProcessor.GetLastSavepoint(inParameter.ChainGenesisHash,
		srv.ProxyProcessor.GetChainProcessorById)
	if heightError != nil {
		srv.logger.Errorf("GetLastSavepoint got error %s",
			heightError.Error())
		ginContext.SecureJSON(http.StatusOK, Response{
			ErrorMsg: heightError.Error(),
			Code:     archive_utils.CodeGetLastSavePointFailed,
		})
		return
	}
	ginContext.SecureJSON(http.StatusOK, Response{
		Data: height,
	})
}

// GetLastConfigBlock 查询最后一个归档区块
func (srv *HttpSrv) GetLastConfigBlock(ginContext *gin.Context) {
	var inParameter QueryParameter
	err := ginContext.BindJSON(&inParameter)
	if err != nil || len(inParameter.ChainGenesisHash) == 0 {
		ginContext.SecureJSON(http.StatusOK, Response{
			ErrorMsg: archive_utils.MsgHttpInvalidParameter,
			Code:     archive_utils.CodeHttpInvalidParameter,
		})
		return
	}

	block, blockError := srv.ProxyProcessor.GetLastConfigBlock(inParameter.ChainGenesisHash,
		srv.ProxyProcessor.GetChainProcessorById)
	if blockError != nil {
		srv.logger.Errorf("GetLastConfigBlock got error %s", blockError.Error())
		ginContext.SecureJSON(http.StatusOK, Response{
			ErrorMsg: blockError.Error(),
			Code:     archive_utils.CodeLastConfigFailed,
		})
		return
	}
	ginContext.SecureJSON(http.StatusOK, Response{
		Data: block,
	})
}

// GetLastConfigBlockHeight 查询最后一个归档区块的高度
func (srv *HttpSrv) GetLastConfigBlockHeight(ginContext *gin.Context) {
	var inParameter QueryParameter
	err := ginContext.BindJSON(&inParameter)
	if err != nil || len(inParameter.ChainGenesisHash) == 0 {
		ginContext.SecureJSON(http.StatusOK, Response{
			ErrorMsg: archive_utils.MsgHttpInvalidParameter,
			Code:     archive_utils.CodeHttpInvalidParameter,
		})
		return
	}

	height, heightError := srv.ProxyProcessor.GetLastConfigBlockHeight(inParameter.ChainGenesisHash,
		srv.ProxyProcessor.GetChainProcessorById)
	if heightError != nil {
		srv.logger.Errorf("GetLastConfigBlockHeight got error %s", heightError.Error())
		ginContext.SecureJSON(http.StatusOK, Response{
			ErrorMsg: heightError.Error(),
			Code:     archive_utils.CodeLastConfigFailed,
		})
		return
	}
	ginContext.SecureJSON(http.StatusOK, Response{
		Data: height,
	})
}

// GetBlockByTx 根据事务id查询区块信息
func (srv *HttpSrv) GetBlockByTx(ginContext *gin.Context) {
	var inParameter QueryParameter
	err := ginContext.BindJSON(&inParameter)
	if err != nil || len(inParameter.ChainGenesisHash) == 0 ||
		len(inParameter.TxId) == 0 {
		ginContext.SecureJSON(http.StatusOK, Response{
			ErrorMsg: archive_utils.MsgHttpInvalidParameter,
			Code:     archive_utils.CodeHttpInvalidParameter,
		})
		return
	}
	block, blockError := srv.ProxyProcessor.GetBlockByTx(inParameter.ChainGenesisHash,
		inParameter.TxId, srv.ProxyProcessor.GetChainProcessorById)
	if blockError != nil {
		srv.logger.Errorf("GetBlockByTx by txid %s got error %s",
			inParameter.TxId, blockError.Error())
		ginContext.SecureJSON(http.StatusOK, Response{
			ErrorMsg: blockError.Error(),
			Code:     archive_utils.CodeBlockByTxIdFailed,
		})
		return
	}
	ginContext.SecureJSON(http.StatusOK, Response{
		Data: block,
	})
}

// GetTxRWSet 根据事务id查询事务读写集
func (srv *HttpSrv) GetTxRWSet(ginContext *gin.Context) {
	var inParameter QueryParameter
	err := ginContext.BindJSON(&inParameter)
	if err != nil || len(inParameter.ChainGenesisHash) == 0 ||
		len(inParameter.TxId) == 0 {
		ginContext.SecureJSON(http.StatusOK, Response{
			ErrorMsg: archive_utils.MsgHttpInvalidParameter,
			Code:     archive_utils.CodeHttpInvalidParameter,
		})
		return
	}
	txRwSet, txRwSetError := srv.ProxyProcessor.GetTxRWSet(inParameter.ChainGenesisHash,
		inParameter.TxId, srv.ProxyProcessor.GetChainProcessorById)
	if txRwSetError != nil {
		srv.logger.Errorf("GetTxRWSet by txid %s got error %s",
			inParameter.TxId, txRwSetError.Error())
		ginContext.SecureJSON(http.StatusOK, Response{
			ErrorMsg: txRwSetError.Error(),
			Code:     archive_utils.CodeTxRWSetByTxIdFailed,
		})
		return
	}
	ginContext.SecureJSON(http.StatusOK, Response{
		Data: txRwSet,
	})
}

// CompressUnderHeight 压缩指定高度下的所有区块
func (srv *HttpSrv) CompressUnderHeight(ginContext *gin.Context) {
	var inParameter QueryParameter
	err := ginContext.BindJSON(&inParameter)
	if err != nil || len(inParameter.ChainGenesisHash) == 0 {
		ginContext.SecureJSON(http.StatusOK, Response{
			ErrorMsg: archive_utils.MsgHttpInvalidParameter,
			Code:     archive_utils.CodeHttpInvalidParameter,
		})
		return
	}
	//  耗时太久，这个要不要go出去一个协程？
	start, end, compressError := srv.ProxyProcessor.CompressUnderHeight(inParameter.ChainGenesisHash,
		inParameter.Height, srv.ProxyProcessor.GetChainProcessorById)
	if compressError != nil {
		srv.logger.Errorf("CompressUnderHeight height %d got error %s",
			inParameter.Height, compressError.Error())
		ginContext.SecureJSON(http.StatusOK, Response{
			ErrorMsg: compressError.Error(),
			Code:     archive_utils.CodeCompressUnderHeightFailed,
		})
		return
	}
	ginContext.SecureJSON(http.StatusOK, Response{
		Data: map[string]interface{}{
			"StartCompressHeight": start,
			"EndCompressHeight":   end,
		},
	})

}

// GetChainCompressStatus 获取指定链的压缩高度，是否正在压缩
func (srv *HttpSrv) GetChainCompressStatus(ginContext *gin.Context) {
	var inParameter QueryParameter
	err := ginContext.BindJSON(&inParameter)
	if err != nil || len(inParameter.ChainGenesisHash) == 0 {
		ginContext.SecureJSON(http.StatusOK, Response{
			ErrorMsg: archive_utils.MsgHttpInvalidParameter,
			Code:     archive_utils.CodeHttpInvalidParameter,
		})
		return
	}
	compressHeight, isCompressing, statusError := srv.ProxyProcessor.GetCompressStatus(inParameter.ChainGenesisHash,
		srv.ProxyProcessor.GetChainProcessorById)
	if statusError != nil {
		srv.logger.Errorf("GetChainCompressStatus got error %s",
			statusError.Error())
		ginContext.SecureJSON(http.StatusOK, Response{
			ErrorMsg: statusError.Error(),
			Code:     archive_utils.CodeGetCompressStatusFailed,
		})
		return
	}
	ginContext.SecureJSON(http.StatusOK, Response{
		Data: map[string]interface{}{
			"CompressedHegiht": compressHeight,
			"IsCompressing":    isCompressing,
		},
	})
}

// AddCA 增加CA接口，直接将CA信息保存到kv数据库中即可
func (srv *HttpSrv) AddCA(ginContext *gin.Context) {
	// 1. 从请求中读取CA文件
	caHeader, caHeaderError := ginContext.FormFile(caFileName)
	if caHeaderError != nil {
		srv.logger.Errorf("AddCA formFile error %s", caHeaderError.Error())
		ginContext.SecureJSON(http.StatusOK, Response{
			Code:     archive_utils.CodeCAFileFailed,
			ErrorMsg: fmt.Sprintf("illegal ca file , error(%s)", caHeaderError.Error()),
		})
		return
	}
	caFile, caFileError := caHeader.Open()
	if caFileError != nil {
		srv.logger.Errorf("AddCA Open error %s", caFileError.Error())
		ginContext.SecureJSON(http.StatusOK, Response{
			Code:     archive_utils.CodeCAFileFailed,
			ErrorMsg: fmt.Sprintf("server open ca error (%s)", caFileError.Error()),
		})
		return
	}
	defer caFile.Close()
	caBytes, caBytesError := ioutil.ReadAll(caFile)
	if caBytesError != nil {
		srv.logger.Errorf("AddCA ReadAll error %s", caBytesError.Error())
		ginContext.SecureJSON(http.StatusOK, Response{
			Code:     archive_utils.CodeCAFileFailed,
			ErrorMsg: fmt.Sprintf("server read ca error (%s)", caBytesError.Error()),
		})
		return
	}
	saveError := srv.ProxyProcessor.SaveCAInKV(caBytes)
	if saveError != nil {
		srv.logger.Errorf("AddCA save ca error %s", saveError.Error())
		ginContext.SecureJSON(http.StatusOK, Response{
			Code:     archive_utils.CodeCAFileFailed,
			ErrorMsg: fmt.Sprintf("server save ca error (%s)", saveError.Error()),
		})
		return
	}
	srv.ProxyProcessor.SendUpdateCASignal()
	ginContext.SecureJSON(http.StatusOK, Response{
		Data: "server add ca successful",
	})
}

// HealthCheck 健康检查接口
func (srv *HttpSrv) HealthCheck(ginContext *gin.Context) {
	ginContext.SecureJSON(http.StatusOK, Response{
		Data: "ok",
	})
}
