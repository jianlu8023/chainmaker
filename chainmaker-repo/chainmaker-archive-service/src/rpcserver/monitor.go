/*
Copyright (C) BABEC. All rights reserved.


SPDX-License-Identifier: Apache-2.0
*/

// Package rpcserver define rpc server
package rpcserver

import (
	"fmt"
	"net"
	"net/http"
	"sync"

	"chainmaker.org/chainmaker-archive-service/src/serverconf"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	mRecv              *prometheus.CounterVec
	mRecvTime          *prometheus.HistogramVec
	grpcPromethus      = "grpc"
	counterVecs        map[string]*prometheus.CounterVec
	histogramVecs      map[string]*prometheus.HistogramVec
	counterVecsMutex   sync.Mutex
	histogramVecsMutex sync.Mutex
	namespace          = "archive"
)

// MonitorServer 监控服务结构
type MonitorServer struct {
	httpServer *http.Server
}

// NewMonitorServer 新建监控服务结构
func NewMonitorServer() *MonitorServer {
	if serverconf.GlobalServerCFG.MonitorCFG.Enabled {
		mux := http.NewServeMux()
		mux.Handle("/metrics", promhttp.Handler())
		return &MonitorServer{
			httpServer: &http.Server{
				Handler: mux,
			},
		}
	}
	return &MonitorServer{}
}

// Start 监控服务开机
func (s *MonitorServer) Start() error {
	if s.httpServer != nil {
		endPoint := fmt.Sprintf(":%d", serverconf.GlobalServerCFG.MonitorCFG.Port)
		conn, err := net.Listen("tcp", endPoint)
		if err != nil {
			return fmt.Errorf("TCP Listen failed , %s", err.Error())
		}
		go func() {
			err = s.httpServer.Serve(conn)
			if err != nil {
				fmt.Printf("Monitor http server failed , %s \n", err.Error())
			}
		}()
		fmt.Printf("Monitor http server listen on %s", endPoint)
	}
	return nil
}

func init() {
	counterVecs = make(map[string]*prometheus.CounterVec)
	histogramVecs = make(map[string]*prometheus.HistogramVec)
}

// NewCounterVec 新建累积统计
func NewCounterVec(subsystem, name, help string, labels ...string) *prometheus.CounterVec {
	counterVecsMutex.Lock()
	defer counterVecsMutex.Unlock()
	s := fmt.Sprintf("%s_%s", subsystem, name)
	if metric, ok := counterVecs[s]; ok {
		return metric
	}
	metric := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      name,
			Help:      help,
		}, labels)
	prometheus.MustRegister(metric)
	counterVecs[s] = metric
	return metric
}

// NewHistogramVec 新建直方图
func NewHistogramVec(subsystem, name, help string,
	buckets []float64, labels ...string) *prometheus.HistogramVec {
	histogramVecsMutex.Lock()
	defer histogramVecsMutex.Unlock()
	s := fmt.Sprintf("%s_%s", subsystem, name)
	if metric, ok := histogramVecs[s]; ok {
		return metric
	}
	metric := prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      name,
			Help:      help,
			Buckets:   buckets,
		}, labels)
	prometheus.MustRegister(metric)
	histogramVecs[s] = metric
	return metric
}
