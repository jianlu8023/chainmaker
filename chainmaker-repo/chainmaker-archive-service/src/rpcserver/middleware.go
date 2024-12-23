/*
Copyright (C) BABEC. All rights reserved.


SPDX-License-Identifier: Apache-2.0
*/

// Package rpcserver define rpc server
package rpcserver

import (
	"context"
	"fmt"
	"net"
	"os"
	"runtime/debug"
	"strings"
	"sync"
	"time"

	"chainmaker.org/chainmaker-archive-service/src/serverconf"
	"golang.org/x/time/rate"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"
)

const (
	// rpc ratelimit config
	rateLimitDefaultTokenPerSecond  = 10000
	rateLimitDefaultTokenBucketSize = 10000
	// UNKNOWN 未知错误
	UNKNOWN                 = "unknown"
	rateLimitGrpcTypeGlobal = 0
)

// RecoveryInterceptor recovery 防崩溃
func RecoveryInterceptor(ctx context.Context,
	req interface{}, _ *grpc.UnaryServerInfo,
	handler grpc.UnaryHandler) (resp interface{}, err error) {
	defer func() {
		if e := recover(); e != nil {
			stack := debug.Stack()
			os.Stderr.Write(stack)
			err = status.Errorf(codes.Internal, "panic error: %v", e)
		}
	}()
	return handler(ctx, req)
}

// getReateLimitBucket 根据ip计算令牌筒
func getReateLimitBucket(bucketMap *sync.Map, rateLimitType, tokenBucketSize,
	tokenPerSecond int, peerIpAddr string) *rate.Limiter {
	var (
		bucket interface{}
		ok     bool
	)
	if rateLimitType == rateLimitGrpcTypeGlobal {
		if bucket, ok = bucketMap.Load(rateLimitGrpcTypeGlobal); ok {
			return bucket.(*rate.Limiter)
		}
	} else {
		if bucket, ok = bucketMap.Load(peerIpAddr); ok {
			return bucket.(*rate.Limiter)
		}
	}

	if tokenBucketSize >= 0 && tokenPerSecond >= 0 {
		if tokenBucketSize == 0 {
			tokenBucketSize = rateLimitDefaultTokenBucketSize
		}
		if tokenPerSecond == 0 {
			tokenPerSecond = rateLimitDefaultTokenPerSecond
		}
		bucket = rate.NewLimiter(rate.Limit(tokenPerSecond), tokenBucketSize)
	} else {
		return nil
	}
	if rateLimitType == rateLimitGrpcTypeGlobal {
		bucket, _ = bucketMap.LoadOrStore(rateLimitGrpcTypeGlobal, bucket)
	} else {
		bucketMap.LoadOrStore(peerIpAddr, bucket)
	}
	return bucket.(*rate.Limiter)
}

// RateLimitInterceptor 限速中间件
func RateLimitInterceptor() grpc.UnaryServerInterceptor {
	rpcCFG := serverconf.GlobalServerCFG.RpcCFG
	tokenBucketSize := rpcCFG.RateLimitConfig.TokenBucketSize
	tokenPerSecond := rpcCFG.RateLimitConfig.TokenPerSecond
	rateLimitType := rpcCFG.RateLimitConfig.Type
	bucketMap := sync.Map{}
	return func(ctx context.Context, req interface{},
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler) (resp interface{}, err error) {
		ipAddr := getClientIp(ctx)
		bucket := getReateLimitBucket(&bucketMap, rateLimitType,
			tokenBucketSize, tokenPerSecond, ipAddr)
		if bucket != nil && !bucket.Allow() {
			errMsg := fmt.Sprintf("%s is rejected by ratelimit , try later", info.FullMethod)
			return nil, status.Error(codes.ResourceExhausted, errMsg)
		}
		return handler(ctx, req)
	}
}

// GetClientAddr 获取客户端ip
func GetClientAddr(ctx context.Context) string {
	pr, ok := peer.FromContext(ctx)
	if !ok {
		return ""
	}
	if pr.Addr == net.Addr(nil) {
		return ""
	}
	return pr.Addr.String()
}

func getClientIp(ctx context.Context) string {
	addr := GetClientAddr(ctx)
	return strings.Split(addr, ":")[0]
}

// WhiteListInterceptor 白名单拦截器
func WhiteListInterceptor() grpc.UnaryServerInterceptor {
	if !serverconf.GlobalServerCFG.RpcCFG.WhiteListConfig.Enabled {
		return func(ctx context.Context, req interface{},
			info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp interface{}, err error) {
			return handler(ctx, req)
		}
	}
	whiteList := serverconf.GlobalServerCFG.RpcCFG.WhiteListConfig.Address
	whiteMp := make(map[string]struct{})
	for i := 0; i < len(whiteList); i++ {
		whiteMp[strings.TrimSpace(whiteList[i])] = struct{}{}
	}
	return func(ctx context.Context, req interface{},
		info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp interface{}, err error) {
		ipAddr := getClientIp(ctx)
		_, ok := whiteMp[ipAddr]
		if !ok {
			fmt.Fprintf(os.Stderr, "ip %s is not in white list ", ipAddr)
			return nil, status.Error(codes.ResourceExhausted, "ip not in white list")
		}
		return handler(ctx, req)
	}
}

func splitMethodName(fullMethodName string) (string, string) {
	fullMethodName = strings.TrimPrefix(fullMethodName, "/") // remove leading slash
	if i := strings.Index(fullMethodName, "/"); i >= 0 {
		return fullMethodName[:i], fullMethodName[i+1:]
	}
	return UNKNOWN, UNKNOWN
}

// MonitorInterceptor - set monitor interceptor
func MonitorInterceptor(ctx context.Context, req interface{},
	info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {

	service, method := splitMethodName(info.FullMethod)
	mRecv.WithLabelValues(service, method).Inc()

	start := time.Now()
	resp, err := handler(ctx, req)
	elapsed := time.Since(start)

	mRecvTime.WithLabelValues(service, method).Observe(elapsed.Seconds())

	return resp, err
}
