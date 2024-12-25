package main

import (
	"context"
	"fmt"
	"strings"

	"archive-center/thridparty/archivecenter"
	util "archive-center/thridparty/utils"
	commonlog "chainmaker.org/chainmaker/common/v2/log"
	pbarchivecenter "chainmaker.org/chainmaker/pb-go/v2/archivecenter"
	"github.com/gosuri/uiprogress"
	"github.com/spf13/pflag"
)

const (
	flagSdkConfPath        = "sdk-conf-path"
	flagArchiveConfPath    = "archive-conf-path"
	flagArchiveBeginHeight = "archive-begin-height"
	flagArchiveEndHeight   = "archive-end-height"
	flagChainId            = "chain-id"
	flagArchiveMode        = "mode"
	archiveModeQuick       = "quick"
	archivedModeNormal     = "normal"
)

var (
	sdkConfPath           string // sdk-go的配置信息
	archiveCenterConfPath string // 归档中心文件配置路径
	chainId               string // chainid
	archiveBeginHeight    uint64 // 开始归档区块
	archiveEndHeight      uint64 // 结束归档区块
	archiveMode           string // 是否使用快速归档(是的话为quick默认为否)
	flags                 *pflag.FlagSet
)

func main() {

	// 检查归档开始高度和归档结束高度
	archiveBeginHeight = 1
	archiveEndHeight = 100
	if archiveBeginHeight >= archiveEndHeight {
		fmt.Println(">>> 归档开始 > 归档结束")
		return
	}

	sdkConfPath = "./configs/sdk_configs/sdk_config.yml"

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
	// chainId 链名称 默认 "" 此处应该也是 链名称 在命令行中传了参数 --chain-id
	cc, err := util.CreateChainClient(sdkConfPath, "archivetest",
		"", "",
		"", "", "",
		log)

	if err != nil {
		fmt.Println("获取client 失败 ", err)
		return
	}
	defer cc.Stop()

	// create archivecenter grpc client
	// archiveCenterConfPath 是archivecenter/config.yml
	archiveCenterConfPath = "./configs/archivecenter/config.yml"
	archiveClient, clientErr := archivecenter.CreateArchiveCenerGrpcClient(archiveCenterConfPath)
	if clientErr != nil {
		fmt.Println("获取 archivecenter grpc client 失败 ", clientErr)
		return
	}
	defer archiveClient.Stop()
	// 1. register genesis block
	genesisHash, registerErr := archivecenter.RegisterChainToArchiveCenter(cc, archiveClient)
	if registerErr != nil {
		fmt.Println("register chain to archive center error ", registerErr)
		return
	}

	barCount := archiveEndHeight - archiveBeginHeight + 1
	progress := uiprogress.New()
	bar := progress.AddBar(int(barCount)).AppendCompleted().PrependElapsed()
	bar.PrependFunc(func(b *uiprogress.Bar) string {
		return fmt.Sprintf("Archiving Blocks (%d/%d)\n", b.Current(), barCount)
	})
	progress.Start()
	defer progress.Stop()

	// 默认 不是quick 那就是normal
	archiveMode = "normal"
	isQuickMode := strings.TrimSpace(archiveMode) == archiveModeQuick
	var singleClientStream pbarchivecenter.ArchiveCenterServer_SingleArchiveBlocksClient
	var clientStream pbarchivecenter.ArchiveCenterServer_ArchiveBlocksClient
	var streamErr error
	if isQuickMode {
		singleClientStream, streamErr =
			archiveClient.Client.SingleArchiveBlocks(context.Background(),
				archiveClient.GrpcCallOption()...)
		if streamErr != nil {
			fmt.Println("quick mode stream error ", streamErr)
			return
		}
	} else {
		clientStream, streamErr = archiveClient.Client.ArchiveBlocks(context.Background(),
			archiveClient.GrpcCallOption()...)
		if streamErr != nil {
			fmt.Println("normal mode stream error ", streamErr)
			return
		}
	}
	var archiveError error
	var queryBlockFailed bool
	for tempHeight := archiveBeginHeight; tempHeight <= archiveEndHeight; tempHeight++ {
		queryBlockFailed, archiveError = archivecenter.ArchiveBlockByHeight(genesisHash, tempHeight,
			cc, clientStream, singleClientStream, isQuickMode)
		if queryBlockFailed { // 如果查询区块失败,直接返回
			break
		}
		if archiveError != nil {
			break
		}
		bar.Incr()
	}
	if isQuickMode {
		archiveResp, archiveRespErr := singleClientStream.CloseAndRecv()
		if archiveRespErr != nil {
			fmt.Println(fmt.Errorf("stream close recv error %s", archiveRespErr.Error()))
			return
		}
		if archiveResp != nil {
			fmt.Printf("archive resp code %d ,message %s , begin %d , end %d \n",
				archiveResp.Code, archiveResp.Message,
				archiveResp.ArchivedBeginHeight, archiveResp.ArchivedEndHeight)
		}
		if archiveError != nil && !queryBlockFailed {
			fmt.Println("quick mode query block failed and archive error ", archiveError)
			return
		}
		return
	}
	archiveRespErr := clientStream.CloseSend()
	if archiveRespErr != nil {
		fmt.Println(fmt.Errorf("stream close recv error %s", archiveRespErr.Error()))
		return
	}
	if archiveError != nil && !queryBlockFailed {
		fmt.Println("normal mode query block failed and archive error ", archiveClient)
		return
	}
	return
}
