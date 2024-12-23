/*
Copyright (C) BABEC. All rights reserved.
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package chainmaker_sdk_go

import (
	"testing"

	"chainmaker.org/chainmaker/pb-go/v2/common"
	"chainmaker.org/chainmaker/pb-go/v2/txpool"
	"github.com/stretchr/testify/require"
)

func TestChainClient_GetPoolStatus(t *testing.T) {
	tests := []struct {
		name       string
		serverResp *txpool.TxPoolStatus
	}{
		{
			"good",
			&txpool.TxPoolStatus{
				ConfigTxPoolSize:     10,
				CommonTxPoolSize:     1,
				ConfigTxNumInQueue:   0,
				ConfigTxNumInPending: 0,
				CommonTxNumInQueue:   0,
				CommonTxNumInPending: 0,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cli, err := newMockChainClient(nil, nil, WithConfPath(sdkConfigPathForUT))
			require.Nil(t, err)
			defer cli.Stop()
			_mockServer.getPoolStatusResp = tt.serverResp

			resp, err := cli.GetPoolStatus()
			require.Nil(t, err)
			require.Equal(t, tt.serverResp, resp)
		})
	}
}

func TestChainClient_GetTxIdsByTypeAndStage(t *testing.T) {
	tests := []struct {
		name       string
		serverResp *txpool.GetTxIdsByTypeAndStageResponse
	}{
		{
			"good",
			&txpool.GetTxIdsByTypeAndStageResponse{TxIds: []string{"123"}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cli, err := newMockChainClient(nil, nil, WithConfPath(sdkConfigPathForUT))
			require.Nil(t, err)
			defer cli.Stop()
			_mockServer.getTxIdsByTypeAndStageResp = tt.serverResp

			txIds, err := cli.GetTxIdsByTypeAndStage(txpool.TxType_ALL_TYPE, txpool.TxStage_ALL_STAGE)
			require.Nil(t, err)
			require.Equal(t, tt.serverResp.TxIds, txIds)
		})
	}
}

func TestChainClient_GetTxsInPoolByTxIds(t *testing.T) {
	tests := []struct {
		name       string
		serverResp *txpool.GetTxsInPoolByTxIdsResponse
	}{
		{
			"good",
			&txpool.GetTxsInPoolByTxIdsResponse{Txs: []*common.Transaction{
				{
					Payload: &common.Payload{
						ChainId: "chain1",
					},
					Result: &common.Result{
						ContractResult: &common.ContractResult{
							Result: []byte("result"),
						},
					},
				},
			}, TxIds: []string{"123"}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cli, err := newMockChainClient(nil, nil, WithConfPath(sdkConfigPathForUT))
			require.Nil(t, err)
			defer cli.Stop()
			_mockServer.getTxsInPoolByTxIdsResp = tt.serverResp

			txs, txIds, err := cli.GetTxsInPoolByTxIds(tt.serverResp.TxIds)
			require.Nil(t, err)
			require.Equal(t, tt.serverResp.Txs, txs)
			require.Equal(t, tt.serverResp.TxIds, txIds)
		})
	}
}
