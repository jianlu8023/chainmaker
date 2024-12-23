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

	"chainmaker.org/chainmaker/common/v2/evmutils/abi"
	"chainmaker.org/chainmaker/common/v2/random/uuid"
	"chainmaker.org/chainmaker/pb-go/v2/common"
	sdk "chainmaker.org/chainmaker/sdk-go/v2"
	"chainmaker.org/chainmaker/sdk-go/v2/examples"
)

const (
	createContractTimeout = 5
	contractCName         = "CreatorC"
	contractDName         = "beCreated"
	storageVersion        = "1.0.0"
	cBinPath              = "../../testdata/inner-create-evm-demo/C.bin"
	cABIPath              = "../../testdata/inner-create-evm-demo/C.abi"
	dABIPath              = "../../testdata/inner-create-evm-demo/D.abi"

	factoryName    = "contractFactory"
	factoryBinPath = "../../testdata/inner-create-evm-demo/Factory.bin"
	factoryABIPath = "../../testdata/inner-create-evm-demo/Factory.abi"
	storeName      = "contractStorage"
	storageBinPath = "../../testdata/storage-evm-demo/storage.bin"
	storageABIPath = "../../testdata/storage-evm-demo/storage.abi"

	claimName    = "claim001"
	claimBinPath = "../../testdata/claim-wasm-demo/rust-fact-2.0.0.wasm"

	sdkConfigOrg1Client1Path = "../sdk_configs/sdk_config_org1_client1.yml"
)

func main() {
	//testStaticCreate(sdkConfigOrg1Client1Path)
	testDynamicCreate(sdkConfigOrg1Client1Path)
	//testCrossVmCreate(sdkConfigOrg1Client1Path)
}

func testCrossVmCreate(sdkPath string) {
	fmt.Println("====================== create client ======================")
	client, err := examples.CreateChainClientWithSDKConf(sdkPath)
	if err != nil {
		log.Fatalln(err)
	}

	//注意，如果testDynamicCreate也执行的话，要注释掉以下三行代码，因为Factory合约已经被创建了，单独测试TestCrossVmCreate可以放开注释
	fmt.Println("====================== 创建Factory合约 ======================")
	usernames := []string{examples.UserNameOrg1Admin1, examples.UserNameOrg2Admin1, examples.UserNameOrg3Admin1,
		examples.UserNameOrg4Admin1}
	testCreateFactory(client, true, true, usernames...)

	fmt.Println("====================== 调用Factory合约的create方法创建claim（rust）合约 ======================")
	testFactoryCreateContractClaim(client, true)

	fmt.Println("====================== 调用claim（rust）合约 ======================")
	fileHash, err := testUserContractClaimInvoke(client, "save", true)
	if err != nil {
		log.Fatalln(err)
	}

	fmt.Println("====================== 查询claim(rust)合约 ======================")
	kvs := []*common.KeyValuePair{
		{
			Key:   "file_hash",
			Value: []byte(fileHash),
		},
	}
	testUserContractClaimQuery(client, "find_by_file_hash", kvs)
}

func testFactoryCreateContractClaim(client *sdk.ChainClient, withSyncResult bool) {

	abiJson, err := ioutil.ReadFile(factoryABIPath)
	if err != nil {
		log.Fatalln(err)
	}

	myAbi, err := abi.JSON(strings.NewReader(string(abiJson)))
	if err != nil {
		log.Fatalln(err)
	}

	//hexCode, err := ioutil.ReadFile(claimBinPath)
	code, err := ioutil.ReadFile(claimBinPath) //wasm文件不需要decode
	if err != nil {
		log.Fatalln(err)
	}
	//code, _ := hex.DecodeString(string(hexCode))

	method := "create"
	dataByte, err := myAbi.Pack(method, uint8(common.RuntimeType_WASMER), claimName, code)
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

	err = invokeUserContract(client, factoryName, method, "", kvs, withSyncResult)
	if err != nil {
		log.Fatalln(err)
	}
}

func testUserContractClaimInvoke(client *sdk.ChainClient,
	method string, withSyncResult bool) (string, error) {

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

	err := invokeUserContract(client, claimName, method, "", kvs, withSyncResult)
	if err != nil {
		return "", err
	}

	return fileHash, nil
}

func testUserContractClaimQuery(client *sdk.ChainClient, method string, kvs []*common.KeyValuePair) {
	resp, err := client.QueryContract(claimName, method, kvs, -1)
	if err != nil {
		log.Fatalln(err)
	}

	fmt.Printf("QUERY claim contract resp: %+v\n", resp)
}

func testDynamicCreate(sdkPath string) {
	fmt.Println("====================== create client ======================")
	client, err := examples.CreateChainClientWithSDKConf(sdkPath)
	if err != nil {
		log.Fatalln(err)
	}

	fmt.Println("====================== 创建Factory合约 ======================")
	usernames := []string{examples.UserNameOrg1Admin1, examples.UserNameOrg2Admin1, examples.UserNameOrg3Admin1,
		examples.UserNameOrg4Admin1}
	testCreateFactory(client, true, true, usernames...)

	fmt.Println("====================== 调用Factory合约的create方法创建store合约 ======================")
	testFactoryCreateContractStore(client, true)

	fmt.Println("====================== 调用(被Factory合约创建的)store合约的set方法 ======================")
	testStoreContractSet(client, true)

	fmt.Println("====================== 调用(被Factory合约创建的)store合约的get方法 ======================")
	testStoreContractGet(client, true)
}

func testCreateFactory(client *sdk.ChainClient, withSyncResult bool, isIgnoreSameContract bool, usernames ...string) {

	codeBytes, err := ioutil.ReadFile(factoryBinPath)
	if err != nil {
		log.Fatalln(err)
	}

	resp, err := createUserContract(client, factoryName, storageVersion, string(codeBytes), common.RuntimeType_EVM,
		nil, withSyncResult, usernames...)
	if !isIgnoreSameContract {
		if err != nil {
			log.Fatalln(err)
		}
	}

	fmt.Printf("CREATE EVM factory contract resp: [code:%d]/[msg:%s]\n", resp.Code, resp.Message)
	fmt.Printf("contract result: [code:%d]/[msg:%s]/[contractResult:%+X]\n", resp.ContractResult.Code,
		resp.ContractResult.Message, resp.ContractResult.Result)
}

func testFactoryCreateContractStore(client *sdk.ChainClient, withSyncResult bool) {

	abiJson, err := ioutil.ReadFile(factoryABIPath)
	if err != nil {
		log.Fatalln(err)
	}

	myAbi, err := abi.JSON(strings.NewReader(string(abiJson)))
	if err != nil {
		log.Fatalln(err)
	}

	hexCode, err := ioutil.ReadFile(storageBinPath)
	if err != nil {
		log.Fatalln(err)
	}
	code, _ := hex.DecodeString(string(hexCode))

	method := "create"
	dataByte, err := myAbi.Pack(method, int8(common.RuntimeType_EVM), storeName, code)
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

	err = invokeUserContract(client, factoryName, method, "", kvs, withSyncResult)
	if err != nil {
		log.Fatalln(err)
	}
}

func testStoreContractSet(client *sdk.ChainClient, withSyncResult bool) {

	abiJson, err := ioutil.ReadFile(storageABIPath)
	if err != nil {
		log.Fatalln(err)
	}

	myAbi, err := abi.JSON(strings.NewReader(string(abiJson)))
	if err != nil {
		log.Fatalln(err)
	}

	method := "set"
	dataByte, err := myAbi.Pack(method, big.NewInt(100))
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

	err = invokeUserContract(client, storeName, method, "", kvs, withSyncResult)
	if err != nil {
		log.Fatalln(err)
	}
}

func testStoreContractGet(client *sdk.ChainClient, withSyncResult bool) {

	abiJson, err := ioutil.ReadFile(storageABIPath)
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

	err = invokeUserContract(client, storeName, method, "", kvs, withSyncResult)
	if err != nil {
		log.Fatalln(err)
	}
}

func testStaticCreate(sdkPath string) {
	fmt.Println("====================== create client ======================")
	client, err := examples.CreateChainClientWithSDKConf(sdkPath)
	if err != nil {
		log.Fatalln(err)
	}

	fmt.Println("====================== 创建Creator C合约 ======================")
	usernames := []string{examples.UserNameOrg1Admin1, examples.UserNameOrg2Admin1, examples.UserNameOrg3Admin1,
		examples.UserNameOrg4Admin1}
	testCCreate(client, true, true, usernames...)

	fmt.Println("====================== 调用创建者C合约 ======================")
	testInvokeC(client, true)

	fmt.Println("====================== 调用被创建者D合约 ======================")
	testInvokeD(client, true)
}

func testCCreate(client *sdk.ChainClient, withSyncResult bool, isIgnoreSameContract bool, usernames ...string) {

	codeBytes, err := ioutil.ReadFile(cBinPath)
	if err != nil {
		log.Fatalln(err)
	}

	resp, err := createUserContract(client, contractCName, storageVersion, string(codeBytes), common.RuntimeType_EVM,
		nil, withSyncResult, usernames...)
	if !isIgnoreSameContract {
		if err != nil {
			log.Fatalln(err)
		}
	}

	fmt.Printf("CREATE EVM factory contract resp: %+v\n", resp)
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

func testInvokeC(client *sdk.ChainClient, withSyncResult bool) {

	abiJson, err := ioutil.ReadFile(cABIPath)
	if err != nil {
		log.Fatalln(err)
	}

	myAbi, err := abi.JSON(strings.NewReader(string(abiJson)))
	if err != nil {
		log.Fatalln(err)
	}

	method := "createDSalted"
	dataByte, err := myAbi.Pack(method, big.NewInt(10000), contractDName)
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

	err = invokeUserContract(client, contractCName, method, "", kvs, withSyncResult)
	if err != nil {
		log.Fatalln(err)
	}
}

func testInvokeD(client *sdk.ChainClient, withSyncResult bool) {

	abiJson, err := ioutil.ReadFile(dABIPath)
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

	err = invokeUserContract(client, contractDName, method, "", kvs, withSyncResult)
	if err != nil {
		log.Fatalln(err)
	}
}

func invokeUserContract(client *sdk.ChainClient, contractName, method, txId string, kvs []*common.KeyValuePair,
	withSyncResult bool) error {

	resp, err := client.InvokeContract(contractName, method, txId, kvs, -1, withSyncResult)
	if err != nil {
		return err
	}

	if resp.Code != common.TxStatusCode_SUCCESS {
		return fmt.Errorf("invoke contract failed, [code:%d]/[msg:%s]", resp.Code, resp.Message)
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
