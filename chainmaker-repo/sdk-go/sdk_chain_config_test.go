/*
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package chainmaker_sdk_go

import (
	"strconv"
	"testing"

	"github.com/gogo/protobuf/proto"
	"github.com/stretchr/testify/require"

	"chainmaker.org/chainmaker/pb-go/v2/common"
	"chainmaker.org/chainmaker/pb-go/v2/config"
)

func TestGetChainConfig(t *testing.T) {
	goodChainConfigBz, err := proto.Marshal(&config.ChainConfig{Crypto: &config.CryptoConfig{Hash: "SHA256"}})
	require.Nil(t, err)

	tests := []struct {
		name         string
		serverTxResp *common.TxResponse
		serverTxErr  error
		wantErr      bool
	}{
		{
			"bad",
			&common.TxResponse{
				Code: common.TxStatusCode_SUCCESS,
				ContractResult: &common.ContractResult{
					Result: []byte("this is a bad chain config bytes"),
				},
			},
			nil,
			true,
		},
		{
			"good",
			&common.TxResponse{
				Code: common.TxStatusCode_SUCCESS,
				ContractResult: &common.ContractResult{
					Result: goodChainConfigBz,
				},
			},
			nil,
			false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cli, err := newMockChainClient(tt.serverTxResp, tt.serverTxErr, WithConfPath(sdkConfigPathForUT))
			require.Nil(t, err)
			defer cli.Stop()

			chainConf, err := cli.GetChainConfig()
			require.Equal(t, tt.wantErr, err != nil)

			if chainConf != nil {
				bz, err := proto.Marshal(chainConf)
				require.Nil(t, err)
				require.Equal(t, tt.serverTxResp.ContractResult.Result, bz)
			}
		})
	}
}

func TestChainClient_CreateChainConfigOptimizeChargeGasPayload(t *testing.T) {
	chainConfigBz, err := proto.Marshal(&config.ChainConfig{
		Sequence: 1,
	})
	require.Nil(t, err)
	tests := []struct {
		name         string
		enable       bool
		serverTxResp *common.TxResponse
	}{
		{
			"true",
			true,
			&common.TxResponse{
				Code: common.TxStatusCode_SUCCESS,
				ContractResult: &common.ContractResult{
					Result: chainConfigBz,
				},
			},
		},
		{
			"false",
			false,
			&common.TxResponse{
				Code: common.TxStatusCode_SUCCESS,
				ContractResult: &common.ContractResult{
					Result: chainConfigBz,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cli, err := newMockChainClient(tt.serverTxResp, nil, WithConfPath(sdkConfigPathForUT))
			require.Nil(t, err)
			defer cli.Stop()

			payload, err := cli.CreateChainConfigOptimizeChargeGasPayload(tt.enable)
			require.Nil(t, err)
			require.Equal(t, len(payload.Parameters), 1)
			actual, err := strconv.ParseBool(string(payload.Parameters[0].Value))
			require.Nil(t, err)
			require.Equal(t, tt.enable, actual)
		})
	}
}
