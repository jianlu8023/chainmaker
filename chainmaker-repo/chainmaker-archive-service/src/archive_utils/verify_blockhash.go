/*
Copyright (C) BABEC. All rights reserved.


SPDX-License-Identifier: Apache-2.0
*/

// Package archive_utils define common utils
package archive_utils

import (
	"bytes"

	commonPb "chainmaker.org/chainmaker/pb-go/v2/common"
	"chainmaker.org/chainmaker/utils/v2"
)

// VerifyBlockHash 验证区块hash头
// @param hashType
// @pram block
// @return bool
// @return error
func VerifyBlockHash(hashType string, block *commonPb.Block) (bool, error) {
	calculateBlockHash, hashError := utils.CalcBlockHash(hashType, block)
	if hashError != nil {
		return false, hashError
	}
	if !bytes.Equal(calculateBlockHash, block.Header.BlockHash) {
		return false, nil
	}
	return true, nil
}

// // IsDagHashValid to check if block dag equals with simulated block dag
// // @param block
// // @param hashType
// // @return error
// func IsDagHashValid(block *commonPb.Block, hashType string) error {
// 	dagHash, err := utils.CalcDagHash(hashType, block.Dag)
// 	if err != nil || !bytes.Equal(dagHash, block.Header.DagHash) {
// 		return fmt.Errorf("dag expect %x, got %x", block.Header.DagHash, dagHash)
// 	}
// 	return nil
// }

// // IsRWSetHashValid to check if read write set is valid
// // @param block
// // @param hashType
// // @param error
// func IsRWSetHashValid(block *commonPb.Block, hashType string) error {
// 	rwSetRoot, err := utils.CalcRWSetRoot(hashType, block.Txs)
// 	if err != nil {
// 		return fmt.Errorf("calc rwset error, %s", err)
// 	}
// 	if !bytes.Equal(rwSetRoot, block.Header.RwSetRoot) {
// 		return fmt.Errorf("rwset expect %x, got %x", block.Header.RwSetRoot, rwSetRoot)
// 	}
// 	return nil
// }

// // IsMerkleRootValid to check if block merkle root equals with simulated merkle root
// // @param block
// // @pram txHashes
// // @param hashType
// // @return error
// func IsMerkleRootValid(block *commonPb.Block, txHashes [][]byte, hashType string) error {
// 	txRoot, err := hash.GetMerkleRoot(hashType, txHashes)
// 	if err != nil || !bytes.Equal(txRoot, block.Header.TxRoot) {
// 		return fmt.Errorf("GetMerkleRoot(%s,%v) get %x ,txroot expect %x, got %x, err: %s",
// 			hashType, txHashes, txRoot, block.Header.TxRoot, txRoot, err)
// 	}
// 	return nil
// }
