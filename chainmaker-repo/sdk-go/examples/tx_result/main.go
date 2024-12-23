/*
Copyright (C) BABEC. All rights reserved.
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package main

import (
	"fmt"
	"log"
	"strconv"
	"sync/atomic"
	"time"

	"chainmaker.org/chainmaker/common/v2/random/uuid"
	"chainmaker.org/chainmaker/pb-go/v2/common"
	sdk "chainmaker.org/chainmaker/sdk-go/v2"
	"chainmaker.org/chainmaker/sdk-go/v2/examples"
)

var (
	threadNum         = 10
	claimContractName = fmt.Sprintf("claim%d", time.Now().UnixNano())
	claimVersion      = "2.0.0"
	claimByteCodePath = "../../testdata/claim-wasm-demo/rust-fact-2.0.0.wasm"

	sdkConfigOrg1Client1Path = "../sdk_configs/sdk_config_org1_client1.yml"
)

func main() {
	testUserContractClaim()
}

func testUserContractClaim() {
	cc, err := sdk.NewChainClient(
		sdk.WithConfPath(sdkConfigOrg1Client1Path),
		sdk.WithEnableTxResultDispatcher(true),
		sdk.WithRetryLimit(20),
		sdk.WithRetryInterval(2000),
	)
	if err != nil {
		log.Fatalln(err)
	}

	fmt.Println("====================== 创建合约 ======================")
	usernames := []string{examples.UserNameOrg1Admin1, examples.UserNameOrg2Admin1, examples.UserNameOrg3Admin1, examples.UserNameOrg4Admin1}
	testUserContractClaimCreate(cc, true, usernames...)

	fmt.Println("====================== 调用合约 ======================")
	var txCount int64
	var calcTpsDuration = time.Second * 5
	for i := 0; i < threadNum; i++ {
		go func() {
			for {
				resp, err := testUserContractClaimInvoke(cc, "save", true)
				if err != nil {
					fmt.Printf("%s\ninvoke contract resp: %+v\n", err, resp)
				} else {
					atomic.AddInt64(&txCount, 1)
				}
			}
		}()
	}
	for {
		txNum := atomic.SwapInt64(&txCount, 0)
		tps := float64(txNum) / calcTpsDuration.Seconds()
		fmt.Printf("TPS: %.2f\n", tps)
		time.Sleep(calcTpsDuration)
	}
}

func testUserContractClaimCreate(client *sdk.ChainClient, withSyncResult bool, usernames ...string) {

	resp, err := createUserContract(client, claimContractName, claimVersion, claimByteCodePath,
		common.RuntimeType_WASMER, []*common.KeyValuePair{}, withSyncResult, usernames...)
	if err != nil {
		log.Fatalln(err)
	}

	fmt.Printf("CREATE claim contract resp: %+v\n", resp)
}

func createUserContract(client *sdk.ChainClient, contractName, version, byteCodePath string, runtime common.RuntimeType,
	kvs []*common.KeyValuePair, withSyncResult bool, usernames ...string) (*common.TxResponse, error) {

	payload, err := client.CreateContractCreatePayload(contractName, version, byteCodePath, runtime, kvs)
	if err != nil {
		return nil, err
	}

	//endorsers, err := examples.GetEndorsers(payload, usernames...)
	endorsers, err := examples.GetEndorsersWithAuthType(client.GetHashType(),
		client.GetAuthType(), payload, usernames...)
	if err != nil {
		return nil, err
	}

	return client.SendContractManageRequest(payload, endorsers, 20, withSyncResult)
}

func testUserContractClaimInvoke(client *sdk.ChainClient,
	method string, withSyncResult bool) (*common.TxResponse, error) {

	curTime := strconv.FormatInt(time.Now().Unix(), 10)

	fileHash := uuid.GetUUID()
	kvs := []*common.KeyValuePair{
		{
			Key:   "time",
			Value: []byte(curTime),
		},
		{
			Key:   "file_hash",
			Value: []byte(fileHash),
		},
		{
			Key:   "file_name",
			Value: []byte(fmt.Sprintf("file_%s", curTime)),
		},
	}

	return client.InvokeContract(claimContractName, method, "", kvs, -1, withSyncResult)
}
