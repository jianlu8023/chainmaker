/*
Copyright (C) BABEC. All rights reserved.


SPDX-License-Identifier: Apache-2.0
*/

// Package archive_utils define common utils
package archive_utils

import "encoding/hex"

// EncodeGenesisHashToString blockhash转为string
func EncodeGenesisHashToString(blockHash []byte) string {
	return hex.EncodeToString(blockHash)
}

// DecodeStringToGenesisHash 从string中解析出byte数组
func DecodeStringToGenesisHash(hashString string) ([]byte, error) {
	return hex.DecodeString(hashString)
}
