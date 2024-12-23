/*
Copyright (C) BABEC. All rights reserved.
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package main

import (
	"fmt"
	"log"

	"chainmaker.org/chainmaker/pb-go/v2/txpool"
	sdk "chainmaker.org/chainmaker/sdk-go/v2"
	"chainmaker.org/chainmaker/sdk-go/v2/examples"
)

const (
	sdkConfigOrg1Client1Path = "../sdk_configs/sdk_config_org1_client1.yml"
)

func main() {
	client, err := examples.CreateChainClientWithSDKConf(sdkConfigOrg1Client1Path)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("====================== 获取交易池状态 ======================")
	if err := getPoolStatus(client); err != nil {
		log.Fatal(err)
	}
	fmt.Println("====================== 获取不同交易类型和阶段中的交易Id列表 ======================")
	if err := getTxIdsByTypeAndStage(client); err != nil {
		log.Fatal(err)
	}
	fmt.Println("====================== 根据txIds获取交易池中存在的txs，并返回交易池缺失的tx的txIds ======================")
	if err := getTxsInPoolByTxIds(client); err != nil {
		log.Fatal(err)
	}
}

func getPoolStatus(cc *sdk.ChainClient) error {
	resp, err := cc.GetPoolStatus()
	if err != nil {
		return err
	}
	examples.PrintPrettyJson(resp)
	return nil
}

func getTxIdsByTypeAndStage(cc *sdk.ChainClient) error {
	txIds, err := cc.GetTxIdsByTypeAndStage(txpool.TxType_ALL_TYPE, txpool.TxStage_ALL_STAGE)
	if err != nil {
		return err
	}
	examples.PrintPrettyJson(txIds)
	return nil
}

func getTxsInPoolByTxIds(cc *sdk.ChainClient) error {
	txs, txIds, err := cc.GetTxsInPoolByTxIds([]string{"not exists tx id"})
	if err != nil {
		return err
	}
	examples.PrintPrettyJson(txs)
	examples.PrintPrettyJson(txIds)
	return nil
}
