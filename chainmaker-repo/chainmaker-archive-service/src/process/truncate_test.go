/*
 * Copyright (C) BABEC. All rights reserved.
 * Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.
 *
 * SPDX-License-Identifier: Apache-2.0
 */

package process

import (
	"testing"

	"chainmaker.org/chainmaker/pb-go/v2/common"
	"github.com/stretchr/testify/assert"
)

func TestTruncateConfig_TruncateBlock(t *testing.T) {
	block := mockBlock()
	assert.Equal(t, 6, len(block.Txs[0].Payload.Parameters[0].Value))

	cfg := newTruncateConfig(2, "empty")
	cfg.TruncateBlock(block)
	assert.Equal(t, 0, len(block.Txs[0].Payload.Parameters[0].Value))
}
func TestTruncateConfig_TruncateTx(t *testing.T) {
	tx := mockTx()
	assert.Equal(t, 6, len(tx.Payload.Parameters[0].Value))
	for _, p := range tx.Payload.Parameters {
		t.Logf("key:%s,value:%x", p.Key, p.Value)
	}
	cfg := newTruncateConfig(2, "hash")
	cfg.TruncateTx(tx)
	for _, p := range tx.Payload.Parameters {
		assert.Equal(t, 32, len(p.Value))
		t.Logf("key:%s,value:%x", p.Key, p.Value)
	}
}

func TestTruncateConfig_TruncateBlockWithRWSet(t *testing.T) {
	b := &common.BlockInfo{
		Block:     mockBlock(),
		RwsetList: []*common.TxRWSet{mockRWSet()},
	}

	cfg := newTruncateConfig(8, "truncate")
	cfg.TruncateBlockWithRWSet(b)
	t.Logf("%v", b.RwsetList[0])
}

func mockTx() *common.Transaction {
	return &common.Transaction{
		Payload: &common.Payload{
			ChainId:        "",
			TxType:         0,
			TxId:           "tx1",
			Timestamp:      0,
			ExpirationTime: 0,
			ContractName:   "",
			Method:         "",
			Parameters: []*common.KeyValuePair{
				{
					Key:   "key1",
					Value: []byte("value1"),
				},
				{
					Key:   "key2",
					Value: []byte("22222222222222222222222222222222222222222222"),
				},
				{
					Key:   "key3",
					Value: make([]byte, 1000),
				},
			},
			Sequence: 0,
			Limit:    nil,
		},
		Sender:    nil,
		Endorsers: nil,
		Result:    nil,
	}
}
func mockBlock() *common.Block {
	return &common.Block{
		Header: &common.BlockHeader{
			BlockVersion:   0,
			BlockType:      0,
			ChainId:        "",
			BlockHeight:    1,
			BlockHash:      nil,
			PreBlockHash:   nil,
			PreConfHeight:  0,
			TxCount:        0,
			TxRoot:         nil,
			DagHash:        nil,
			RwSetRoot:      nil,
			BlockTimestamp: 0,
			ConsensusArgs:  nil,
			Proposer:       nil,
			Signature:      nil,
		},
		Dag:            nil,
		Txs:            []*common.Transaction{mockTx()},
		AdditionalData: nil,
	}

}

func mockRWSet() *common.TxRWSet {
	return &common.TxRWSet{
		TxId: "tx1",
		TxReads: []*common.TxRead{
			{
				Key:          []byte("key1"),
				Value:        make([]byte, 500),
				ContractName: "c1",
				Version:      nil,
			},
		},
		TxWrites: []*common.TxWrite{
			{
				Key:          []byte("key1"),
				Value:        make([]byte, 200),
				ContractName: "c1",
			},
		},
	}
}
