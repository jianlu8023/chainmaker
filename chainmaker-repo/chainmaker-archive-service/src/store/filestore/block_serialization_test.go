/*
Copyright (C) BABEC. All rights reserved.


SPDX-License-Identifier: Apache-2.0
*/

// Package filestore define file operation
package filestore

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"reflect"
	"testing"

	acPb "chainmaker.org/chainmaker/pb-go/v2/accesscontrol"

	commonPb "chainmaker.org/chainmaker/pb-go/v2/common"
	storePb "chainmaker.org/chainmaker/pb-go/v2/store"
	"github.com/gogo/protobuf/proto"
	"github.com/stretchr/testify/assert"
)

var chainId = "testchain1"

func generateBlockHash(chainId string, height uint64) []byte {
	blockHash := sha256.Sum256([]byte(fmt.Sprintf("%s-%d", chainId, height)))
	return blockHash[:]
}

func generateTxId(chainId string, height uint64, index int) string {
	txIdBytes := sha256.Sum256([]byte(fmt.Sprintf("%s-%d-%d", chainId, height, index)))
	return hex.EncodeToString(txIdBytes[:32])
}

func createBlockAndRWSets(chainId string, height uint64, txNum int) *storePb.BlockWithRWSet {
	block := &commonPb.Block{
		Header: &commonPb.BlockHeader{
			ChainId:     chainId,
			BlockHeight: height,
		},
	}

	for i := 0; i < txNum; i++ {
		tx := &commonPb.Transaction{
			Payload: &commonPb.Payload{
				ChainId: chainId,
				TxId:    generateTxId(chainId, height, i),
			},
			Sender: &commonPb.EndorsementEntry{
				Signer: &acPb.Member{
					OrgId:      "org1",
					MemberInfo: []byte("User1"),
				},
				Signature: []byte("signature1"),
			},
			Result: &commonPb.Result{
				Code: commonPb.TxStatusCode_SUCCESS,
				ContractResult: &commonPb.ContractResult{
					Result: []byte("ok"),
				},
			},
		}
		block.Txs = append(block.Txs, tx)
	}

	block.Header.BlockHash = generateBlockHash(chainId, height)
	var txRWSets []*commonPb.TxRWSet
	for i := 0; i < txNum; i++ {
		key := fmt.Sprintf("key_%d", i)
		value := fmt.Sprintf("value_%d", i)
		txRWset := &commonPb.TxRWSet{
			TxId: block.Txs[i].Payload.TxId,
			TxWrites: []*commonPb.TxWrite{
				{
					Key:          []byte(key),
					Value:        []byte(value),
					ContractName: "contract1",
				},
			},
		}
		txRWSets = append(txRWSets, txRWset)
	}
	var events []*commonPb.ContractEvent
	for i := 0; i < txNum; i++ {
		events = append(events, &commonPb.ContractEvent{
			Topic:           fmt.Sprintf("topic_%d", i),
			TxId:            block.Txs[i].Payload.TxId,
			ContractName:    fmt.Sprintf("contract_%d", i),
			ContractVersion: "1.0",
			EventData:       []string{fmt.Sprintf("event_%d", i)},
		})
	}

	return &storePb.BlockWithRWSet{Block: block, TxRWSets: txRWSets, ContractEvents: events}
}

func TestSerializeBlock(t *testing.T) {
	for i := 0; i < 10; i++ {
		block := createBlockAndRWSets(chainId, uint64(i), 5000)
		bytes, blockInfo, err := SerializeBlock(block)
		for i, v := range blockInfo.TxsIndex {
			txBytes := bytes[v.Offset : v.Offset+v.ByteLen]
			rwSet := &commonPb.Transaction{}
			errors := proto.Unmarshal(txBytes, rwSet)
			assert.Nil(t, errors)
			assert.Equal(t, blockInfo.Meta.TxIds[i], rwSet.Payload.TxId)
			//break
		}
		assert.Nil(t, err)
		assert.Equal(t, blockInfo.Block.String(), block.Block.String())
		assert.Equal(t, len(block.Block.Txs), len(blockInfo.SerializedTxs))
		assert.Equal(t, len(block.TxRWSets), len(blockInfo.TxRWSets))
		result, err := DeserializeBlock(bytes)
		assert.Nil(t, err)
		assert.Equal(t, block.Block.String(), result.Block.String())
		blockInfo.ReSet()
	}
}

func TestDeserializeMeta(t *testing.T) {
	for i := 0; i < 10; i++ {
		block := createBlockAndRWSets(chainId, uint64(i), 5000)
		bytes, blockInfo, err := serializeBlockParallelForTest(block)
		assert.Nil(t, err)
		assert.NotNil(t, blockInfo)
		deserialBlock, deserialErr := DeserializeBlock(bytes)
		assert.Nil(t, deserialErr)
		isSame := reflect.DeepEqual(*block, *deserialBlock)
		assert.True(t, isSame)
	}
}

func serializeBlockParallelForTest(blockWithRWSet *storePb.BlockWithRWSet) ([]byte, *BlockWithSerializedInfo, error) {

	buf := proto.NewBuffer(nil)
	block := blockWithRWSet.Block
	txRWSets := blockWithRWSet.TxRWSets
	events := blockWithRWSet.ContractEvents
	info := &BlockWithSerializedInfo{}
	info.Block = block
	meta := &storePb.SerializedBlock{
		Header:         block.Header,
		Dag:            block.Dag,
		TxIds:          make([]string, 0, len(block.Txs)),
		AdditionalData: block.AdditionalData,
	}

	for _, tx := range block.Txs {
		meta.TxIds = append(meta.TxIds, tx.Payload.TxId)
		info.Txs = append(info.Txs, tx)
	}

	info.TxRWSets = append(info.TxRWSets, txRWSets...)
	info.Meta = meta
	info.ContractEvents = events
	//序列化 meta
	if err := info.SerializeMeta(buf); err != nil {
		return nil, nil, err
	}
	//序列化 txs 交易
	if err := info.SerializeTxs(buf); err != nil {
		return nil, nil, err
	}
	//序列化 交易读写集
	if err := info.SerializeTxRWSets(buf); err != nil {
		return nil, nil, err
	}
	//序列化 ContractEvents
	if err := info.SerializeEventTopicTable(buf); err != nil {
		return nil, nil, err
	}

	return buf.Bytes(), info, nil
}

func TestDeserializeMeta2(t *testing.T) {
	NewBlockSerializedInfo()
}
