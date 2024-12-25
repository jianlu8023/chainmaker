package archivecenter

import (
	"context"
	"fmt"
	"io/ioutil"
	"time"

	"chainmaker.org/chainmaker/common/v2/ca"
	archivePb "chainmaker.org/chainmaker/pb-go/v2/archivecenter"
	pbarchivecenter "chainmaker.org/chainmaker/pb-go/v2/archivecenter"
	chainmakersdkgo "chainmaker.org/chainmaker/sdk-go/v2"
	"google.golang.org/grpc"
)

const (
	genesisBlockHeight = 0
)

var (
	grpcTimeoutSeconds = 5
)

type ArchiveCenterClient struct {
	conn        *grpc.ClientConn
	Client      archivePb.ArchiveCenterServerClient
	maxSendSize int
}

func (client *ArchiveCenterClient) GrpcCallOption() []grpc.CallOption {
	return []grpc.CallOption{grpc.MaxCallSendMsgSize(client.maxSendSize)}
}

func (client *ArchiveCenterClient) Stop() error {
	return client.conn.Close()
}

func CreateArchiveCenerGrpcClient(confPath string) (*ArchiveCenterClient,
	error) {
	readErr := ReadConfigFile(confPath)
	if readErr != nil {
		return nil, readErr
	}
	return initGrpcClient()
}

func initGrpcClient() (*ArchiveCenterClient, error) {
	var retClient ArchiveCenterClient
	retClient.maxSendSize = archiveCenterCFG.MaxSendMsgSize * 1024 * 1024
	var dialOptions []grpc.DialOption
	if archiveCenterCFG.TlsEnable {
		var caFiles []string
		for _, pemFile := range archiveCenterCFG.Tls.TrustCaList {
			rootBuf, rootError := ioutil.ReadFile(pemFile)
			if rootError != nil {
				return nil, fmt.Errorf("read trust-ca-list file %s got error %s",
					pemFile, rootError.Error())
			}
			caFiles = append(caFiles, string(rootBuf))
		}
		tlsClient := ca.CAClient{
			ServerName: archiveCenterCFG.Tls.ServerName,
			CertFile:   archiveCenterCFG.Tls.CertFile,
			KeyFile:    archiveCenterCFG.Tls.PrivKeyFile,
			CaCerts:    caFiles,
		}
		creds, credsErr := tlsClient.GetCredentialsByCA()
		if credsErr != nil {
			return nil, fmt.Errorf("GetCredentialsByCA error %s", credsErr.Error())
		}
		dialOptions = append(dialOptions, grpc.WithTransportCredentials(*creds))
	} else {
		dialOptions = append(dialOptions, grpc.WithInsecure())
	}
	dialOptions = append(dialOptions,
		grpc.WithDefaultCallOptions(
			grpc.MaxCallRecvMsgSize(
				archiveCenterCFG.MaxRecvMsgSize*1024*1024)))
	conn, connErr := grpc.Dial(
		archiveCenterCFG.RpcAddress,
		dialOptions...)
	if connErr != nil {
		return nil, fmt.Errorf("dial grpc error %s", connErr.Error())
	}
	retClient.conn = conn
	retClient.Client = archivePb.NewArchiveCenterServerClient(conn)
	return &retClient, nil
}

func RegisterChainToArchiveCenter(chainClient *chainmakersdkgo.ChainClient,
	archiveClient *ArchiveCenterClient) (string, error) {
	statusCtx, statusCancel := context.WithTimeout(context.Background(),
		time.Duration(grpcTimeoutSeconds)*time.Second)
	defer statusCancel()
	archivedStatus, archiveStatusErr := archiveClient.Client.GetArchivedStatus(
		statusCtx, &pbarchivecenter.ArchiveStatusRequest{
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
	registerResp, registerError := archiveClient.Client.Register(ctx,
		&pbarchivecenter.ArchiveBlockRequest{
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
		registerResp.RegisterStatus == pbarchivecenter.RegisterStatus_RegisterStatusSuccess {
		return genesisHash, nil
	}
	return genesisHash, fmt.Errorf("register got code %d , message %s, status %d",
		registerResp.Code, registerResp.Message, registerResp.RegisterStatus)
}

func ArchiveBlockByHeight(chainGenesis string, height uint64,
	chainClient *chainmakersdkgo.ChainClient,
	archiveClient pbarchivecenter.ArchiveCenterServer_ArchiveBlocksClient,
	singleClientStream pbarchivecenter.ArchiveCenterServer_SingleArchiveBlocksClient,
	isQuickMode bool) (bool, error) {
	queryBlockFailed := false
	block, blockError := chainClient.GetFullBlockByHeight(height)
	if blockError != nil {
		queryBlockFailed = true
		return queryBlockFailed, fmt.Errorf("query block height %d got error %s",
			height, blockError.Error())

	}
	if isQuickMode {
		singleSendErr := singleClientStream.Send(&pbarchivecenter.ArchiveBlockRequest{
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
	sendErr := archiveClient.Send(&pbarchivecenter.ArchiveBlockRequest{
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
	if archiveResp.ArchiveStatus == pbarchivecenter.ArchiveStatus_ArchiveStatusFailed {
		return queryBlockFailed, fmt.Errorf("send height %d failed %s ", height, archiveResp.Message)
	}
	return queryBlockFailed, nil
}
