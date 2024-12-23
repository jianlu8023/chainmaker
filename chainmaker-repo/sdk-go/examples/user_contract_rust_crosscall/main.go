package main

import (
	"chainmaker.org/chainmaker/pb-go/v2/common"
	sdk "chainmaker.org/chainmaker/sdk-go/v2"
	"chainmaker.org/chainmaker/sdk-go/v2/examples"
	"encoding/base64"
	"fmt"
	"io/fs"
	"io/ioutil"
	"log"
)

const (
	createContractTimeout = 5

	calleeName    = "callee018"
	calleeVersion = "2.0.0"
	calleeBinPath = "../../testdata/cross-call-rust-demo/Callee.bin"

	callerName    = "caller018"
	callerVersion = "2.0.0"
	callerBinPath = "../../testdata/cross-call-rust-demo/Caller.bin"

	sdkConfigOrg1Client1Path = "../sdk_configs/sdk_config_org1_client1.yml"
)

func ConvertWasm2Base64() (string, error) {
	wasmData, err := ioutil.ReadFile("../../testdata/cross-call-rust-demo/Caller.wasm")
	if err != nil {
		return "", err
	}

	base64Data := base64.StdEncoding.EncodeToString(wasmData)
	err = ioutil.WriteFile(callerBinPath, []byte(base64Data), fs.ModeAppend)
	if err != nil {
		return "", err
	}

	return callerBinPath, nil
}

func main() {
	//binFile, err := ConvertWasm2Base64()
	//if err != nil {
	//	fmt.Printf("write file failed: %v \n", err)
	//	return
	//}
	//fmt.Printf("write file success: %v \n", binFile)

	testCrossVmCall(sdkConfigOrg1Client1Path)
}

func testCrossVmCall(sdkPath string) {
	fmt.Println("====================== create client ======================")
	client, err := examples.CreateChainClientWithSDKConf(sdkPath)
	if err != nil {
		log.Fatalln(err)
	}

	usernames := []string{examples.UserNameOrg1Admin1, examples.UserNameOrg2Admin1, examples.UserNameOrg3Admin1, examples.UserNameOrg4Admin1}
	fmt.Println("====================== 创建被调用者合约(evm 版本) ======================")
	testCreateCalleeContract(client, true, usernames...)

	fmt.Println("====================== 创建调用者合约(rust 版本) ======================")
	testCreateCallerContract(client, true, true, usernames...)

	fmt.Println("====================== caller(rust) 跨合约调用 evm 合约的 Adder 方法 ======================")
	params := []*common.KeyValuePair{
		&common.KeyValuePair{
			Key:   "contract",
			Value: []byte(calleeName),
		},
		&common.KeyValuePair{
			Key:   "method",
			Value: []byte("Adder"),
		},
	}
	testRustCallEvm(client, callerName, "test", params, true)
}

func testCreateCalleeContract(client *sdk.ChainClient, withSyncResult bool, usernames ...string) {
	resp, err := createUserContract(client, calleeName, calleeVersion, calleeBinPath,
		common.RuntimeType_EVM, []*common.KeyValuePair{}, withSyncResult, usernames...)
	if err != nil {
		log.Fatalln(err)
	}

	fmt.Printf("create `callee` contract resp: %v \n", resp)
}

func testCreateCallerContract(client *sdk.ChainClient, withSyncResult bool,
	isIgnoreSameContract bool, usernames ...string) {

	codeBytes, err := ioutil.ReadFile(callerBinPath)
	if err != nil {
		log.Fatalln(err)
	}

	resp, err := createUserContract(client, callerName, callerVersion, string(codeBytes), common.RuntimeType_WASMER,
		nil, withSyncResult, usernames...)
	if !isIgnoreSameContract {
		if err != nil {
			log.Fatalln(err)
		}
	}

	fmt.Printf("create `caller` contract resp: %v \n", resp)
}

func testRustCallEvm(client *sdk.ChainClient, contractName string,
	method string, params []*common.KeyValuePair, withSyncResult bool) error {

	resp, err := client.InvokeContract(contractName, method, "", params, -1, withSyncResult)
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

func createUserContract(client *sdk.ChainClient,
	contractName, version, byteCodePath string,
	runtimeType common.RuntimeType,
	kvs []*common.KeyValuePair,
	withSyncResult bool,
	usernames ...string) (*common.TxResponse, error) {

	payload, err := client.CreateContractCreatePayload(contractName, version, byteCodePath, runtimeType, kvs)
	if err != nil {
		return nil, err
	}

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
