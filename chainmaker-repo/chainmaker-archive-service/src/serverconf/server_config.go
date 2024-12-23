/*
Copyright (C) BABEC. All rights reserved.


SPDX-License-Identifier: Apache-2.0
*/

// Package serverconf define config
package serverconf

import (
	"fmt"
	"os"

	"github.com/spf13/viper"
)

var (
	// GlobalServerCFG 全局变量,server配置
	GlobalServerCFG *ServerConfig
)

// LogConfig 日志配置
type LogConfig struct {
	LogInConsole bool   `mapstructure:"log_in_console"`
	ShowColor    bool   `mapstructure:"show_color"`
	LogLevel     string `mapstructure:"log_level"`
	LogPath      string `mapstructure:"log_path"`
	MaxSize      int    `mapstructure:"max_size"`
	MaxBackups   int    `mapstructure:"max_backups"`
	MaxAge       int    `mapstructure:"max_age"`
	Compress     bool   `mapstructure:"compress"`
}

// ServerConfig server配置
type ServerConfig struct {
	MonitorCFG  MonitorConfig `mapstructure:"monitor"`
	RpcCFG      RpcConfig     `mapstructure:"rpc"`
	HttpCFG     HttpConfig    `mapstructure:"http"`
	StoreageCFG StorageConfig `mapstructure:"storage_template"`
	LogCFG      LogConfig     `mapstructure:"log"`
	PProfCFG    PprofConfig   `mapstructure:"pprof"`
}

// MonitorConfig 监控配置
type MonitorConfig struct {
	Enabled bool `mapstructure:"enabled"`
	Port    int  `mapstructure:"port"`
}

// PprofConfig 配置
type PprofConfig struct {
	Enabled bool `mapstructure:"enabled"`
	Port    int  `mapstructure:"port"`
}

// RateLimitConfig 限速配置
type RateLimitConfig struct {
	Enabled         bool `mapstructure:"enabled"`
	Type            int  `mapstructure:"type"`
	TokenPerSecond  int  `mapstructure:"token_per_second"`
	TokenBucketSize int  `mapstructure:"token_bucket_size"`
}

// RpcConfig rpc配置
type RpcConfig struct {
	Port            int             `mapstructure:"port"`
	RateLimitConfig RateLimitConfig `mapstructure:"ratelimit"`
	WhiteListConfig WhiteListConfig `mapstructure:"white_list"`
	TLSEnable       bool            `mapstructure:"tls_enable"`
	TLSConfig       TlsConfig       `mapstructure:"tls"`
	MaxSendMsgSize  int             `mapstructure:"max_send_msg_size"`
	MaxRecvMsgSize  int             `mapstructure:"max_recv_msg_size"`
}

// WhiteListConfig 白名单配置
type WhiteListConfig struct {
	Enabled bool     `mapstructure:"enabled"`
	Address []string `mapstructure:"address"`
}

// HttpConfig http配置
type HttpConfig struct {
	Port            int             `mapstructure:"port"`
	CommonToken     string          `mapstructure:"common_token"`
	AdminToken      string          `mapstructure:"admin_token"`
	RateLimitConfig RateLimitConfig `mapstructure:"ratelimit"`
	WhiteListConfig WhiteListConfig `mapstructure:"white_list"`
}

// TlsConfig tls配置
type TlsConfig struct {
	//	Mode                  string   `mapstructure:"mode"`
	PrivKeyFile           string   `mapstructure:"priv_key_file"`
	CertFile              string   `mapstructure:"cert_file"`
	TestClientPrivKeyFile string   `mapstructure:"test_client_priv_key_file"`
	TestClientCertFile    string   `mapstructure:"test_client_cert_file"`
	TrustCaList           []string `mapstructure:"trust_ca_list"`
}

// StorageConfig 存储配置
type StorageConfig struct {
	// StorePath 为系统db的存储信息（leveldb使用）
	StorePath string `mapstructure:"store_path"`
	// BfdbPath 为链的db数据的存储根目录
	BfdbPath string `mapstructure:"bfdb_path"`
	// IndexPath 为链的索引数据存储的根目录（leveldb使用）
	IndexPath string `mapstructure:"index_path"`
	// CompressPath 为压缩文件的根目录
	CompressPath string `mapstructure:"compress_path"`
	// DecompressPath 为解压缩文件的根目录
	DecompressPath string `mapstructure:"decompress_path"`

	DbPrefix             string `mapstructure:"db_prefix"`
	WriteBufferSize      int    `mapstructure:"write_buffer_size"`
	BloomFilterBits      int    `mapstructure:"bloom_filter_bits"`
	BlockWriteBufferSize int    `mapstructure:"block_write_buffer_size"`

	LogDBSegmentAsync bool   `mapstructure:"logdb_segment_async"`
	LogDBSegmentSize  int    `mapstructure:"logdb_segment_size"`
	SegmentCacheSize  int    `mapstructure:"segment_cache_size"`
	UseMmap           bool   `mapstructure:"use_mmap"`
	WriteBatchSize    uint64 `mapstructure:"write_batch_size"`
	// 压缩解压缩配置
	CompressMethod string `mapstructure:"compress_method"` // gzip , 7z , 默认的为default(gzip)
	//CompressDir    string                 `mapstructure:"compress_dir"`    // 压缩文件夹
	//DecompressDir  string                 `mapstructure:"de_compress_dir"` // 解压缩到文件夹
	LevelDBCFG          map[string]interface{} `mapstructure:"leveldb_config"`        // leveldb options
	ScanIntervalSeconds int                    `mapstructure:"scan_interval_seconds"` // 扫描过期文件的时间间隔
	RetainSeconds       int                    `mapstructure:"retain_seconds"`        // 多余文件的保留时长
	CompressSeconds     int                    `mapstructure:"compress_seconds"`      //压缩,解压缩超时时间
}

// ReadConfigFile 从指定文件中读取配置信息到配置实例
// @param string
// @return error
func ReadConfigFile(confPath string) error {
	var (
		err       error
		confViper *viper.Viper
	)
	if confViper, err = initViper(confPath); err != nil {
		return fmt.Errorf("load sdk config failed, %s", err)
	}
	GlobalServerCFG = &ServerConfig{}
	if err = confViper.Unmarshal(GlobalServerCFG); err != nil {
		return fmt.Errorf("unmarshal config file failed, %s", err)
	}
	fmt.Fprintf(os.Stdout, "config is : %+v ", GlobalServerCFG)

	if err != nil {
		return err
	}
	return nil
}

// initViper 加载配置文件，解析
// @param string
// @return *viper.Viper
// @return error
func initViper(confPath string) (*viper.Viper, error) {
	cmViper := viper.New()
	cmViper.SetConfigFile(confPath)
	err := cmViper.ReadInConfig()
	if err != nil {
		return nil, err
	}

	return cmViper, nil
}
