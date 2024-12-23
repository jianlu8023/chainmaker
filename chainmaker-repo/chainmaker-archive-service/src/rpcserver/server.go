/*
Copyright (C) BABEC. All rights reserved.


SPDX-License-Identifier: Apache-2.0
*/

// Package rpcserver define rpc server
package rpcserver

import (
	"context"
	"crypto/tls"
	"fmt"
	"io/ioutil"
	"net"
	"time"

	"crypto/x509"

	"chainmaker.org/chainmaker-archive-service/src/logger"
	"chainmaker.org/chainmaker-archive-service/src/process"
	"chainmaker.org/chainmaker-archive-service/src/serverconf"
	archivePb "chainmaker.org/chainmaker/pb-go/v2/archivecenter"
	grpc_middleware "github.com/grpc-ecosystem/go-grpc-middleware"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/keepalive"
)

// RpcServer rpc服务结构
type RpcServer struct {
	ProxyProcessor *process.ProcessorMgr
	rpcConfig      *serverconf.RpcConfig
	grpcServer     *grpc.Server
	ctx            context.Context
	cancel         context.CancelFunc
	isShutdown     bool
	logger         *logger.WrappedLogger
}

// NewRPCServer 新建rpc服务
func NewRPCServer(proxy *process.ProcessorMgr, rpcConfig *serverconf.RpcConfig) (*RpcServer, error) {
	grpcServer, grpcServerError := newGrpcServer(proxy, rpcConfig)
	if grpcServerError != nil {
		return nil, grpcServerError
	}
	grpcLog := logger.NewLogger("GRPCSERVER", &serverconf.LogConfig{
		LogPath:      fmt.Sprintf("%s/grpcserver.log", serverconf.GlobalServerCFG.LogCFG.LogPath),
		LogLevel:     serverconf.GlobalServerCFG.LogCFG.LogLevel,
		LogInConsole: serverconf.GlobalServerCFG.LogCFG.LogInConsole,
		ShowColor:    serverconf.GlobalServerCFG.LogCFG.ShowColor,
		MaxSize:      serverconf.GlobalServerCFG.LogCFG.MaxSize,
		MaxBackups:   serverconf.GlobalServerCFG.LogCFG.MaxBackups,
		MaxAge:       serverconf.GlobalServerCFG.LogCFG.MaxAge,
		Compress:     serverconf.GlobalServerCFG.LogCFG.Compress,
	})
	retServer := &RpcServer{
		grpcServer:     grpcServer,
		ProxyProcessor: proxy,
		rpcConfig:      rpcConfig,
		logger:         grpcLog,
	}
	go func() {
		retServer.WaitCAUpdateSignalAndRestartRpc()
	}()
	archivePb.RegisterArchiveCenterServerServer(retServer.grpcServer, retServer)
	return retServer, nil
}

// WaitCAUpdateSignalAndRestartRpc 接收到ca更新信号,重启rpc
func (s *RpcServer) WaitCAUpdateSignalAndRestartRpc() {
	// 如果未开启tls，直接返回就可以了
	if !s.rpcConfig.TLSEnable {
		return
	}
	s.logger.Info("rpc server begin wait update ca signal")
	sigs := s.ProxyProcessor.ReceiveUpdateCASignal()
	for range sigs {
		s.logger.Info("rpc server got ca add signal , restart grpc service ")
		_ = s.Restart()
	}
}

// Start rpc服务开机
func (s *RpcServer) Start() error {
	var (
		err error
	)
	s.ctx, s.cancel = context.WithCancel(context.Background())
	s.isShutdown = false
	endPoint := fmt.Sprintf(":%d", serverconf.GlobalServerCFG.RpcCFG.Port)
	conn, err := net.Listen("tcp", endPoint)
	if err != nil {
		return fmt.Errorf("TCP listen failed , %s", err.Error())
	}
	go func() {
		err = s.grpcServer.Serve(conn)
		if err != nil {
			s.logger.Errorf("rpc server connection got error %s ", err.Error())
		}
	}()
	return nil
}

func (s *RpcServer) stopGrpcServer() {
	stopped := make(chan struct{})
	go func() {
		s.grpcServer.GracefulStop()
		close(stopped)
	}()
	t := time.NewTimer(10 * time.Second)
	defer t.Stop()
	select {
	case <-t.C:
		s.grpcServer.Stop()
	case <-stopped:

	}
}

// Stop rpc服务关机
func (s *RpcServer) Stop() {
	s.isShutdown = true
	s.cancel()
	s.stopGrpcServer()
	s.logger.Info("rpc server shutdown")
}

func newGrpcServer(proxy *process.ProcessorMgr, rpcConfig *serverconf.RpcConfig) (*grpc.Server, error) {
	var opts []grpc.ServerOption
	if serverconf.GlobalServerCFG.MonitorCFG.Enabled {
		opts = []grpc.ServerOption{
			grpc_middleware.WithUnaryServerChain(RecoveryInterceptor,
				MonitorInterceptor, WhiteListInterceptor(), RateLimitInterceptor()),
		}
		mRecv = NewCounterVec(grpcPromethus, "grpc_msg_received_total",
			"Total number of RPC messages received on the server.",
			"grpc_service", "grpc_method")
		mRecvTime = NewHistogramVec(grpcPromethus, "grpc_msg_received_time",
			"The time of RPC messages received on the server.",
			[]float64{0.005, 0.01, 0.015, 0.05, 0.1, 1, 10},
			"grpc_service", "grpc_method")
	} else {
		opts = []grpc.ServerOption{
			grpc_middleware.WithUnaryServerChain(RecoveryInterceptor, WhiteListInterceptor(), RateLimitInterceptor()),
		}
	}

	// 增加容量限制
	opts = append(opts, grpc.MaxSendMsgSize(serverconf.GlobalServerCFG.RpcCFG.MaxSendMsgSize*1024*1024))
	opts = append(opts, grpc.MaxRecvMsgSize(serverconf.GlobalServerCFG.RpcCFG.MaxRecvMsgSize*1024*1024))
	if rpcConfig.TLSEnable {
		// 如果开启了grpc的tls选项，加一下tls信息
		opts = appendTlsConfig(opts, proxy, rpcConfig)
	}
	// keep alive
	var kaep = keepalive.EnforcementPolicy{
		MinTime:             2 * time.Second,
		PermitWithoutStream: true,
	}
	var kasp = keepalive.ServerParameters{
		Time:    5 * time.Second,
		Timeout: 1 * time.Second,
	}
	opts = append(opts,
		grpc.KeepaliveEnforcementPolicy(kaep), grpc.KeepaliveParams(kasp))
	server := grpc.NewServer(opts...)
	return server, nil
}

// loadCAFromLocalFile 读取目录下的CA文件
func loadCAFromLocalFile(filePaths []string) ([][]byte, error) {
	var retList [][]byte
	for i := 0; i < len(filePaths); i++ {
		tempCA, tempCAError := ioutil.ReadFile(filePaths[i])
		if tempCAError != nil {
			return retList, tempCAError
		}
		if len(tempCA) > 0 {
			retList = append(retList, tempCA)
		}
	}
	return retList, nil
}

// appendTlsConfig 增加grpc的tls opts
func appendTlsConfig(opts []grpc.ServerOption, proxy *process.ProcessorMgr,
	rpcConfig *serverconf.RpcConfig) []grpc.ServerOption {
	caArray, caArrayError := proxy.LoadCAFromKV()
	if caArrayError != nil {
		panic(fmt.Sprintf("load ca from kv error (%s)", caArrayError.Error()))
	}
	if len(caArray) == 0 {
		// 从文件中加载ca,说明是第一次启动
		fileCAList, fileError := loadCAFromLocalFile(rpcConfig.TLSConfig.TrustCaList)
		if fileError != nil {
			panic(fmt.Sprintf("load ca from file error (%s)", fileError.Error()))
		}
		saveError := proxy.BatchSaveCAInKV(fileCAList)
		if saveError != nil {
			panic(fmt.Sprintf("save ca in kv error (%s)", saveError.Error()))
		}
		caArray = fileCAList
	}
	if len(caArray) > 0 {
		cert, certErr := tls.LoadX509KeyPair(rpcConfig.TLSConfig.CertFile, rpcConfig.TLSConfig.PrivKeyFile)
		if certErr != nil {
			panic(fmt.Sprintf("server load cert,key file error (%s)", certErr.Error()))
		}
		certPool := x509.NewCertPool()
		for i := 0; i < len(caArray); i++ {
			if !certPool.AppendCertsFromPEM(caArray[i]) {
				panic(fmt.Sprintf("append ca file(%d) failed", i))
			}
		}
		//nolint
		opts = append(opts, grpc.Creds(credentials.NewTLS(&tls.Config{
			ClientAuth:   tls.RequireAndVerifyClientCert,
			Certificates: []tls.Certificate{cert},
			ClientCAs:    certPool,
		})))
	}
	return opts
}

// Restart 重启rpc
func (s *RpcServer) Restart() error {
	var (
		err error
	)
	s.cancel()
	s.stopGrpcServer()
	grpcServer, grpcServerError := newGrpcServer(s.ProxyProcessor, s.rpcConfig)
	if grpcServerError != nil {
		return grpcServerError
	}
	s.grpcServer = grpcServer
	archivePb.RegisterArchiveCenterServerServer(s.grpcServer, s)
	if err = s.Start(); err != nil {
		s.logger.Errorf("rpc server restart got error %s ", err.Error())
		return err
	}
	s.logger.Info("rpc server restart success")
	return nil
}
