/*
Copyright (C) BABEC. All rights reserved.
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package chainmaker_sdk_go

import (
	"errors"
	"testing"

	"chainmaker.org/chainmaker/pb-go/v2/common"
	"github.com/stretchr/testify/require"
)

func TestChainClient_syncCanonicalTxResult(t *testing.T) {
	var txID = "b374f23e4e6747e4b5fcb3ca975ef1655ad56555adfd4534ae8676cd9f1eb145"
	var goodResult = &common.TransactionInfo{
		Transaction: &common.Transaction{
			Payload: &common.Payload{
				TxId: txID,
			},
			Result: &common.Result{
				Code:      0,
				RwSetHash: []byte("RwSetHash"),
				Message:   "OK",
			},
		},
	}
	goodResultBz, _ := goodResult.Marshal()

	tests := []struct {
		name         string
		cliTxReq     *common.TxRequest
		serverTxResp *common.TxResponse
		serverTxErr  error
		wantErr      bool
	}{
		{
			"bad",
			&common.TxRequest{Payload: &common.Payload{TxId: txID}},
			&common.TxResponse{
				Code: common.TxStatusCode_CONTRACT_FAIL,
				ContractResult: &common.ContractResult{
					Result:  []byte("some failed result"),
					Message: "failed",
				},
			},
			errors.New("rpc server throw an error"),
			true,
		},
		{
			"good",
			&common.TxRequest{Payload: &common.Payload{TxId: txID}},
			&common.TxResponse{
				Code: common.TxStatusCode_SUCCESS,
				ContractResult: &common.ContractResult{
					Result:  goodResultBz,
					Message: "OK",
				},
			},
			nil,
			false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cli, err := newMockChainClient(tt.serverTxResp, tt.serverTxErr, WithConfPath(sdkConfigPathForUT),
				WithEnableSyncCanonicalTxResult(true), WithRetryLimit(1), WithRetryInterval(100))
			require.Nil(t, err)
			defer cli.Stop()

			_, err = cli.syncCanonicalTxResult(txID)
			require.Equal(t, tt.wantErr, err != nil)
		})
	}
}
