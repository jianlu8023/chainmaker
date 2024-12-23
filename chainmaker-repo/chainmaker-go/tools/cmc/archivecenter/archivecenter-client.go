package archivecenter

import (
	"fmt"
	"io/ioutil"

	"chainmaker.org/chainmaker/common/v2/ca"
	archivePb "chainmaker.org/chainmaker/pb-go/v2/archivecenter"
	"google.golang.org/grpc"
)

type ArchiveCenterClient struct {
	conn        *grpc.ClientConn
	client      archivePb.ArchiveCenterServerClient
	maxSendSize int
}

func (client *ArchiveCenterClient) GrpcCallOption() []grpc.CallOption {
	return []grpc.CallOption{grpc.MaxCallSendMsgSize(client.maxSendSize)}
}

func (client *ArchiveCenterClient) Stop() error {
	return client.conn.Close()
}

func createArchiveCenerGrpcClient(confPath string) (*ArchiveCenterClient,
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
	retClient.client = archivePb.NewArchiveCenterServerClient(conn)
	return &retClient, nil
}
