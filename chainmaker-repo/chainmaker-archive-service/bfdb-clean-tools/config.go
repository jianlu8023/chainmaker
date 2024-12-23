/*
Copyright (C) BABEC. All rights reserved.


SPDX-License-Identifier: Apache-2.0
*/

package main

import (
	"fmt"
	"os"

	"github.com/spf13/viper"
)

var (
	// GlobalServerCFG 配置信息变量
	GlobalServerCFG *ServerConfig
)

// LogConfig 日志配置
type LogConfig struct {
	LogInConsole bool   `mapstructure:"log_in_console"`
	ShowColor    bool   `mapstructure:"show_color"`
	LogLevel     string `mapstructure:"log_level"`
	LogPath      string `mapstructure:"log_path"`
}

// ArchiveCenterCFG 归档中心http配置
type ArchiveCenterCFG struct {
	ChainGenesisHash     string `mapstructure:"chain_genesis_hash"`
	ArchiveCenterHttpUrl string `mapstructure:"archive_center_http_url"` // 归档中心的http地址
	ReqeustSecondLimit   int    `mapstructure:"request_second_limit"`    // 查询归档中心http接口超时时间
}

// ServerConfig 服务配置
type ServerConfig struct {
	LogCFG      LogConfig        `mapstructure:"log"`          // 日志信息配置
	BfdbPath    string           `mapstructure:"bfdb_path"`    // 配置的bfdb文件存储路径
	Mode        int              `mapstructure:"mode"`         // 1为仅输出信息模式;2为清理模式
	CleanStub   string           `mapstructure:"clean_stub"`   // 只有mode为2的时候有效果
	CleanHeight uint64           `mapstructure:"clean_height"` // 设置的最大高度
	ArchiveCFG  ArchiveCenterCFG `mapstructure:"archive_center_config"`
}

// ReadConfigFile 读取配置信息
// @param confPath
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
	dealConfig(GlobalServerCFG)
	fmt.Fprintf(os.Stdout, "config is : %+v ", GlobalServerCFG)

	if err != nil {
		return err
	}
	return nil
}

// dealConfig 验证配置
// @param cfg
func dealConfig(cfg *ServerConfig) {
	if cfg.Mode < 1 || cfg.Mode > 2 {
		panic("mode must be 1 or 2")
	}
	if cfg.Mode == 2 && len(cfg.CleanStub) == 0 {
		panic("mode 2 must set clean_stub")
	}
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
