/*
Copyright (C) BABEC. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/
package main

import (
	"context"
	"flag"
	"fmt"
	"math/rand"
	"net/http"
	_ "net/http/pprof"
	"os"
	"os/signal"
	"syscall"
	"time"

	"chainmaker.org/chainmaker-archive-service/src/archive_utils"
	"chainmaker.org/chainmaker-archive-service/src/httpserver"

	"chainmaker.org/chainmaker-archive-service/src/process"
	"chainmaker.org/chainmaker-archive-service/src/rpcserver"
	"chainmaker.org/chainmaker-archive-service/src/serverconf"
)

func main() {
	rand.Seed(time.Now().Unix())
	var defaultConfigPath string
	flag.StringVar(&defaultConfigPath, "i", "../configs/config.yml", "config file path")
	flag.Parse()
	configError := serverconf.ReadConfigFile(defaultConfigPath) //初始化配置
	if configError != nil {
		fmt.Fprintf(os.Stderr, "load config file error %s ", configError.Error())
		return
	}
	// 初始化业务处理类
	businessProcessor := process.InitProcessorMgr(&serverconf.GlobalServerCFG.StoreageCFG,
		&serverconf.GlobalServerCFG.LogCFG)
	// new api
	httpSrv := httpserver.NewHttpServer(businessProcessor)
	httpSrv.Listen(serverconf.GlobalServerCFG.HttpCFG)
	// new grpc
	grpcSrv, grpcError := rpcserver.NewRPCServer(businessProcessor, &serverconf.GlobalServerCFG.RpcCFG)
	if grpcError != nil {
		fmt.Fprintf(os.Stderr, "new grpc server got error %s ", grpcError.Error())
		return
	}
	grpcErr := grpcSrv.Start()
	if grpcErr != nil {
		panic("grpc start error , " + grpcErr.Error())
	}
	// 开启prometheus
	monitor := rpcserver.NewMonitorServer()
	monitorErr := monitor.Start()
	if monitorErr != nil {
		panic("monitor start error , " + monitorErr.Error())
	}
	// 开启pprof
	startPProf()
	// 注册信号
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	<-stop
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer func() {
		cancel()
	}()
	_ = httpSrv.ApiSrv.Shutdown(ctx)
	grpcSrv.Stop()
	// custom quit
	archive_utils.GlobalServerLatch.Wait()
	businessProcessor.Close() //业务处理器关闭
	fmt.Fprintf(os.Stderr, "server has shutdown")
}

func startPProf() {
	if !serverconf.GlobalServerCFG.PProfCFG.Enabled {
		return
	}
	go func() {
		addr := fmt.Sprintf(":%d", serverconf.GlobalServerCFG.PProfCFG.Port)
		err := http.ListenAndServe(addr, nil)
		if err != nil {
			panic(err)
		}
	}()
}
