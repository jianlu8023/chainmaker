/*
Copyright (C) BABEC. All rights reserved.


SPDX-License-Identifier: Apache-2.0
*/

// Package rpcserver define rpc server
package rpcserver

import (
	"context"
	"encoding/hex"
	"fmt"
	"io"

	"chainmaker.org/chainmaker-archive-service/src/archive_utils"
	"chainmaker.org/chainmaker-archive-service/src/process"
	"chainmaker.org/chainmaker-archive-service/src/serverconf"
	archivePb "chainmaker.org/chainmaker/pb-go/v2/archivecenter"
	commonPb "chainmaker.org/chainmaker/pb-go/v2/common"
	storePb "chainmaker.org/chainmaker/pb-go/v2/store"
)

// ApiService api服务结构
type ApiService struct {
	ProxyProcessor process.ProcessorMgr
}

// Register 节点注册
func (s *RpcServer) Register(ctx context.Context,
	req *archivePb.ArchiveBlockRequest) (*archivePb.RegisterResp, error) {
	var resp archivePb.RegisterResp
	archive_utils.GlobalServerLatch.Add(1)
	defer archive_utils.GlobalServerLatch.Done()
	if req == nil || req.Block == nil {
		resp.Message = archive_utils.MsgRegisterBlockNil
		resp.Code = uint32(archive_utils.CodeRegisterBlockNil)
		return &resp, nil
	}
	registerStatus, registerError := s.ProxyProcessor.RegisterChainByGenesisBlock(req.Block,
		&serverconf.GlobalServerCFG.LogCFG)
	if registerError != nil {
		resp.Code = registerStatus
		resp.Message = registerError.Error()
		return &resp, nil
	}
	resp.RegisterStatus = archivePb.RegisterStatus(registerStatus)
	return &resp, nil
}

func (s *RpcServer) streamContextError(ctx context.Context) bool {
	switch ctx.Err() {
	case context.Canceled:
		s.logger.Warnf("RpcServer got canceled signal")
		return true
	case context.DeadlineExceeded:
		s.logger.Warn("RpcServer got deadline signal")
		return true
	default:
		return false
	}
}

// SingleArchiveBlocks 客户端单向流传输归档数据
func (s *RpcServer) SingleArchiveBlocks(srv archivePb.ArchiveCenterServer_SingleArchiveBlocksServer) error {
	var archivedBeginHeight, archivedEndHeight uint64
	var resp archivePb.SingleArchiveBlockResp
	i := 0
	for {
		tempRequest, tempRequestError := srv.Recv()
		if tempRequestError != nil {
			if tempRequestError == io.EOF {
				// 说明客户端已经发送完了数据
				s.logger.Info("ArchiveBlocks got EOF")
				break

			}
			s.logger.Errorf("ArchiveBlocks recv got error %s ", tempRequestError.Error())
			resp.ArchivedBeginHeight = archivedBeginHeight
			resp.ArchivedEndHeight = archivedEndHeight
			resp.Code = uint32(archive_utils.CodeArchiveRecvError)
			resp.Message = tempRequestError.Error()
			srv.SendAndClose(&resp)
			return tempRequestError

		}
		if len(tempRequest.ChainUnique) == 0 || tempRequest == nil ||
			tempRequest.Block == nil ||
			tempRequest.Block.Block == nil ||
			tempRequest.Block.Block.Header == nil {
			resp.ArchivedBeginHeight = archivedBeginHeight
			resp.ArchivedEndHeight = archivedEndHeight
			resp.Code = uint32(archive_utils.CodeArchiveBlockNil)
			resp.Message = archive_utils.MsgArchiveBlockNil
			srv.SendAndClose(&resp)
			return nil
		}
		s.logger.Debugf("ArchiveBlocks.block %d", tempRequest.Block.Block.Header.BlockHeight)
		if i == 0 {
			s.ProxyProcessor.MarkInArchive(tempRequest.ChainUnique,
				s.ProxyProcessor.GetChainProcessorByHash)
			defer s.ProxyProcessor.MarkNotInArchive(tempRequest.ChainUnique,
				s.ProxyProcessor.GetChainProcessorByHash)
		}
		// 归档
		_, resultErr := s.ProxyProcessor.ArchiveBlock(tempRequest.ChainUnique,
			tempRequest.Block, s.ProxyProcessor.GetChainProcessorByHash)
		if resultErr != nil {
			resp.ArchivedBeginHeight = archivedBeginHeight
			resp.ArchivedEndHeight = archivedEndHeight
			resp.Code = uint32(archive_utils.CodeArchiveBlockError)
			resp.Message = fmt.Sprintf("ArchiveBlock height %d ,got error %s",
				tempRequest.Block.Block.Header.BlockHeight, resultErr.Error())
			s.logger.Errorf("ArchiveBlocks.err block %d , error %s",
				tempRequest.Block.Block.Header.BlockHeight, resultErr.Error())
			srv.SendAndClose(&resp)
			return nil
		}
		if i == 0 {
			archivedBeginHeight = tempRequest.Block.Block.Header.BlockHeight
		}
		archivedEndHeight = tempRequest.Block.Block.Header.BlockHeight
		i++
	}
	srv.SendAndClose(&archivePb.SingleArchiveBlockResp{
		ArchivedBeginHeight: archivedBeginHeight,
		ArchivedEndHeight:   archivedEndHeight,
	})
	return nil
}

// ArchiveBlocks 流式归档区块
func (s *RpcServer) ArchiveBlocks(srv archivePb.ArchiveCenterServer_ArchiveBlocksServer) error {
	i := 0
	ctx := srv.Context()
	for {
		if s.streamContextError(ctx) {
			break
		}
		tempRequest, tempRequestError := srv.Recv()
		if tempRequestError != nil {
			if tempRequestError == io.EOF {
				// 说明客户端已经发送完了数据
				s.logger.Info("ArchiveBlocks got EOF")
				break
			}
			s.logger.Errorf("ArchiveBlocks recv got error %s ", tempRequestError.Error())
			return tempRequestError
		}
		if len(tempRequest.ChainUnique) == 0 || tempRequest == nil ||
			tempRequest.Block == nil ||
			tempRequest.Block.Block == nil ||
			tempRequest.Block.Block.Header == nil {
			_ = srv.Send(&archivePb.ArchiveBlockResp{
				ArchiveStatus: archivePb.ArchiveStatus_ArchiveStatusFailed,
				Code:          uint32(archive_utils.CodeArchiveBlockNil),
				Message:       archive_utils.MsgArchiveBlockNil,
			})
			break
		}
		s.logger.Debugf("ArchiveBlocks.block %d", tempRequest.Block.Block.Header.BlockHeight)
		if i == 0 {
			s.ProxyProcessor.MarkInArchive(tempRequest.ChainUnique,
				s.ProxyProcessor.GetChainProcessorByHash)
			defer s.ProxyProcessor.MarkNotInArchive(tempRequest.ChainUnique,
				s.ProxyProcessor.GetChainProcessorByHash)
		}
		// 归档
		archiveStatus, resultErr := s.ProxyProcessor.ArchiveBlock(tempRequest.ChainUnique,
			tempRequest.Block, s.ProxyProcessor.GetChainProcessorByHash)
		resp := &archivePb.ArchiveBlockResp{
			ArchiveStatus: archiveStatus,
		}
		if archiveStatus == archivePb.ArchiveStatus_ArchiveStatusFailed {
			resp.Code = uint32(archive_utils.CodeArchiveBlockError)
			resp.Message = fmt.Sprintf("ArchiveBlock height %d ,got error %s",
				tempRequest.Block.Block.Header.BlockHeight, resultErr.Error())
		}
		sendError := srv.Send(resp)
		if archiveStatus == archivePb.ArchiveStatus_ArchiveStatusFailed {
			break
		}
		if sendError != nil {
			s.logger.Errorf("ArchiveBlocks height %d ,send error %s ",
				tempRequest.Block.Block.Header.BlockHeight,
				sendError.Error())
			return sendError
		}
		i++
	}
	return nil
}

// GetArchivedStatus 获取当前链归档区块信息
// @receiver s
// @param ctx
// @param req
// @return ArchiveStatusResp
// @return error
func (s *RpcServer) GetArchivedStatus(ctx context.Context,
	req *archivePb.ArchiveStatusRequest) (*archivePb.ArchiveStatusResp, error) {
	height, isInArchive, statusError := s.ProxyProcessor.GetInArchiveAndArchivedHeight(req.ChainUnique,
		s.ProxyProcessor.GetChainProcessorByHash)
	var resp archivePb.ArchiveStatusResp
	if statusError != nil {
		s.logger.Errorf("GetArchivedStatus error %s", statusError.Error())
		resp.Code = uint32(archive_utils.CodeGetArchiveStatusFail)
		resp.Message = statusError.Error()
		return &resp, nil
	}
	return &archivePb.ArchiveStatusResp{
		ArchivedHeight: height,
		InArchive:      isInArchive,
	}, nil
}

// GetRangeBlocks 获取指定范围内的区块数据
// @receiver s
// @param srv
// @param req
// @return error
func (s *RpcServer) GetRangeBlocks(req *archivePb.RangeBlocksRequest,
	srv archivePb.ArchiveCenterServer_GetRangeBlocksServer) error {
	archivedHeight, statusError := s.ProxyProcessor.GetArchivedHeight(req.ChainUnique,
		s.ProxyProcessor.GetChainProcessorByHash)
	if statusError != nil {
		s.logger.Error("GetRangeBlocks error %s", statusError.Error())
		return statusError
	}
	if req.StartHeight > req.EndHeight {
		return fmt.Errorf("invalid parameter")
	}
	if req.StartHeight > archivedHeight {
		return fmt.Errorf("archived height is [0,%d]", archivedHeight)
	}
	endHeight := req.EndHeight
	if endHeight > archivedHeight {
		endHeight = archivedHeight
	}
	for tempHeight := req.StartHeight; tempHeight <= endHeight; tempHeight++ {
		tempBlock, tempError := s.ProxyProcessor.GetBlockWithRWSetByHeight(req.ChainUnique,
			tempHeight, s.ProxyProcessor.GetChainProcessorByHash)
		if tempError != nil {
			s.logger.Errorf("GetRangeBlocks GetBlockWithRWSetByHeight %d got error %s",
				tempHeight, tempError.Error())
			return tempError
		}
		if tempBlock == nil {
			// 应该不会走到这里
			continue
		}
		sendError := srv.Send(tempBlock)
		if sendError != nil {
			s.logger.Errorf("GetRangeBlocks Send %d got error %s",
				tempHeight, sendError.Error())
			return sendError
		}
	}
	return nil
}

// GetBlockByHash 根据区块hash查询区块高度、区块数据、区块是否存在
// @receiver s
// @param ctx
// @param req
// @return BlockWithRWSetResp
// @return error
func (s *RpcServer) GetBlockByHash(ctx context.Context,
	req *archivePb.BlockByHashRequest) (*archivePb.BlockWithRWSetResp, error) {
	var resp archivePb.BlockWithRWSetResp
	hashHex, hashHexErr := hex.DecodeString(req.BlockHash)
	if hashHexErr != nil {
		resp.Code = uint32(archive_utils.CodeHexDecodeFailed)
		resp.Message = fmt.Sprintf("hex decode %s got error %s", req.BlockHash, hashHexErr.Error())
		return &resp, nil
	}
	if req.Operation == archivePb.OperationByHash_OperationBlockExists {
		// 仅仅查询区块是否存在,返回的block数据中，blockhash非空
		blockExist, existError := s.ProxyProcessor.BlockExists(req.ChainUnique, hashHex,
			s.ProxyProcessor.GetChainProcessorByHash)
		if existError != nil {
			s.logger.Errorf("GetBlockByHash BlockExists hash %s error %s",
				req.BlockHash, existError.Error())
			resp.Code = uint32(archive_utils.CodeBlockExistsFailed)
			resp.Message = existError.Error()
			return &resp, nil
		}
		if blockExist {
			resp.BlockData = &storePb.BlockWithRWSet{
				Block: &commonPb.Block{
					Header: &commonPb.BlockHeader{
						BlockHash: hashHex,
					},
				},
			}
			return &resp, nil
		}
		return &resp, nil

	} else if req.Operation == archivePb.OperationByHash_OperationGetBlockByHash {
		// 根据hash查询区块,进返回block数据
		tempBlock, tempError := s.ProxyProcessor.GetBlockByHash(req.ChainUnique, hashHex,
			s.ProxyProcessor.GetChainProcessorByHash)
		if tempError != nil {
			s.logger.Errorf("GetBlockByHash hash %s error %s",
				req.BlockHash, tempError.Error())
			resp.Code = uint32(archive_utils.CodeBlockByHashFailed)
			resp.Message = tempError.Error()
			return &resp, nil
		}
		if tempBlock == nil {
			return &resp, nil
		}
		resp.BlockData = &storePb.BlockWithRWSet{
			Block: tempBlock,
		}
		return &resp, nil
	} else if req.Operation == archivePb.OperationByHash_OperationGetHeightByHash {
		// 根据hash查询高度
		height, heightError := s.ProxyProcessor.GetHeightByHash(req.ChainUnique, hashHex,
			s.ProxyProcessor.GetChainProcessorByHash)
		if heightError != nil {
			s.logger.Errorf("GetBlockByHash GetHeightByHash %s error %s",
				req.BlockHash, heightError.Error())
			resp.Code = uint32(archive_utils.CodeHeightByHashFailed)
			resp.Message = heightError.Error()
			return &resp, nil
		}
		resp.BlockData = &storePb.BlockWithRWSet{
			Block: &commonPb.Block{
				Header: &commonPb.BlockHeader{
					BlockHeight: height,
				},
			},
		}
		return &resp, nil

	}
	resp.Code = uint32(archive_utils.CodeHashInvalidParam)
	resp.Message = fmt.Sprintf("bad request operation %d ", req.Operation)
	return &resp, nil
}

// GetBlockByHeight 根据高度查询区块头、查询区块信息
// @receiver s
// @param ctx
// @param req
// @return BlockWithRWSetResp
// @return error
func (s *RpcServer) GetBlockByHeight(ctx context.Context,
	req *archivePb.BlockByHeightRequest) (*archivePb.BlockWithRWSetResp, error) {
	var resp archivePb.BlockWithRWSetResp
	if req.Operation == archivePb.OperationByHeight_OperationGetBlockHeaderByHeight {
		// 根据高度查询区块头，仅返回header数据
		header, headerError := s.ProxyProcessor.GetBlockHeaderByHeight(req.ChainUnique, req.Height,
			s.ProxyProcessor.GetChainProcessorByHash)
		if headerError != nil {
			s.logger.Errorf("GetBlockByHeight GetBlockHeaderByHeight %d got error %s ",
				req.Height, headerError.Error())
			resp.Code = uint32(archive_utils.CodeHeaderByHeightFailed)
			resp.Message = headerError.Error()
			return &resp, nil
		}
		resp.BlockData = &storePb.BlockWithRWSet{
			Block: &commonPb.Block{
				Header: header,
			},
		}
		return &resp, nil
	} else if req.Operation == archivePb.OperationByHeight_OperationGetBlockByHeight {
		// 根据高度查询区块,仅返回block数据
		block, blockError := s.ProxyProcessor.GetBlock(req.ChainUnique,
			req.Height, s.ProxyProcessor.GetChainProcessorByHash)
		if blockError != nil {
			s.logger.Errorf("GetBlockByHeight GetBlock %d got error %s",
				req.Height, blockError.Error())
			resp.Code = uint32(archive_utils.CodeBlockByHeightFailed)
			resp.Message = blockError.Error()
			return &resp, nil
		}
		resp.BlockData = &storePb.BlockWithRWSet{
			Block: block,
		}
		return &resp, nil
	}
	resp.Code = uint32(archive_utils.CodeHeightInvalidParam)
	resp.Message = fmt.Sprintf("bad request operation %d", req.Operation)
	return &resp, nil
}

// GetBlockByTxId 根据txid查询block数据
// @receiver s
// @param ctx
// @param req
// @return BlockWithRWSetResp
// @return error
func (s *RpcServer) GetBlockByTxId(ctx context.Context,
	req *archivePb.BlockByTxIdRequest) (*archivePb.BlockWithRWSetResp, error) {
	var resp archivePb.BlockWithRWSetResp
	txHeight, txHeightError := s.ProxyProcessor.GetTxHeight(req.ChainUnique, req.TxId,
		s.ProxyProcessor.GetChainProcessorByHash)
	if txHeightError != nil {
		resp.Code = uint32(archive_utils.CodeBlockByTxIdFailed)
		resp.Message = txHeightError.Error()
		s.logger.Errorf("GetBlockByTxId txid %s got error %s",
			req.TxId, txHeightError.Error())
		return &resp, nil
	}
	block, blockError := s.ProxyProcessor.GetBlockWithRWSetByHeight(req.ChainUnique,
		txHeight, s.ProxyProcessor.GetChainProcessorByHash)
	if blockError != nil {
		resp.Code = uint32(archive_utils.CodeBlockByTxIdFailed)
		resp.Message = blockError.Error()
		s.logger.Errorf("GetBlockByTxId txid %s got error %s",
			req.TxId, blockError.Error())
		return &resp, nil
	}
	resp.BlockData = block
	return &resp, nil
}

// GetTxRWSetByTxId 根据txid查询读写集
// @receiver s
// @param ctx
// @param req
// @return TxRWSetResp
// @return error
func (s *RpcServer) GetTxRWSetByTxId(ctx context.Context,
	req *archivePb.BlockByTxIdRequest) (*archivePb.TxRWSetResp, error) {
	var resp archivePb.TxRWSetResp
	txRWSet, txError := s.ProxyProcessor.GetTxRWSet(req.ChainUnique, req.TxId,
		s.ProxyProcessor.GetChainProcessorByHash)
	if txError != nil {
		resp.Code = uint32(archive_utils.CodeTxRWSetByTxIdFailed)
		resp.Message = txError.Error()
		s.logger.Errorf("GetTxRWSetByTxId txid %s got error %s",
			req.TxId, txError.Error())
		return &resp, nil
	}
	resp.RwSet = txRWSet
	return &resp, nil
}

// GetTxByTxId 根据txid查询transaction
// @receiver s
// @param ctx
// @param req
// @return TransactionResp
// @return error
func (s *RpcServer) GetTxByTxId(ctx context.Context,
	req *archivePb.BlockByTxIdRequest) (*archivePb.TransactionResp, error) {
	var resp archivePb.TransactionResp
	transaction, transactionError := s.ProxyProcessor.GetTx(req.ChainUnique, req.TxId,
		s.ProxyProcessor.GetChainProcessorByHash)
	if transactionError != nil {
		resp.Code = uint32(archive_utils.CodeTxByTxIdFailed)
		resp.Message = transactionError.Error()
		return &resp, nil
	}
	resp.Transaction = transaction
	return &resp, nil
}

// GetLastConfigBlock 获取最新的配置区块
// @receiver s
// @param ctx
// @param req
// @return BlockWithRWSetResp
// @return error
func (s *RpcServer) GetLastConfigBlock(ctx context.Context,
	req *archivePb.ArchiveStatusRequest) (*archivePb.BlockWithRWSetResp, error) {
	var resp archivePb.BlockWithRWSetResp
	configBlock, configBlockError := s.ProxyProcessor.GetLastConfigBlock(req.ChainUnique,
		s.ProxyProcessor.GetChainProcessorByHash)
	if configBlockError != nil {
		resp.Code = uint32(archive_utils.CodeLastConfigFailed)
		resp.Message = configBlockError.Error()
		s.logger.Errorf("GetLastConfigBlock error %s", configBlockError.Error())
		return &resp, nil
	}
	resp.BlockData = &storePb.BlockWithRWSet{
		Block: configBlock,
	}
	return &resp, nil
}

// GetTxDetailByTxId 根据txid查询tx是否存在、tx所在区块高度、tx确认时间
// @receiver s
// @param ctx
// @param req
// @return TxDetailByIdResp
// @return error
func (s *RpcServer) GetTxDetailByTxId(ctx context.Context,
	req *archivePb.TxDetailByIdRequest) (*archivePb.TxDetailByIdResp, error) {
	var resp archivePb.TxDetailByIdResp
	if req.Operation == archivePb.OperationByTxId_OperationTxExists {
		// 判断tx是否存在
		exist, existError := s.ProxyProcessor.TxExists(req.ChainUnique, req.TxId,
			s.ProxyProcessor.GetChainProcessorByHash)
		if existError != nil {
			resp.Code = uint32(archive_utils.CodeTxExistsByTxIdFailed)
			resp.Message = existError.Error()
			s.logger.Errorf("GetTxDetailByTxId txid %s got error %s",
				req.TxId, existError.Error())
			return &resp, nil
		}
		resp.TxExist = exist
		return &resp, nil

	} else if req.Operation == archivePb.OperationByTxId_OperationGetTxHeight {
		// 获取tx所在区块高度
		height, heightError := s.ProxyProcessor.GetTxHeight(req.ChainUnique, req.TxId,
			s.ProxyProcessor.GetChainProcessorByHash)
		if heightError != nil {
			resp.Code = uint32(archive_utils.CodeTxHeightByTxIdFailed)
			resp.Message = heightError.Error()
			s.logger.Errorf("GetTxDetailByTxId GetTxHeight txid %s got error %s",
				req.TxId, heightError.Error())
			return &resp, nil
		}
		resp.Height = height
		return &resp, nil
	} else if req.Operation == archivePb.OperationByTxId_OperationGetTxConfirmedTime {
		// 获取交易confirm的时间
		confirmedTime, confirmedError := s.ProxyProcessor.GetTxConfirmedTime(req.ChainUnique, req.TxId,
			s.ProxyProcessor.GetChainProcessorByHash)
		if confirmedError != nil {
			resp.Code = uint32(archive_utils.CodeTxConfirmedByTxIdFailed)
			resp.Message = confirmedError.Error()
			return &resp, nil
		}
		resp.TxConfirmedTime = uint64(confirmedTime)
		return &resp, nil
	}
	resp.Code = uint32(archive_utils.CodeTxInvalidParam)
	resp.Message = fmt.Sprintf("bad request operation %d", req.Operation)
	return &resp, nil
}
