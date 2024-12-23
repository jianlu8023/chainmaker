/*
Copyright (C) BABEC. All rights reserved.
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package main

import (
	"fmt"
	"log"
	"time"

	"chainmaker.org/chainmaker/common/v2/crypto"
	"chainmaker.org/chainmaker/pb-go/v2/common"
	"chainmaker.org/chainmaker/pb-go/v2/syscontract"
	sdk "chainmaker.org/chainmaker/sdk-go/v2"
	"chainmaker.org/chainmaker/sdk-go/v2/examples"
)

const (
	sdkConfigPKUser1Path = "../sdk_configs/sdk_config_pk_user1.yml"
	amount               = 900000000
)

func main() {
	client, err := examples.CreateChainClientWithSDKConf(sdkConfigPKUser1Path)
	if err != nil {
		log.Fatal(err)
	}
	pk, err := client.GetPublicKey().String()
	if err != nil {
		log.Fatal(err)
	}

	addr, err := sdk.GetZXAddressFromPKPEM(pk, crypto.HASH_TYPE_SHA256)
	fmt.Printf("addr: %s\n", addr)

	fmt.Println("====================== 设置client自己为 gas admin ======================")
	if err := setGasAdmin(client, addr); err != nil {
		log.Fatal(err)
	}

	fmt.Println("====================== 获取 gas admin ======================")
	if err := getGasAdmin(client); err != nil {
		log.Fatal(err)
	}

	fmt.Printf("====================== 充值gas账户%d个gas ======================\n", amount)
	rechargeGasList := []*syscontract.RechargeGas{
		{
			Address:   addr,
			GasAmount: amount,
		},
	}
	if err := rechargeGas(client, rechargeGasList); err != nil {
		log.Fatal(err)
	}

	fmt.Println("====================== 查询gas账户余额 ======================")
	if err := getGasBalance(client, addr); err != nil {
		log.Fatal(err)
	}

	fmt.Println("====================== 退还gas账户5个gas ======================")
	if err := refundGas(client, addr, 5); err != nil {
		log.Fatal(err)
	}

	fmt.Println("====================== 查询gas账户余额 ======================")
	if err := getGasBalance(client, addr); err != nil {
		log.Fatal(err)
	}

	fmt.Println("====================== 冻结指定gas账户 ======================")
	if err := frozenGasAccount(client, addr); err != nil {
		log.Fatal(err)
	}

	fmt.Println("====================== 查询gas账户的状态 ======================")
	if err := getGasAccountStatus(client, addr); err != nil {
		log.Fatal(err)
	}

	fmt.Println("====================== 解冻指定gas账户 ======================")
	if err := unfrozenGasAccount(client, addr); err != nil {
		log.Fatal(err)
	}

	fmt.Println("====================== 查询gas账户的状态 ======================")
	if err := getGasAccountStatus(client, addr); err != nil {
		log.Fatal(err)
	}

	fmt.Println("====================== 估算交易的gas消耗量 ======================")
	if err := estimateGas(client); err != nil {
		log.Fatal(err)
	}

	fmt.Println("====================== 配置账户基础gas消耗数量 ======================")
	if err := setInvokeBaseGas(client); err != nil {
		log.Fatal(err)
	}
}

func setGasAdmin(cc *sdk.ChainClient, address string) error {
	payload, err := cc.CreateSetGasAdminPayload(address)
	if err != nil {
		return err
	}
	endorsers, err := examples.GetEndorsersWithAuthType(cc.GetHashType(),
		cc.GetAuthType(), payload, examples.UserNameOrg1Admin1, examples.UserNameOrg2Admin1,
		examples.UserNameOrg3Admin1, examples.UserNameOrg4Admin1)
	if err != nil {
		return err
	}
	resp, err := cc.SendGasManageRequest(payload, endorsers, -1, true)
	if err != nil {
		return err
	}

	fmt.Printf("set gas admin resp: %+v\n", resp)
	return nil
}

func getGasAdmin(cc *sdk.ChainClient) error {
	adminAddr, err := cc.GetGasAdmin()
	if err != nil {
		return err
	}
	fmt.Printf("get gas admin address: %+v\n", adminAddr)
	return nil
}

func rechargeGas(cc *sdk.ChainClient, rechargeGasList []*syscontract.RechargeGas) error {
	payload, err := cc.CreateRechargeGasPayload(rechargeGasList)
	if err != nil {
		return err
	}
	resp, err := cc.SendGasManageRequest(payload, nil, -1, true)
	if err != nil {
		return err
	}

	fmt.Printf("recharge gas resp: %+v\n", resp)
	return nil
}

func getGasBalance(cc *sdk.ChainClient, address string) error {
	balance, err := cc.GetGasBalance(address)
	if err != nil {
		return err
	}

	fmt.Printf("get gas balance: %+v\n", balance)
	return nil
}

func refundGas(cc *sdk.ChainClient, address string, amount int64) error {
	payload, err := cc.CreateRefundGasPayload(address, amount)
	if err != nil {
		return err
	}
	resp, err := cc.SendGasManageRequest(payload, nil, -1, true)
	if err != nil {
		return err
	}

	fmt.Printf("refund gas resp: %+v\n", resp)
	return nil
}

func frozenGasAccount(cc *sdk.ChainClient, address string) error {
	payload, err := cc.CreateFrozenGasAccountPayload(address)
	if err != nil {
		return err
	}
	resp, err := cc.SendGasManageRequest(payload, nil, -1, true)
	if err != nil {
		return err
	}

	fmt.Printf("frozen gas account resp: %+v\n", resp)
	return nil
}

func unfrozenGasAccount(cc *sdk.ChainClient, address string) error {
	payload, err := cc.CreateUnfrozenGasAccountPayload(address)
	if err != nil {
		return err
	}
	resp, err := cc.SendGasManageRequest(payload, nil, -1, true)
	if err != nil {
		return err
	}

	fmt.Printf("unfrozen gas account resp: %+v\n", resp)
	return nil
}

func getGasAccountStatus(cc *sdk.ChainClient, address string) error {
	status, err := cc.GetGasAccountStatus(address)
	if err != nil {
		return err
	}

	fmt.Printf("get gas account status: %+v\n", status)
	return nil
}

func estimateGas(cc *sdk.ChainClient) error {
	var (
		claimContractName = fmt.Sprintf("claim%d", time.Now().UnixNano())
		claimVersion      = "2.0.0"
		claimByteCodePath = "../../testdata/claim-wasm-demo/rust-fact-2.0.0.wasm"
	)
	payload, err := cc.CreateContractCreatePayload(claimContractName, claimVersion, claimByteCodePath,
		common.RuntimeType_WASMER, []*common.KeyValuePair{})
	if err != nil {
		return err
	}

	gas, err := cc.EstimateGas(payload)
	if err != nil {
		return err
	}
	fmt.Printf("estimate gas: %+v\n", gas)
	return nil
}

func setInvokeBaseGas(cc *sdk.ChainClient) error {
	payload, err := cc.CreateSetInvokeBaseGasPayload(123)
	if err != nil {
		return err
	}

	endorsers, err := examples.GetEndorsersWithAuthType(cc.GetHashType(),
		cc.GetAuthType(), payload, examples.UserNameOrg1Admin1, examples.UserNameOrg2Admin1,
		examples.UserNameOrg3Admin1, examples.UserNameOrg4Admin1)
	if err != nil {
		return err
	}
	resp, err := cc.SendGasManageRequest(payload, endorsers, -1, true)
	if err != nil {
		return err
	}

	fmt.Printf("set invoke base gas resp: %+v\n", resp)

	chainConfig, err := cc.GetChainConfig()
	if err != nil {
		return err
	}
	examples.PrintPrettyJson(chainConfig)
	return nil
}
