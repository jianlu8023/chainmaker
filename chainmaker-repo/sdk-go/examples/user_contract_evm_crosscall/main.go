/*
Copyright (C) BABEC. All rights reserved.
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package main

import (
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"log"
	"math/big"
	"strconv"
	"strings"
	"time"

	"chainmaker.org/chainmaker/common/v2/evmutils"
	"chainmaker.org/chainmaker/common/v2/evmutils/abi"
	"chainmaker.org/chainmaker/common/v2/random/uuid"
	"chainmaker.org/chainmaker/pb-go/v2/common"
	sdk "chainmaker.org/chainmaker/sdk-go/v2"
	"chainmaker.org/chainmaker/sdk-go/v2/examples"
)

const (
	createContractTimeout = 5
	contractVersion       = "1.0.0"

	calleeName = "callee"
	calleeBin  = "../../testdata/cross-call-evm-demo/Callee.bin"

	callerName = "caller"
	callerBin  = "../../testdata/cross-call-evm-demo/Caller.bin"
	callerABI  = "../../testdata/cross-call-evm-demo/Caller.abi"

	crossVmName = "crossVm"
	crossVmBin  = "../../testdata/cross-call-evm-demo/CrossCall.bin"
	crossVmABI  = "../../testdata/cross-call-evm-demo/CrossCall.abi"

	claimName                = "claim001"
	claimVersion             = "2.0.0"
	claimBinPath             = "../../testdata/claim-wasm-demo/rust-fact-2.0.0.wasm"
	sdkConfigOrg1Client1Path = "../sdk_configs/sdk_config_org1_client1.yml"
)

func main() {
	//testCrossCall(sdkConfigOrg1Client1Path)
	testCrossVmCall(sdkConfigOrg1Client1Path)
}

func testCrossVmCall(sdkPath string) {
	fmt.Println("====================== create client ======================")
	client, err := examples.CreateChainClientWithSDKConf(sdkPath)
	if err != nil {
		log.Fatalln(err)
	}

	usernames := []string{examples.UserNameOrg1Admin1, examples.UserNameOrg2Admin1, examples.UserNameOrg3Admin1, examples.UserNameOrg4Admin1}
	fmt.Println("====================== 创建被调用者claim合约(rust版本) ======================")
	testCreateClaimContract(client, true, usernames...)

	fmt.Println("====================== 创建caller(调用者)合约 ======================")
	testCreateCrossVmContract(client, true, true, usernames...)

	fmt.Println("====================== caller跨合约调用claim的save方法 ======================")
	fileHash := testCallerCrossCallClaimSaveMethod(client, true)

	fmt.Println("====================== 查询claim(rust)合约 ======================")
	kvs := []*common.KeyValuePair{
		{
			Key:   "file_hash",
			Value: []byte(fileHash),
		},
	}
	testUserContractClaimQuery(client, "find_by_file_hash", kvs)
}

func testUserContractClaimQuery(client *sdk.ChainClient, method string, kvs []*common.KeyValuePair) {
	resp, err := client.QueryContract(claimName, method, kvs, -1)
	if err != nil {
		log.Fatalln(err)
	}

	fmt.Printf("QUERY claim contract resp: %+v\n", resp)
}

func testCreateCrossVmContract(client *sdk.ChainClient, withSyncResult bool, isIgnoreSameContract bool, usernames ...string) {

	codeBytes, err := ioutil.ReadFile(crossVmBin)
	if err != nil {
		log.Fatalln(err)
	}

	resp, err := createUserContract(client, crossVmName, contractVersion, string(codeBytes), common.RuntimeType_EVM,
		nil, withSyncResult, usernames...)
	if !isIgnoreSameContract {
		if err != nil {
			log.Fatalln(err)
		}
	}

	fmt.Printf("CREATE EVM cross vm call contract resp: [code:%d]/[msg:%s]\n", resp.Code, resp.Message)
	fmt.Printf("contract result: [code:%d]/[msg:%s]/[contractResult:%+X]\n", resp.ContractResult.Code,
		resp.ContractResult.Message, resp.ContractResult.Result)
}

func testCallerCrossCallClaimSaveMethod(client *sdk.ChainClient, withSyncResult bool) string {
	abiJson, err := ioutil.ReadFile(crossVmABI)
	if err != nil {
		log.Fatalln(err)
	}

	myAbi, err := abi.JSON(strings.NewReader(string(abiJson)))
	if err != nil {
		log.Fatalln(err)
	}

	addr := evmutils.MakeAddress([]byte(claimName))
	claim := evmutils.BigToAddress(addr)
	crossMethod := "save"
	fileHash := uuid.GetUUID()
	curTime := strconv.FormatInt(time.Now().Unix(), 10)
	fileName := fmt.Sprintf("file_%s", curTime)
	method := "cross_call"
	dataByte, err := myAbi.Pack(method, claim, crossMethod, curTime, fileName, fileHash)
	if err != nil {
		log.Fatalln(err)
	}

	dataString := hex.EncodeToString(dataByte)
	kvs := []*common.KeyValuePair{
		{
			Key:   "data", //protocol.ContractEvmParamKey
			Value: []byte(dataString),
		},
	}

	err = invokeUserContract(client, crossVmName, method, "", kvs, withSyncResult)
	if err != nil {
		log.Fatalln(err)
	}

	return fileHash
}

func testCreateClaimContract(client *sdk.ChainClient, withSyncResult bool, usernames ...string) {

	resp, err := createUserContract(client, claimName, claimVersion, claimBinPath,
		common.RuntimeType_WASMER, []*common.KeyValuePair{}, withSyncResult, usernames...)
	if err != nil {
		log.Fatalln(err)
	}

	fmt.Printf("CREATE claim contract resp: %+v\n", resp)
}

func testCrossCall(sdkPath string) {
	fmt.Println("====================== create client ======================")
	client, err := examples.CreateChainClientWithSDKConf(sdkPath)
	if err != nil {
		log.Fatalln(err)
	}

	usernames := []string{examples.UserNameOrg1Admin1, examples.UserNameOrg2Admin1, examples.UserNameOrg3Admin1, examples.UserNameOrg4Admin1}
	fmt.Println("====================== 创建callee(被调用者)合约 ======================")
	testCreateCallee(client, true, true, usernames...)

	fmt.Println("====================== 创建caller(调用者)合约 ======================")
	testCreateCaller(client, true, true, usernames...)

	fmt.Println("====================== caller跨合约调用callee的Adder方法 ======================")
	testCallerCrossCallCalleeAdder(client, true)
}

func testCreateCallee(client *sdk.ChainClient, withSyncResult bool, isIgnoreSameContract bool, usernames ...string) {

	codeBytes, err := ioutil.ReadFile(calleeBin)
	if err != nil {
		log.Fatalln(err)
	}

	resp, err := createUserContract(client, calleeName, contractVersion, string(codeBytes), common.RuntimeType_EVM,
		nil, withSyncResult, usernames...)
	if !isIgnoreSameContract {
		if err != nil {
			log.Fatalln(err)
		}
	}

	fmt.Printf("CREATE EVM callee contract resp: [code:%d]/[msg:%s]\n", resp.Code, resp.Message)
	fmt.Printf("contract result: [code:%d]/[msg:%s]/[contractResult:%+X]\n", resp.ContractResult.Code,
		resp.ContractResult.Message, resp.ContractResult.Result)
}

func testCreateCaller(client *sdk.ChainClient, withSyncResult bool, isIgnoreSameContract bool, usernames ...string) {

	codeBytes, err := ioutil.ReadFile(callerBin)
	if err != nil {
		log.Fatalln(err)
	}

	resp, err := createUserContract(client, callerName, contractVersion, string(codeBytes), common.RuntimeType_EVM,
		nil, withSyncResult, usernames...)
	if !isIgnoreSameContract {
		if err != nil {
			log.Fatalln(err)
		}
	}

	fmt.Printf("CREATE EVM caller contract resp: [code:%d]/[msg:%s]\n", resp.Code, resp.Message)
	fmt.Printf("contract result: [code:%d]/[msg:%s]/[contractResult:%+X]\n", resp.ContractResult.Code,
		resp.ContractResult.Message, resp.ContractResult.Result)
}

func testCallerCrossCallCalleeAdder(client *sdk.ChainClient, withSyncResult bool) {

	abiJson, err := ioutil.ReadFile(callerABI)
	if err != nil {
		log.Fatalln(err)
	}

	myAbi, err := abi.JSON(strings.NewReader(string(abiJson)))
	if err != nil {
		log.Fatalln(err)
	}

	addr := evmutils.MakeAddress([]byte(calleeName))
	callee := evmutils.BigToAddress(addr)
	method := "crossCall"
	dataByte, err := myAbi.Pack(method, callee, big.NewInt(40), big.NewInt(60))
	if err != nil {
		log.Fatalln(err)
	}

	dataString := hex.EncodeToString(dataByte)

	kvs := []*common.KeyValuePair{
		{
			Key:   "data", //protocol.ContractEvmParamKey
			Value: []byte(dataString),
		},
	}

	err = invokeUserContract(client, callerName, method, "", kvs, withSyncResult)
	if err != nil {
		log.Fatalln(err)
	}
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

	resp, err := client.SendContractManageRequest(payload, endorsers, createContractTimeout, withSyncResult)
	if err != nil {
		return nil, err
	}

	err = examples.CheckProposalRequestResp(resp, true)
	if err != nil {
		return nil, err
	}

	return resp, nil
}

func invokeUserContract(client *sdk.ChainClient, contractName, method, txId string, kvs []*common.KeyValuePair, withSyncResult bool) error {

	resp, err := client.InvokeContract(contractName, method, txId, kvs, -1, withSyncResult)
	if err != nil {
		return err
	}

	if resp.Code != common.TxStatusCode_SUCCESS {
		return fmt.Errorf("invoke contract failed, [code:%d]/[msg:%s]\n", resp.Code, resp.Message)
	}

	if !withSyncResult {
		fmt.Printf("invoke contract success, resp: [code:%d]/[msg:%s]/[txId:%s]\n", resp.Code, resp.Message,
			resp.ContractResult.Result)
	} else {
		fmt.Printf("invoke contract success, resp: [code:%d]/[msg:%s]\n", resp.Code, resp.Message)
		fmt.Printf("contract result: [code:%d]/[msg:%s]/[contractResult:%+X]\n", resp.ContractResult.Code,
			resp.ContractResult.Message, resp.ContractResult.Result)
	}

	return nil
}
