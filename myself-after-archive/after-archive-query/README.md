
# 归档操作后 变化

1. go.mod 的变化

tips: 甚至可以升级到2.3.6使用 这个应该无所谓

```text

require (
	chainmaker.org/chainmaker/common/v2 v2.3.0
	chainmaker.org/chainmaker/pb-go/v2 v2.3.2-0.20221212031024-fc4ec021d4ca
	chainmaker.org/chainmaker/sdk-go/v2 v2.3.2-0.20221208074130-988888d1d02f
)
```

2. sdk_config.yml的变化

```yml
chain_client:
  chain_id: archive
  org_id: gmorg1
  user_key_file_path: ./configs/sdk_configs/gmorg1/sdk_configs/crypto-config/gmorg1/user/gmorg1admin/gmorg1admin.tls.key
  user_crt_file_path: ./configs/sdk_configs/gmorg1/sdk_configs/crypto-config/gmorg1/user/gmorg1admin/gmorg1admin.tls.crt
  user_sign_key_file_path: ./configs/sdk_configs/gmorg1/sdk_configs/crypto-config/gmorg1/user/gmorg1admin/gmorg1admin.sign.key
  user_sign_crt_file_path: ./configs/sdk_configs/gmorg1/sdk_configs/crypto-config/gmorg1/user/gmorg1admin/gmorg1admin.sign.crt
  retry_limit: 10
  retry_interval: 500
  nodes:
  - node_addr: 172.25.138.51:11010
    conn_cnt: 10
    enable_tls: true
    trust_root_paths:
    - ./configs/sdk_configs/gmorg1/sdk_configs/crypto-config/gmorg1/ca
    tls_host_name: chainmaker.org
  archive:
    type: mysql
    dest: root:123456:localhost:3306
    secret_key: xxx
  rpc_client:
    max_receive_message_size: 16
    max_send_message_size: 16
  pkcs11:
    enabled: false
    type: ""
    library: /usr/local/lib64/pkcs11/libupkcs11.so
    label: HSM
    password: "11111111"
    session_cache_size: 10
    hash: SHA256
  # 添加了这里 
  archive_center_config:
    chain_genesis_hash: a840c4e495975d90b4c5bb7d0b7641df8028163667132d0d0ab06b876987b433 # 归档的链的创世区块的hash
    archive_center_http_url: http://172.25.138.51:13119 # 归档中心的http接口地址
    request_second_limit: 10 # 查询归档中心http请求超时时间

```

3. 代码的变化

tips:
 若使用节点信息等信息创建chainclient 的时候

```go
package main

import (
    "fmt"
    
    chainmakersdkgo "chainmaker.org/chainmaker/sdk-go/v2"
)

func main() {
    node := chainmakersdkgo.NewNodeConfig(
        // 节点地址，格式：127.0.0.1:12301
        chainmakersdkgo.WithNodeAddr("127.0.0.1:11010"),
        // 节点连接数
        chainmakersdkgo.WithNodeConnCnt(10),
        // 节点是否启用TLS认证
        chainmakersdkgo.WithNodeUseTLS(false),
        // 根证书路径，支持多个
        chainmakersdkgo.WithNodeCAPaths([]string{"path/to/ca"}),
        // TLS Hostname
        chainmakersdkgo.WithNodeTLSHostName("chainmaker.org"),
    )
    
    client, err := chainmakersdkgo.NewChainClient(
        // 设置归属组织
        chainmakersdkgo.WithChainClientOrgId(""),
        // 设置链ID
        chainmakersdkgo.WithChainClientChainId("archive"),
        // 设置logger句柄，若不设置，将采用默认日志文件输出日志
        chainmakersdkgo.WithChainClientLogger(logger),
        // 设置客户端用户私钥路径
        chainmakersdkgo.WithUserKeyBytes([]byte("")),
        chainmakersdkgo.WithUserKeyFilePath("userKeyPath"),
        // 设置客户端用户证书
        chainmakersdkgo.WithUserCrtBytes([]byte("")),
        chainmakersdkgo.WithUserCrtFilePath("userCrtPath"),
        // 添加节点1
        chainmakersdkgo.AddChainClientNodeConfig(node),
        // 归档中心
        chainmakersdkgo.WithArchiveCenterHttpConfig(&chainmakersdkgo.ArchiveCenterHttpConfig{}),
    )
    
    if err != nil {
        
        fmt.Println("new chain client error", err)
        return
    }
    
    fmt.Println(client)
}

```

```text
# 有变化的方法  这些方法只是查询前到归档中心查询了一次 走http请求的方式

GetChainConfigByBlockHeight(blockHeight uint64) (*config.ChainConfig, error)

GetTxByTxId(txId string) (*common.TransactionInfo, error)

GetTxWithRWSetByTxId(txId string) (*common.TransactionInfoWithRWSet, error)

GetBlockByHeight(blockHeight uint64, withRWSet bool) (*common.BlockInfo, error)

GetBlockByHash(blockHash string, withRWSet bool) (*common.BlockInfo, error)

GetBlockByTxId(txId string, withRWSet bool) (*common.BlockInfo, error)

GetFullBlockByHeight(blockHeight uint64) (*store.BlockWithRWSet, error)

GetBlockHeaderByHeight(blockHeight uint64) (*common.BlockHeader, error)

GetMerklePathByTxId(txId string) ([]byte, error)

GetBlockByHeightTruncate(blockHeight uint64, withRWSet bool, truncateLength int,truncateModel string)(*common.BlockInfo, error)

GetTxByTxIdTruncate(txId string, withRWSet bool, truncateLength int,truncateModel string) (*common.TransactionInfoWithRWSet, error)



```

