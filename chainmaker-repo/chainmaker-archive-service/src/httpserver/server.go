/*
Copyright (C) BABEC. All rights reserved.


SPDX-License-Identifier: Apache-2.0
*/

// Package httpserver define http operation
package httpserver

import (
	"fmt"
	"net/http"
	"os"

	"chainmaker.org/chainmaker-archive-service/src/logger"
	"chainmaker.org/chainmaker-archive-service/src/process"
	"chainmaker.org/chainmaker-archive-service/src/serverconf"
	"github.com/gin-contrib/gzip"
	"github.com/gin-gonic/gin"
)

// HttpSrv http服务结构
type HttpSrv struct {
	router         *gin.Engine
	ProxyProcessor *process.ProcessorMgr
	ApiSrv         *http.Server
	logger         *logger.WrappedLogger
}

// NewHttpServer 构造新的http服务器
func NewHttpServer(proxy *process.ProcessorMgr) *HttpSrv {
	gin.SetMode(gin.ReleaseMode)
	apiLog := logger.NewLogger("APISERVICE", &serverconf.LogConfig{
		LogPath:      fmt.Sprintf("%s/apiservice.log", serverconf.GlobalServerCFG.LogCFG.LogPath),
		LogLevel:     serverconf.GlobalServerCFG.LogCFG.LogLevel,
		LogInConsole: serverconf.GlobalServerCFG.LogCFG.LogInConsole,
		ShowColor:    serverconf.GlobalServerCFG.LogCFG.ShowColor,
		MaxSize:      serverconf.GlobalServerCFG.LogCFG.MaxSize,
		MaxBackups:   serverconf.GlobalServerCFG.LogCFG.MaxBackups,
		MaxAge:       serverconf.GlobalServerCFG.LogCFG.MaxAge,
		Compress:     serverconf.GlobalServerCFG.LogCFG.Compress,
	})
	return &HttpSrv{
		router:         gin.New(),
		ProxyProcessor: proxy,
		logger:         apiLog,
	}
}

// Listen 开启http监听
func (srv *HttpSrv) Listen(apiCfg serverconf.HttpConfig) {
	if serverconf.GlobalServerCFG.HttpCFG.WhiteListConfig.Enabled {
		srv.router.Use(gin.Recovery(), whiteListMiddleWare(), rateLimitMiddleWare(), gzip.Gzip(gzip.DefaultCompression))
	} else {
		srv.router.Use(gin.Recovery(), rateLimitMiddleWare(), gzip.Gzip(gzip.DefaultCompression))
	}

	srv.registerRouters()
	srv.ApiSrv = &http.Server{
		Addr:    fmt.Sprintf(":%d", apiCfg.Port),
		Handler: srv.router,
	}
	go func() {
		if err := srv.ApiSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			fmt.Fprintf(os.Stderr, " http server start error , %s \n", err.Error())
		}
	}()
}

func (srv *HttpSrv) registerRouters() {
	srv.router.POST("/get_archived_height", srv.GetArchivedHeight)
	srv.router.POST("/get_range_blocks", srv.GetRangeBlocks)
	srv.router.POST("/get_archive_status", srv.GetInArchiveAndArchivedHeight)
	srv.router.POST("/block_exists_by_hash", srv.BlockExists)
	srv.router.POST("/get_block_by_hash", srv.GetBlockByHash)
	srv.router.POST("/get_height_by_hash", srv.GetHeightByHash)
	srv.router.POST("/get_block_by_height", srv.GetBlock)
	srv.router.POST("/get_tx_by_tx_id", srv.GetTx)
	srv.router.POST("/get_tx_block_by_tx_id", srv.GetTxWithBlockInfo)
	srv.router.POST("/get_tx_info_by_tx_id", srv.GetTxInfoOnly)
	srv.router.POST("/get_height_by_tx_id", srv.GetTxHeight)
	srv.router.POST("/tx_exist_by_tx_id", srv.TxExists)
	srv.router.POST("/get_confirmed_time_by_tx_id", srv.GetTxConfirmedTime)
	srv.router.POST("/get_last_block", srv.GetLastBlock)
	srv.router.POST("/get_filtered_block_by_height", srv.GetFilteredBlock)
	srv.router.POST("/get_last_save_point", srv.GetLastSavepoint)
	srv.router.POST("/get_last_config_block", srv.GetLastConfigBlock)
	srv.router.POST("/get_last_config_block_height", srv.GetLastConfigBlockHeight)
	srv.router.POST("/get_block_by_tx_id", srv.GetBlockByTx)
	srv.router.POST("/get_rwset_by_tx_id", srv.GetTxRWSet)
	srv.router.POST("/get_block_header_by_height", srv.GetBlockHeaderByHeight)
	srv.router.POST("/get_full_block_by_height", srv.GetFullBlockWithRWSetByHeight)
	srv.router.POST("/get_block_info_by_height", srv.GetBlockInfoByHeight)
	srv.router.POST("/get_block_info_by_hash", srv.GetBlockInfoByHash)
	srv.router.POST("/get_block_info_by_txid", srv.GetBlockInfoByTxId)
	srv.router.POST("/get_merklepath_by_txid", srv.GetMerklePathByTxId)
	srv.router.POST("/get_truncate_block_by_height", srv.GetTruncateBlockByHeight)
	srv.router.POST("/get_truncate_tx_by_txid", srv.GetTruncateTxByTxId)
	srv.router.POST("/get_chain_compress", srv.GetChainCompressStatus)
	srv.router.POST("/get_transaction_info_by_txid", srv.GetCommonTxInfo)
	srv.router.POST("/get_full_transaction_info_by_txid", srv.GetCommonTxInfoWithRWSet)
	srv.router.POST("/get_chainconfig_by_height", srv.GetChainConfigByHeight)
	srv.router.POST("/get_hashhex_by_hashbyte", srv.ComputeBlockHashhexByHashByte)
	srv.router.POST("/get_chains_infos", srv.GetChainInfos)

	// 下面两个需要加一下token
	adminGroup := srv.router.Group("/admin", srv.tokenMiddleWare())
	adminGroup.POST("/compress_under_height", srv.CompressUnderHeight)
	adminGroup.POST("/add_ca", srv.AddCA)
}
