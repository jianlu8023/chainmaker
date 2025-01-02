package main

import (
	"encoding/json"
	"fmt"
	"log"

	"after-archive-query/internal/logger"
	chainmakercommon "chainmaker.org/chainmaker/pb-go/v2/common"
	chainmakersdkgo "chainmaker.org/chainmaker/sdk-go/v2"
)

func main() {
	confPath := "./configs/sdk_configs/gmorg1/sdk_configs/sdk_config.yml"
	cc, err := chainmakersdkgo.NewChainClient(
		chainmakersdkgo.WithConfPath(confPath),
		chainmakersdkgo.WithChainClientLogger(logger.GenChainMakerLogger(true)),
	)

	// if err != nil {
	//
	// 	fmt.Println("new logger error", err)
	// 	return
	// }
	// node := sdk.NewNodeConfig(
	// 	// 节点地址，格式：127.0.0.1:12301
	// 	sdk.WithNodeAddr("127.0.0.1:11010"),
	// 	// 节点连接数
	// 	sdk.WithNodeConnCnt(10),
	// 	// 节点是否启用TLS认证
	// 	sdk.WithNodeUseTLS(false),
	// 	// 根证书路径，支持多个
	// 	sdk.WithNodeCAPaths([]string{"path/to/ca"}),
	// 	// TLS Hostname
	// 	sdk.WithNodeTLSHostName("chainmaker.org"),
	// )
	//
	// client, err := sdk.NewChainClient(
	// 	// 设置归属组织
	// 	sdk.WithChainClientOrgId(""),
	// 	// 设置链ID
	// 	sdk.WithChainClientChainId("archive"),
	// 	// 设置logger句柄，若不设置，将采用默认日志文件输出日志
	// 	sdk.WithChainClientLogger(logger),
	// 	// 设置客户端用户私钥路径
	// 	sdk.WithUserKeyBytes([]byte("")),
	// 	sdk.WithUserKeyFilePath("userKeyPath"),
	// 	// 设置客户端用户证书
	// 	sdk.WithUserCrtBytes([]byte("")),
	// 	sdk.WithUserCrtFilePath("userCrtPath"),
	// 	// 添加节点1
	// 	sdk.AddChainClientNodeConfig(node),
	// 	// 归档中心
	// 	sdk.WithArchiveCenterHttpConfig(&sdk.ArchiveCenterHttpConfig{}),
	// )
	//
	// if err != nil {
	//
	// 	fmt.Println("new chain client error", err)
	// 	return
	// }
	//
	// fmt.Println(client)

	if err != nil {
		log.Fatal("new chain client failed error ", err)
		return
	}
	defer cc.Stop()

	txResponse, err := cc.QueryContract("fact", "findByFileHash", []*chainmakercommon.KeyValuePair{
		{Key: "file_hash", Value: []byte("eca2d5a815e94b00805739c0d80ccdb7")},
	},
		-1,
	)
	if err != nil {
		log.Fatal("query contract error ", err)
		return
	}
	if txResponse.Code != chainmakercommon.TxStatusCode_SUCCESS {
		log.Fatal("query contract failed ", txResponse.Code.String())
		return
	}
	log.Println("query contract result ", txResponse)

	fmt.Println("11111111111111111111111111111111111111111")

	txId, err := cc.GetTxByTxId("18158f4c62f05a3dca10c393ad7cc7b21d90fceab65b4edca2e4b5693d2912f4")
	if err != nil {
		log.Fatal("get tx by txId error ", err)
		return
	}
	log.Println("get tx by txId result ", txId)
	height := txId.BlockHeight
	fmt.Println("height ", height)
	transaction := txId.GetTransaction()
	marshal, err := json.Marshal(transaction)
	if err != nil {
		log.Fatal("marshal transaction error ", err)
		return
	}
	fmt.Println(string(marshal))

}