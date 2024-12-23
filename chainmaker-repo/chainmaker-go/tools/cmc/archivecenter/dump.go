package archivecenter

import (
	"context"
	"fmt"
	"strings"
	"time"

	"chainmaker.org/chainmaker-go/tools/cmc/util"
	"chainmaker.org/chainmaker/pb-go/v2/archivecenter"
	chainmaker_sdk_go "chainmaker.org/chainmaker/sdk-go/v2"
	"github.com/gosuri/uiprogress"
	"github.com/spf13/cobra"
)

const (
	genesisBlockHeight = 0
)

var (
	grpcTimeoutSeconds = 5
)

func newDumpCMD() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "dump",
		Short: "dump blockchain data",
		Long:  "dump blockchain data to archive center storage",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(chainId) == 0 ||
				len(sdkConfPath) == 0 ||
				len(archiveCenterConfPath) == 0 {
				return fmt.Errorf("%s or %s or %s must not be empty",
					flagChainId, flagSdkConfPath, flagArchiveConfPath)
			}
			if archiveBeginHeight >= archiveEndHeight {
				return fmt.Errorf("%s must be greater than %s",
					flagArchiveEndHeight, flagArchiveBeginHeight)
			}
			return runDumpCMD(archiveBeginHeight, archiveEndHeight)
		},
	}
	util.AttachAndRequiredFlags(cmd, flags,
		[]string{
			flagSdkConfPath, flagArchiveConfPath, flagChainId,
			flagArchiveBeginHeight, flagArchiveEndHeight, flagArchiveMode,
		})
	return cmd
}

func runDumpCMD(beginHeight, endHeight uint64) error {
	// create chain client
	cc, err := util.CreateChainClient(sdkConfPath, chainId,
		"", "", "", "", "")
	if err != nil {
		return err
	}
	defer cc.Stop()
	// create archivecenter grpc client
	archiveClient, clientErr := createArchiveCenerGrpcClient(archiveCenterConfPath)
	if clientErr != nil {
		return clientErr
	}
	defer archiveClient.Stop()
	// 1. register genesis block
	genesisHash, registerErr := registerChainToArchiveCenter(cc, archiveClient)
	if registerErr != nil {
		return registerErr
	}

	barCount := archiveEndHeight - archiveBeginHeight + 1
	progress := uiprogress.New()
	bar := progress.AddBar(int(barCount)).AppendCompleted().PrependElapsed()
	bar.PrependFunc(func(b *uiprogress.Bar) string {
		return fmt.Sprintf("Archiving Blocks (%d/%d)\n", b.Current(), barCount)
	})
	progress.Start()
	defer progress.Stop()
	isQuickMode := strings.TrimSpace(archiveMode) == archiveModeQuick
	var singleClientStream archivecenter.ArchiveCenterServer_SingleArchiveBlocksClient
	var clientStream archivecenter.ArchiveCenterServer_ArchiveBlocksClient
	var streamErr error
	if isQuickMode {
		singleClientStream, streamErr =
			archiveClient.client.SingleArchiveBlocks(context.Background(),
				archiveClient.GrpcCallOption()...)
		if streamErr != nil {
			return streamErr
		}
	} else {
		clientStream, streamErr = archiveClient.client.ArchiveBlocks(context.Background(),
			archiveClient.GrpcCallOption()...)
		if streamErr != nil {
			return streamErr
		}
	}
	var archiveError error
	var queryBlockFailed bool
	for tempHeight := archiveBeginHeight; tempHeight <= archiveEndHeight; tempHeight++ {
		queryBlockFailed, archiveError = archiveBlockByHeight(genesisHash, tempHeight,
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
			return fmt.Errorf("stream close recv error %s", archiveRespErr.Error())
		}
		if archiveResp != nil {
			fmt.Printf("archive resp code %d ,message %s , begin %d , end %d \n",
				archiveResp.Code, archiveResp.Message,
				archiveResp.ArchivedBeginHeight, archiveResp.ArchivedEndHeight)
		}
		if archiveError != nil && !queryBlockFailed {
			return archiveError
		}
		return nil
	}
	archiveRespErr := clientStream.CloseSend()
	if archiveRespErr != nil {
		return fmt.Errorf("stream close recv error %s", archiveRespErr.Error())
	}
	if archiveError != nil && !queryBlockFailed {
		return archiveError
	}
	return nil
}

func registerChainToArchiveCenter(chainClient *chainmaker_sdk_go.ChainClient,
	archiveClient *ArchiveCenterClient) (string, error) {
	statusCtx, statusCancel := context.WithTimeout(context.Background(),
		time.Duration(grpcTimeoutSeconds)*time.Second)
	defer statusCancel()
	archivedStatus, archiveStatusErr := archiveClient.client.GetArchivedStatus(
		statusCtx, &archivecenter.ArchiveStatusRequest{
			ChainUnique: archiveCenterCFG.ChainGenesisHash,
		})
	if archiveStatusErr == nil && archivedStatus != nil && archivedStatus.Code == 0 {
		fmt.Printf("chain %s archivestatus %+v \n",
			archiveCenterCFG.ChainGenesisHash, *archivedStatus)
		return archiveCenterCFG.ChainGenesisHash, nil
	}
	genesisBlock, genesisErr := chainClient.GetFullBlockByHeight(genesisBlockHeight)
	if genesisErr != nil {
		return "", fmt.Errorf("query genesis block error %s", genesisErr.Error())
	}
	genesisHash := genesisBlock.Block.GetBlockHashStr()
	ctx, ctxCancel := context.WithTimeout(context.Background(),
		time.Duration(grpcTimeoutSeconds)*time.Second)
	defer ctxCancel()
	registerResp, registerError := archiveClient.client.Register(ctx,
		&archivecenter.ArchiveBlockRequest{
			ChainUnique: genesisHash,
			Block:       genesisBlock,
		}, archiveClient.GrpcCallOption()...)
	if registerError != nil {
		return genesisHash, fmt.Errorf("register genesis rpc error %s", genesisErr.Error())
	}
	if registerResp == nil {
		return genesisHash, fmt.Errorf("register genesis rpc no response")
	}
	if registerResp.Code == 0 &&
		registerResp.RegisterStatus == archivecenter.RegisterStatus_RegisterStatusSuccess {
		return genesisHash, nil
	}
	return genesisHash, fmt.Errorf("register got code %d , message %s, status %d",
		registerResp.Code, registerResp.Message, registerResp.RegisterStatus)
}

func archiveBlockByHeight(chainGenesis string, height uint64,
	chainClient *chainmaker_sdk_go.ChainClient,
	archiveClient archivecenter.ArchiveCenterServer_ArchiveBlocksClient,
	singleClientStream archivecenter.ArchiveCenterServer_SingleArchiveBlocksClient,
	isQuickMode bool) (bool, error) {
	queryBlockFailed := false
	block, blockError := chainClient.GetFullBlockByHeight(height)
	if blockError != nil {
		queryBlockFailed = true
		return queryBlockFailed, fmt.Errorf("query block height %d got error %s",
			height, blockError.Error())

	}
	if isQuickMode {
		singleSendErr := singleClientStream.Send(&archivecenter.ArchiveBlockRequest{
			ChainUnique: chainGenesis,
			Block:       block,
		})
		if singleSendErr != nil {
			return queryBlockFailed, fmt.Errorf("send height %d got error %s",
				height, singleSendErr.Error())
		} else {
			return queryBlockFailed, nil
		}
	}
	sendErr := archiveClient.Send(&archivecenter.ArchiveBlockRequest{
		ChainUnique: chainGenesis,
		Block:       block,
	})
	if sendErr != nil {
		return queryBlockFailed, fmt.Errorf("send height %d got error %s",
			height, sendErr.Error())
	}
	archiveResp, archiveRespErr := archiveClient.Recv()
	if archiveRespErr != nil {
		return queryBlockFailed, fmt.Errorf("send height %d got error %s", height, archiveRespErr.Error())
	}
	if archiveResp.ArchiveStatus == archivecenter.ArchiveStatus_ArchiveStatusFailed {
		return queryBlockFailed, fmt.Errorf("send height %d failed %s ", height, archiveResp.Message)
	}
	return queryBlockFailed, nil
}
