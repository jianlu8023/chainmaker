/*
Copyright (C) BABEC. All rights reserved.
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package chainmaker_sdk_go

import (
	"context"
	"errors"
	"fmt"
	"time"

	"chainmaker.org/chainmaker/common/v2/crypto"
	"chainmaker.org/chainmaker/common/v2/crypto/hash"
	"chainmaker.org/chainmaker/pb-go/v2/common"
	"chainmaker.org/chainmaker/pb-go/v2/syscontract"
	"chainmaker.org/chainmaker/sdk-go/v2/utils"
	"github.com/Rican7/retry"
	"github.com/Rican7/retry/strategy"
	"github.com/gogo/protobuf/proto"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// syncCanonicalTxResult 同步获取权威的公认的交易结果，即超过半数共识的交易
func (cc *ChainClient) syncCanonicalTxResult(txId string) (*txResult, error) {
	txResultC := make(chan *txResult, 1)
	defer close(txResultC)
	txResultCount := make(map[string]int)
	receiveCount := 0
	canonicalNum := (len(cc.canonicalTxFetcherPools) / 2) + 1
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	for nodeAddr, pool := range cc.canonicalTxFetcherPools {
		cc.logger.Debugf("node [%s] start canonicalPollingTxResult", nodeAddr)
		go canonicalPollingTxResult(ctx, cc, pool, txId, txResultC)
	}

	timeout := time.Duration(cc.retryInterval*cc.retryLimit) * time.Millisecond
	ticker := time.NewTicker(timeout)
	defer ticker.Stop()
	for {
		select {
		case r := <-txResultC:
			if r != nil {
				bz, err := proto.Marshal(r.Result)
				if err != nil {
					return nil, err
				}
				sum, err := hash.Get(crypto.HASH_TYPE_SHA256, bz)
				if err != nil {
					return nil, err
				}
				sumStr := string(sum)
				if count, ok := txResultCount[sumStr]; ok {
					txResultCount[sumStr] = count + 1
				} else {
					txResultCount[sumStr] = 1
				}
				if txResultCount[sumStr] >= canonicalNum {
					return r, nil
				}
			}
			receiveCount++
			if receiveCount >= len(cc.canonicalTxFetcherPools) {
				return nil, errors.New("syncCanonicalTxResult failed")
			}
		case <-ticker.C:
			return nil, fmt.Errorf("get canonical transaction result timed out, timeout=%s", timeout)
		}
	}
}

func canonicalPollingTxResult(ctx context.Context, cc *ChainClient, pool ConnectionPool,
	txId string, txResultC chan *txResult) error {
	var (
		txInfo *common.TransactionInfo
		err    error
	)

	select {
	case <-ctx.Done():
		return nil
	default:
		err = retry.Retry(func(uint) error {
			txInfo, err = canonicalGetTxByTxId(cc, pool, txId)
			if err != nil {
				return err
			}
			return nil
		},
			strategy.Wait(time.Duration(cc.retryInterval)*time.Millisecond),
			strategy.Limit(uint(cc.retryLimit)),
		)

		if err != nil {
			select {
			case <-ctx.Done():
				return nil
			default:
			}
			txResultC <- nil
			return fmt.Errorf("get tx by txId [%s] failed, %s", txId, err)
		}

		if txInfo == nil || txInfo.Transaction == nil || txInfo.Transaction.Result == nil {
			select {
			case <-ctx.Done():
				return nil
			default:
			}
			txResultC <- nil
			return fmt.Errorf("get result by txId [%s] failed, %+v", txId, txInfo)
		}

		select {
		case <-ctx.Done():
			return nil
		default:
		}
		txResultC <- &txResult{
			Result:        txInfo.Transaction.Result,
			TxTimestamp:   txInfo.Transaction.Payload.Timestamp,
			TxBlockHeight: txInfo.BlockHeight,
		}
		return nil
	}
}

func canonicalGetTxByTxId(cc *ChainClient, pool ConnectionPool, txId string) (*common.TransactionInfo, error) {
	cc.logger.Debugf("[SDK] begin canonicalGetTxByTxId, [method:%s]/[txId:%s]",
		syscontract.ChainQueryFunction_GET_TX_BY_TX_ID, txId)

	payload := cc.CreatePayload("", common.TxType_QUERY_CONTRACT, syscontract.SystemContract_CHAIN_QUERY.String(),
		syscontract.ChainQueryFunction_GET_TX_BY_TX_ID.String(), []*common.KeyValuePair{
			{
				Key:   utils.KeyBlockContractTxId,
				Value: []byte(txId),
			},
		}, defaultSeq, nil,
	)

	req, err := cc.GenerateTxRequest(payload, nil)
	if err != nil {
		return nil, err
	}

	var errMsg string
	var resp *common.TxResponse
	logger := pool.getLogger()
	ignoreAddrs := make(map[string]struct{})
	for {
		client, err := pool.getClientWithIgnoreAddrs(ignoreAddrs)
		if err != nil {
			return nil, err
		}

		if len(ignoreAddrs) > 0 {
			logger.Debugf("[SDK] begin try to connect node [%s]", client.ID)
		}

		resp, err = client.sendRequestWithTimeout(req, DefaultGetTxTimeout)
		if err != nil {
			resp = &common.TxResponse{
				Message: err.Error(),
				TxId:    req.Payload.TxId,
			}

			statusErr, ok := status.FromError(err)
			if ok && (statusErr.Code() == codes.DeadlineExceeded ||
				// desc = "transport: Error while dialing dial tcp 127.0.0.1:12301: connect: connection refused"
				statusErr.Code() == codes.Unavailable) {

				resp.Code = common.TxStatusCode_TIMEOUT
				errMsg = fmt.Sprintf("call [%s] meet network error, try to connect another node if has, %s",
					client.ID, err.Error())

				logger.Errorf(sdkErrStringFormat, errMsg)
				ignoreAddrs[client.ID] = struct{}{}
				continue
			}

			logger.Errorf("statusErr.Code() : %s", statusErr.Code())

			resp.Code = common.TxStatusCode_INTERNAL_ERROR
			errMsg = fmt.Sprintf("client.call failed, %+v", err)
			logger.Errorf(sdkErrStringFormat, errMsg)
			break
		}

		resp.TxId = req.Payload.TxId
		logger.Debugf("[SDK] proposalRequest resp: %+v", resp)
		break
	}

	if err = utils.CheckProposalRequestResp(resp, true); err != nil {
		if utils.IsArchived(resp.Code) {
			return nil, errors.New(resp.Code.String())
		}
		return nil, fmt.Errorf(errStringFormat, payload.TxType, err)
	}

	transactionInfo := &common.TransactionInfo{}
	if err = proto.Unmarshal(resp.ContractResult.Result, transactionInfo); err != nil {
		return nil, fmt.Errorf("canonicalGetTxByTxId unmarshal transaction info payload failed, %s", err)
	}

	return transactionInfo, nil
}
