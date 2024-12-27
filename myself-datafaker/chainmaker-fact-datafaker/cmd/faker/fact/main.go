package main

import (
	"fmt"
	"math/rand/v2"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"chainmaker-fact-datafaker/internal/database/entity"
	"chainmaker-fact-datafaker/internal/database/mysql"
	"chainmaker-fact-datafaker/internal/logger"
	"chainmaker-fact-datafaker/pkg/chainmaker/chaincode/invoke"
	"chainmaker.org/chainmaker/common/v2/random/uuid"
	chainmakercommon "chainmaker.org/chainmaker/pb-go/v2/common"
	chainmakersdkgo "chainmaker.org/chainmaker/sdk-go/v2"
)

func main() {

	mysql.InitConn()

	sdks, err := initSdk()
	if err != nil {
		logger.GetAppLogger().Errorf("get all chain client error %v", err)
		return
	}
	fmt.Println(sdks)

	for range time.NewTicker(time.Second * 30).C {
		count := rand.IntN(30)

		for i := 0; i < count; i++ {
			random := rand.Int()
			sdk := sdks[random%len(sdks)]
			fact := entity.FactEntity{
				FileHash:  uuid.GetUUID(),
				FileName:  uuid.GetUUID(),
				TimeStamp: time.Now().UnixMilli(),
			}
			txResponse, err := invoke.CommonUpChain(sdk, "fact", "save",
				"", []*chainmakercommon.KeyValuePair{
					{Key: "file_hash", Value: []byte(fact.FileHash)},
					{Key: "file_name", Value: []byte(fact.FileName)},
					{Key: "time", Value: []byte(strconv.FormatInt(fact.TimeStamp, 10))},
				},
				-1, false)
			if err != nil {
				logger.GetAppLogger().Errorf("invoke chain error %v", err)
				continue
			} else {
				fact.TxId = txResponse.GetTxId()
				if err = fact.Save(); err != nil {
					logger.GetAppLogger().Errorf("save fact error %v", err)
				}
			}
		}
		logger.GetAppLogger().Infof("save %v facts", count)
	}
}

func initSdk() ([]*chainmakersdkgo.ChainClient, error) {

	const (
		defaultSdkConf = "sdk_configs/sdk_config.yml"
		configDirName  = "./configs"
	)

	var (
		confPaths = make(map[string]string)
		sdkInfo   []*chainmakersdkgo.ChainClient
		err       error
	)
	dirs, err := os.ReadDir(configDirName)
	if err != nil {
		logger.GetAppLogger().Errorf("read config dir %v error %v", configDirName, err)
		return nil, err
	}

	for _, dir := range dirs {
		confPaths[dir.Name()] = filepath.Clean(filepath.Join(configDirName, dir.Name(), defaultSdkConf))
	}

	for org, confPath := range confPaths {
		chainClient, err := chainmakersdkgo.NewChainClient(
			chainmakersdkgo.WithConfPath(confPath),
			chainmakersdkgo.WithChainClientLogger(logger.GetAppLogger()),
		)
		if err != nil {
			logger.GetAppLogger().Errorf("new chain client for %v error %v", org, err)
			return nil, err
		}

		if err = chainClient.EnableCertHash(); err != nil {
			logger.GetAppLogger().Errorf("enable cert hash for %v error %v", org, err)
			return nil, err
		}
		sdkInfo = append(sdkInfo, chainClient)
	}
	return sdkInfo, nil
}
