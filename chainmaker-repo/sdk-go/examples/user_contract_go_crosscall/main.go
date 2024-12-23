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

	"strings"

	"chainmaker.org/chainmaker/common/v2/evmutils/abi"
	"chainmaker.org/chainmaker/pb-go/v2/common"
	sdk "chainmaker.org/chainmaker/sdk-go/v2"
	"chainmaker.org/chainmaker/sdk-go/v2/examples"
)

const (
	createContractTimeout = 5

	goCallerName    = "goCaller"
	goCallerVersion = "1.0.0"
	goCallerBin     = "../../testdata/cross-call-go-demo/go_call_evm.wasm"

	evmCalleeName            = "storage"
	evmCalleeVersion         = "2.0.0"
	evmCalleeBinPath         = "../../testdata/storage-evm-demo/storage.bin"
	evmCalleeABI             = "../../testdata/storage-evm-demo/storage.abi"
	sdkConfigOrg1Client1Path = "../sdk_configs/sdk_config_org1_client1.yml"
)

func main() {
	testGoCallEvmContract(sdkConfigOrg1Client1Path)
}

func testGoCallEvmContract(sdkPath string) {
	fmt.Println("====================== create client ======================")
	client, err := examples.CreateChainClientWithSDKConf(sdkPath)
	if err != nil {
		log.Fatalln(err)
	}

	usernames := []string{examples.UserNameOrg1Admin1, examples.UserNameOrg2Admin1, examples.UserNameOrg3Admin1, examples.UserNameOrg4Admin1}
	fmt.Println("====================== 创建调用者合约caller(go版本) ======================")
	testCreateGoCaller(client, true, usernames...)

	fmt.Println("====================== 创建被调用者合约storage(solidity版本) ======================")
	testCreateStorage(client, true, true, usernames...)

	fmt.Println("====================== caller跨合约调用set方法 ======================")
	testGoCallEvmContractSetMethod(client, true)

	fmt.Println("====================== 查询跨合约调用evm合约set的结果 ======================")
	testUserContractStorageEVMGet(client, true)
}

func testUserContractStorageEVMGet(client *sdk.ChainClient, withSyncResult bool) {

	abiJson, err := ioutil.ReadFile(evmCalleeABI)
	if err != nil {
		log.Fatalln(err)
	}

	myAbi, err := abi.JSON(strings.NewReader(string(abiJson)))
	if err != nil {
		log.Fatalln(err)
	}

	method := "get"
	dataByte, err := myAbi.Pack(method)
	if err != nil {
		log.Fatalln(err)
	}

	dataString := hex.EncodeToString(dataByte)

	kvs := []*common.KeyValuePair{
		{
			Key:   "data",
			Value: []byte(dataString),
		},
	}

	result, err := invokeUserContractWithResult(client, evmCalleeName, method, "", kvs, withSyncResult)
	if err != nil {
		log.Fatalln(err)
	}

	val, err := myAbi.Unpack("get", result)
	if err != nil {
		log.Fatalln(err)
	}

	fmt.Printf("val: %v\n", val)
}

func testCreateStorage(client *sdk.ChainClient, withSyncResult bool, isIgnoreSameContract bool, usernames ...string) {

	codeBytes, err := ioutil.ReadFile(evmCalleeBinPath)
	if err != nil {
		log.Fatalln(err)
	}

	resp, err := createUserContract(client, evmCalleeName, evmCalleeVersion, string(codeBytes), common.RuntimeType_EVM,
		nil, withSyncResult, usernames...)
	if !isIgnoreSameContract {
		if err != nil {
			log.Fatalln(err)
		}
	}

	fmt.Printf("CREATE CALLEE EVM contract resp: [code:%d]/[msg:%s]\n", resp.Code, resp.Message)
	fmt.Printf("contract result: [code:%d]/[msg:%s]/[contractResult:%+X]\n", resp.ContractResult.Code,
		resp.ContractResult.Message, resp.ContractResult.Result)
}

func testGoCallEvmContractSetMethod(client *sdk.ChainClient, withSyncResult bool) bool {
	abiJson, err := ioutil.ReadFile(evmCalleeABI)
	if err != nil {
		log.Fatalln(err)
	}

	myAbi, err := abi.JSON(strings.NewReader(string(abiJson)))
	if err != nil {
		log.Fatalln(err)
	}

	method := "set"
	dataByte, err := myAbi.Pack(method, 100)
	if err != nil {
		log.Fatalln(err)
	}

	dataString := hex.EncodeToString(dataByte)
	kvs := []*common.KeyValuePair{
		{
			Key:   "name",
			Value: []byte(evmCalleeName),
		},
		{
			Key:   "method",
			Value: []byte(method),
		},
		{
			Key:   "calldata",
			Value: []byte(dataString),
		},
	}

	err = invokeUserContract(client, goCallerName, "crossCallEvmContract", "", kvs, withSyncResult)
	if err != nil {
		log.Fatalln(err)
		return false
	}

	return true
}

func testCreateGoCaller(client *sdk.ChainClient, withSyncResult bool, usernames ...string) {

	resp, err := createUserContract(client, goCallerName, goCallerVersion, goCallerBin,
		common.RuntimeType_GASM, []*common.KeyValuePair{}, withSyncResult, usernames...)
	if err != nil {
		log.Fatalln(err)
	}

	fmt.Printf("CREATE CALLER go contract resp: %+v\n", resp)
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
func invokeUserContractWithResult(client *sdk.ChainClient, contractName, method, txId string,
	kvs []*common.KeyValuePair, withSyncResult bool) ([]byte, error) {

	resp, err := client.InvokeContract(contractName, method, txId, kvs, -1, withSyncResult)
	if err != nil {
		return nil, err
	}

	if resp.Code != common.TxStatusCode_SUCCESS {
		return nil, fmt.Errorf("invoke contract failed, [code:%d]/[msg:%s]\n", resp.Code, resp.Message)
	}

	return resp.ContractResult.Result, nil
}
