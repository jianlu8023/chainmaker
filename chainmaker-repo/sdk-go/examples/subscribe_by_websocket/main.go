/*
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/
package main

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"

	"chainmaker.org/chainmaker/common/v2/crypto/tls/config"
	wss "chainmaker.org/chainmaker/common/v2/crypto/tls/wss"

	"github.com/gogo/protobuf/proto"

	"chainmaker.org/chainmaker/pb-go/v2/common"
	sdk "chainmaker.org/chainmaker/sdk-go/v2"
	"github.com/gorilla/websocket"
)

const (
	sdkConfigOrg1Client1Path = "../sdk_configs/sdk_config_org1_client1.yml"
	caCertPath               = "../../testdata/crypto-config/wx-org1.chainmaker.org/ca/ca.crt"
	userTlsCrtPath           = "../../testdata/crypto-config/wx-org1.chainmaker.org/user/client1/client1.tls.crt"
	userTlsKeyPath           = "../../testdata/crypto-config/wx-org1.chainmaker.org/user/client1/client1.tls.key"

	//enableTLS = false
	enableTLS = true

	schemeWS  = "ws"
	schemeWSS = "wss"

	nodeAddr = "localhost:12301"
	path     = "/v1/subscribe"
)

const (
	_ = iota
	subscribeTypeBlock
	subscribeTypeBlockHeader
	subscribeTypeTx
	subscribeTypeEvent
)

var subType = flag.Int("subscribeType", 1, "1-block; 2-blockHeader; 3-tx; 4-event")

type StreamError struct {
	GrpcCode   int32  `json:"grpc_code,omitempty"`
	HttpCode   int32  `json:"http_code,omitempty"`
	Message    string `json:"message,omitempty"`
	HttpStatus string `json:"http_status,omitempty"`
}

type WSResp struct {
	Error  StreamError            `json:"error,omitempty"`
	Result common.SubscribeResult `json:"result"`
}

func main() {
	flag.Parse()

	client, err := sdk.NewChainClient(sdk.WithConfPath(sdkConfigOrg1Client1Path))
	if err != nil {
		log.Fatal(err)
	}

	if *subType == subscribeTypeBlock {
		testSubscribeBlock(client, 0, -1, false, false)
	} else if *subType == subscribeTypeBlockHeader {
		testSubscribeBlock(client, 0, -1, false, true)
	} else if *subType == subscribeTypeTx {
		testSubscribeTx(client, 0, -1, "", nil)
	} else if *subType == subscribeTypeEvent {
		testSubscribeContractEvent(client, 0, -1, "claim_restful_001", "")
	}
}

func receiveHandler(connection *websocket.Conn, done chan struct{}) {
	defer close(done)

	for {
		_, data, err := connection.ReadMessage()
		if err != nil {
			log.Println("read data from conn failed, ", err)
			return
		}

		//log.Printf("received data: %s\n", data)

		var result WSResp
		err = json.Unmarshal(data, &result)
		if err != nil {
			log.Println("json unmarshal failed, ", err)
			return
		}

		//log.Printf("received result: %+v\n", result)

		if result.Error.HttpCode != http.StatusOK && result.Error.GrpcCode != 0 {
			log.Printf("subscribe by websocket failed, [httpCode:%d]/[httpStatus:%s]/[grpcCode:%d]/[errMsg:%s]\n",
				result.Error.HttpCode, result.Error.HttpStatus, result.Error.GrpcCode, result.Error.Message)
			return
		}

		if *subType == subscribeTypeBlock {
			blockInfo := &common.BlockInfo{}
			if err = proto.Unmarshal(result.Result.Data, blockInfo); err != nil {
				log.Println("unmarshal data failed:", err)
				return
			}
			log.Printf(">>> blockInfo: %+v\n", blockInfo)
		} else if *subType == subscribeTypeBlockHeader {
			blockHeader := &common.BlockHeader{}
			if err = proto.Unmarshal(result.Result.Data, blockHeader); err != nil {
				log.Println("unmarshal data failed:", err)
				return
			}
			log.Printf(">>> blockHeader: %+v\n", blockHeader)

		} else if *subType == subscribeTypeTx {
			tx := &common.Transaction{}
			if err = proto.Unmarshal(result.Result.Data, tx); err != nil {
				log.Println("unmarshal data failed:", err)
				return
			}
			log.Printf(">>> tx: %+v\n", tx)
		} else if *subType == subscribeTypeEvent {
			events := &common.ContractEventInfoList{}
			if err = proto.Unmarshal(result.Result.Data, events); err != nil {
				log.Println("unmarshal data failed:", err)
				return
			}
			log.Printf(">>> enents: %+v\n", events)
		}

		fmt.Printf("\n\n")

	}
}

func testSubscribeBlock(client *sdk.ChainClient, startBlock, endBlock int64, withRWSet,
	onlyHeader bool) {

	payload := client.CreateSubscribeBlockPayload(startBlock, endBlock, withRWSet, onlyHeader)

	subscribe(client, payload)
}

func testSubscribeTx(client *sdk.ChainClient, startBlock, endBlock int64, contractName string,
	txIds []string) {

	payload := client.CreateSubscribeTxPayload(startBlock, endBlock, contractName, txIds)

	subscribe(client, payload)
}

func testSubscribeContractEvent(client *sdk.ChainClient, startBlock, endBlock int64, contractName, topic string) {

	payload := client.CreateSubscribeContractEventPayload(startBlock, endBlock, contractName, topic)

	subscribe(client, payload)
}

func subscribe(client *sdk.ChainClient, payload *common.Payload) {
	var (
		scheme string
		//tlsConfig *tls.Config
		dial *websocket.Dialer
	)

	req, err := client.GenerateTxRequest(payload, nil)
	if err != nil {
		log.Fatalln(err)
	}

	rawTxReq, err := req.Marshal()
	if err != nil {
		log.Fatalln(err)
	}

	params := url.Values{}
	params.Add("rawTx", base64.StdEncoding.EncodeToString(rawTxReq))

	scheme = schemeWS
	if enableTLS {
		scheme = schemeWSS
		//tlsConfig = createTLSConfig()
		//dial = &websocket.Dialer{TLSClientConfig: tlsConfig}

		cfg, _ := config.GetConfig(userTlsCrtPath, userTlsKeyPath, caCertPath, false)
		dial = wss.NewDial(cfg)

	} else {
		dial = websocket.DefaultDialer
	}

	u := url.URL{
		Scheme:   scheme,
		Host:     nodeAddr,
		Path:     path,
		RawQuery: params.Encode(),
	}

	//fmt.Printf("url: %s\n", u.String())

	conn, _, err := dial.Dial(u.String(), nil)
	if err != nil {
		log.Fatal("Error connecting to Websocket Server:", err)
	}
	defer conn.Close()

	done := make(chan struct{})

	go receiveHandler(conn, done)

	select {
	case <-done:
		log.Printf("subscriber is finished!")
		return
	}
}

func createTLSConfig() *tls.Config {
	pool := x509.NewCertPool()

	caCrt, err := ioutil.ReadFile(caCertPath)
	if err != nil {
		log.Fatal("ReadFile err:", err)
	}
	pool.AppendCertsFromPEM(caCrt)

	cliCrt, err := tls.LoadX509KeyPair(userTlsCrtPath, userTlsKeyPath)
	if err != nil {
		log.Fatal("LoadX509KeyPair err:", err)
	}

	return &tls.Config{
		//ServerName:   "chainmaker.org",
		RootCAs:      pool,
		Certificates: []tls.Certificate{cliCrt},
	}
}
