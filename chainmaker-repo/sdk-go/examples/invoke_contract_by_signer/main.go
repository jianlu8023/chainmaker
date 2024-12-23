/*
Copyright (C) BABEC. All rights reserved.
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"strconv"
	"time"

	"chainmaker.org/chainmaker/common/v2/crypto/asym"
	"chainmaker.org/chainmaker/common/v2/random/uuid"
	"chainmaker.org/chainmaker/pb-go/v2/common"
	sdk "chainmaker.org/chainmaker/sdk-go/v2"
	"chainmaker.org/chainmaker/sdk-go/v2/examples"
	sdkutils "chainmaker.org/chainmaker/sdk-go/v2/utils"
)

var (
	claimContractName = fmt.Sprintf("claim%d", time.Now().UnixNano())
	claimVersion      = "v2.0.0"
	claimByteCodePath = "../../testdata/claim-wasm-demo/rust-fact-2.0.0.wasm"

	sdkConfigOrg1Client1Path = "../sdk_configs/sdk_config_org1_client1.yml"

	secondSignerCertFilePath    = "../../testdata/crypto-config/wx-org2.chainmaker.org/user/client1/client1.sign.crt"
	secondSignerPrivKeyFilePath = "../../testdata/crypto-config/wx-org2.chainmaker.org/user/client1/client1.sign.key"
	secondSignerOrgId           = "wx-org2.chainmaker.org"
)

func main() {
	cc, err := sdk.NewChainClient(
		sdk.WithConfPath(sdkConfigOrg1Client1Path),
	)
	if err != nil {
		log.Fatalln(err)
	}

	fmt.Println("====================== 创建合约 ======================")
	usernames := []string{examples.UserNameOrg1Admin1, examples.UserNameOrg2Admin1, examples.UserNameOrg3Admin1, examples.UserNameOrg4Admin1}
	testCreateUserContract(cc, claimContractName, claimVersion, claimByteCodePath,
		common.RuntimeType_WASMER, []*common.KeyValuePair{}, true, usernames...)

	fmt.Println("====================== 使用其他signer调用合约 ======================")
	certPem, err := ioutil.ReadFile(secondSignerCertFilePath)
	if err != nil {
		log.Fatalln(err)
	}
	cert, err := sdkutils.ParseCert(certPem)
	if err != nil {
		log.Fatalln(err)
	}
	privKeyPem, err := ioutil.ReadFile(secondSignerPrivKeyFilePath)
	if err != nil {
		log.Fatalln(err)
	}
	privateKey, err := asym.PrivateKeyFromPEM(privKeyPem, nil)
	if err != nil {
		log.Fatalln(err)
	}

	signer := &sdk.CertModeSigner{
		PrivateKey: privateKey,
		Cert:       cert,
		OrgId:      secondSignerOrgId,
	}
	_, err = testInvokeContractBySigner(cc, "save", true, signer)
	if err != nil {
		log.Println(err)
	}
}

func testCreateUserContract(client *sdk.ChainClient, contractName, version, byteCodePath string, runtime common.RuntimeType,
	kvs []*common.KeyValuePair, withSyncResult bool, usernames ...string) {

	payload, err := client.CreateContractCreatePayload(contractName, version, byteCodePath, runtime, kvs)
	if err != nil {
		log.Fatalln(err)
	}

	//endorsers, err := examples.GetEndorsers(payload, usernames...)
	endorsers, err := examples.GetEndorsersWithAuthType(client.GetHashType(),
		client.GetAuthType(), payload, usernames...)
	if err != nil {
		log.Fatalln(err)
	}

	resp, err := client.SendContractManageRequest(payload, endorsers, -1, withSyncResult)
	if err != nil {
		log.Fatalln(err)
	}

	err = examples.CheckProposalRequestResp(resp, true)
	if err != nil {
		log.Fatalln(err)
	}

	fmt.Printf("CREATE claim contract resp: %+v\n", resp)
}

func testInvokeContractBySigner(client *sdk.ChainClient,
	method string, withSyncResult bool, signer sdk.Signer) (string, error) {

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

	resp, err := client.InvokeContractBySigner(claimContractName, method, "", kvs, -1,
		withSyncResult, nil, signer)
	if err != nil {
		return "", err
	}

	if resp.Code != common.TxStatusCode_SUCCESS {
		return "", fmt.Errorf("invoke contract failed, [code:%d]/[msg:%s]\n", resp.Code, resp.Message)
	}

	if !withSyncResult {
		fmt.Printf("invoke contract success, resp: [code:%d]/[msg:%s]/[txId:%s]\n", resp.Code, resp.Message, resp.ContractResult.Result)
	} else {
		fmt.Printf("invoke contract success, resp: [code:%d]/[msg:%s]/[contractResult:%s]\n", resp.Code, resp.Message, resp.ContractResult)
	}
	return fileHash, nil
}
