package main

import (
	"encoding/base64"
	"encoding/hex"
	"fmt"

	util "archive-center/thridparty/utils"
	commonlog "chainmaker.org/chainmaker/common/v2/log"
	sdk "chainmaker.org/chainmaker/sdk-go/v2"
	"github.com/bytedance/sonic"
	"github.com/tidwall/gjson"
)

func main() {
	// 1. 读取sdk_configs
	fmt.Println(">>> 读取sdk_configs")
	// 2. cmc的query-by-height
	fmt.Println(">>> cmc的query-by-height")

	// 获取height 获取创世区块时 应等于0
	var height uint64
	// var err error
	// if len(args) == 0 {
	// 	height = math.MaxUint64
	// } else {
	// 	height, err = strconv.ParseUint(args[0], 10, 64)
	// 	if err != nil {
	// 		return err
	// 	}
	// }
	height = 0
	fmt.Println(">>> height = 0")

	// 获取cc
	// chainId 默认 "" 此处应该也是 ""
	// sdkConfPath 路经 后续可从baas直接获取信息构建cc
	// 目前采用配置文件方式
	// wd, _ := os.Getwd()
	sdkConfPath := "./configs/sdk_configs/sdk_config.yml"

	config := commonlog.LogConfig{
		Module:       "[SDK]",
		LogPath:      "./log/sdk.log",
		LogLevel:     commonlog.LEVEL_DEBUG,
		MaxAge:       30,
		JsonFormat:   false,
		ShowLine:     true,
		LogInConsole: false,
		// LogInConsole: true,
	}

	log, _ := commonlog.InitSugarLogger(&config)

	cc, err := sdk.NewChainClient(
		sdk.WithConfPath(sdkConfPath),
		sdk.WithChainClientChainId(""),
		sdk.WithChainClientLogger(log),
	)
	if err != nil {
		fmt.Println("获取client 失败 ", err)
		return
	}
	defer cc.Stop()
	// enableCertHash bool
	if err := util.DealChainClientCertHash(cc, true); err != nil {
		fmt.Println("证书压缩失败 ", err)
		return
	}

	// // 2.Query block on-chain.
	truncateLength := 0
	// truncateValue bool 默认true
	if true {
		truncateLength = 1000
	}
	// withRWSet 默认true  此处应该时true
	blkWithRWSetOnChain, err := cc.GetBlockByHeightTruncate(height, true, truncateLength, "truncate")
	if err != nil {
		fmt.Println("获取区块失败 ", err)
		return
	}

	header := blkWithRWSetOnChain.GetBlock().GetHeader()

	marshalString, err := sonic.MarshalString(header)
	if err != nil {
		fmt.Println("序列化失败 ", err)
		return
	}
	get := gjson.Get(marshalString, "block_hash")
	hash := get.String()
	fmt.Println(hash)
	h, base64DecodeErr := base64.StdEncoding.DecodeString(hash)
	if base64DecodeErr != nil {
		fmt.Println("解析base64失败 ", base64DecodeErr)
		return

	}
	hexString := hex.EncodeToString(h)
	fmt.Println("hexString ", hexString)

}
