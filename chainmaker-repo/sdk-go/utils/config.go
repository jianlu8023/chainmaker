/*
Copyright (C) BABEC. All rights reserved.
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package utils

import (
	"fmt"
	"strings"

	"github.com/spf13/viper"
)

type cryptoModel struct {
	Hash string `mapstructure:"hash"`
}

type nodesConfigModel struct {
	// 节点地址
	NodeAddr string `mapstructure:"node_addr"`
	// 节点连接数
	ConnCnt int `mapstructure:"conn_cnt"`
	// RPC连接是否启用双向TLS认证
	EnableTLS bool `mapstructure:"enable_tls"`
	// 信任证书池路径
	TrustRootPaths []string `mapstructure:"trust_root_paths"`
	// TLS hostname
	TLSHostName string `mapstructure:"tls_host_name"`
}

type archiveConfigModel struct {
	// 链外存储类型，已支持：mysql
	Type string `mapstructure:"type"`
	// 链外存储目标，格式：
	// 	- mysql: user:pwd:host:port
	Dest      string `mapstructure:"dest"`
	SecretKey string `mapstructure:"secret_key"`
}

type rpcClientConfigModel struct {
	MaxRecvMsgSize int   `mapstructure:"max_receive_message_size"`
	MaxSendMsgSize int   `mapstructure:"max_send_message_size"`
	SendTxTimeout  int64 `mapstructure:"send_tx_timeout"`
	GetTxTimeout   int64 `mapstructure:"get_tx_timeout"`
}

type pkcs11ConfigModel struct {
	// 是否开启pkcs11
	Enabled bool `mapstructure:"enabled"`
	// interface type of lib, only support pkcs11 and sdf
	Type string `mapstructure:"type"`
	// path to the .so file of pkcs11 interface
	Library string `mapstructure:"library"`
	// label for the slot to be used
	Label string `mapstructure:"label"`
	// password to logon the HSM(Hardware security module)
	Password string `mapstructure:"password"`
	// size of HSM session cache
	SessionCacheSize int `mapstructure:"session_cache_size"`
	// hash algorithm used to compute SKI, eg, SHA256
	Hash string `mapstructure:"hash"`
}

type archiveCenterHttpConfigModel struct {
	ChainGenesisHash     string `mapstructure:"chain_genesis_hash"`
	ArchiveCenterHttpUrl string `mapstructure:"archive_center_http_url"`
	ReqeustSecondLimit   int    `mapstructure:"request_second_limit"` // http请求的超时间隔,默认5秒
}

type chainClientConfigModel struct {
	// 链ID
	ChainId string `mapstructure:"chain_id"`
	// 组织ID
	OrgId string `mapstructure:"org_id"`
	// 客户端用户私钥路径
	UserKeyFilePath string `mapstructure:"user_key_file_path"`
	// 客户端用户私钥密码
	UserKeyPwd string `mapstructure:"user_key_pwd"`
	// 客户端用户证书路径
	UserCrtFilePath string `mapstructure:"user_crt_file_path"`
	// 客户端用户加密私钥路径
	UserEncKeyFilePath string `mapstructure:"user_enc_key_file_path"`
	// 客户端用户加密私钥密码
	UserEncKeyPwd string `mapstructure:"user_enc_key_pwd"`
	// 客户端用户加密证书路径
	UserEncCrtFilePath string `mapstructure:"user_enc_crt_file_path"`
	// 证书模式下：客户端用户交易签名私钥路径(若未设置，将使用user_key_file_path)
	// 公钥模式下：客户端用户交易签名的私钥路径(必须设置)
	UserSignKeyFilePath string `mapstructure:"user_sign_key_file_path"`
	// 客户端用户交易签名私钥密码
	UserSignKeyPwd string `mapstructure:"user_sign_key_pwd"`
	// 客户端用户交易签名证书路径(若未设置，将使用user_crt_file_path)
	UserSignCrtFilePath string `mapstructure:"user_sign_crt_file_path"`
	// 同步交易结果模式下，轮训获取交易结果时的最大轮训次数
	RetryLimit int `mapstructure:"retry_limit"`
	// 同步交易结果模式下，每次轮训交易结果时的等待时间 单位：ms
	RetryInterval int `mapstructure:"retry_interval"`
	// 节点配置
	NodesConfig []nodesConfigModel `mapstructure:"nodes"`
	// 归档特性的配置
	ArchiveConfig *archiveConfigModel `mapstructure:"archive,omitempty"`
	// 设置grpc客户端配置
	RPCClientConfig *rpcClientConfigModel `mapstructure:"rpc_client"`
	// pkcs11配置(若未设置，则不使用pkcs11)
	Pkcs11Config *pkcs11ConfigModel `mapstructure:"pkcs11"`
	// 认证模式
	AuthType string `mapstructure:"auth_type"`
	// 需要额外指定的算法类型，当前只用于指定公钥身份模式下的Hash算法
	Crypto *cryptoModel `mapstructure:"crypto"`
	// 别名
	Alias string `mapstructure:"alias"`
	// 默认使用 TimestampKey ，如果 EnableNormalKey 设置为 true 则使用 NormalKey
	EnableNormalKey bool `mapstructure:"enable_normal_key"`
	// ArchiveCenterConfig 归档中心的http配置
	ArchiveCenterConfig *archiveCenterHttpConfigModel `mapstructure:"archive_center_config"`
}

// ChainClientConfigModel define ChainClientConfigModel
type ChainClientConfigModel struct {
	ChainClientConfig chainClientConfigModel `mapstructure:"chain_client"`
}

// InitConfig init config from config file path
func InitConfig(confPath string) (*ChainClientConfigModel, error) {
	var (
		err       error
		confViper *viper.Viper
	)

	if confViper, err = initViper(confPath); err != nil {
		return nil, fmt.Errorf("Load sdk config failed, %s", err)
	}

	configModel := &ChainClientConfigModel{}
	if err = confViper.Unmarshal(&configModel); err != nil {
		return nil, fmt.Errorf("Unmarshal config file failed, %s", err)
	}

	configModel.ChainClientConfig.AuthType = strings.ToLower(configModel.ChainClientConfig.AuthType)

	return configModel, nil
}

func initViper(confPath string) (*viper.Viper, error) {
	cmViper := viper.New()
	cmViper.SetConfigFile(confPath)
	if err := cmViper.ReadInConfig(); err != nil {
		return nil, err
	}

	return cmViper, nil
}
