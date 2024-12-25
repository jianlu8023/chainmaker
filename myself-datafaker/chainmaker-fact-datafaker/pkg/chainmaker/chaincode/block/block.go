package block

import (
	chainmakercommon "chainmaker.org/chainmaker/pb-go/v2/common"

	commonentity "tools/web/entity"

	"tools/response"

	"tools/chain/sync"
	"tools/internal/logger"
)

// CommonBlockQueryChain 通用区块查询
// @param user: 链用户
// @param txId: 交易ID
// @param withRw: 是否返回读写集
// @return errCode: 错误码
// @return errMsg: 错误信息
// @return blockInfo: 区块信息
func CommonBlockQueryChain(user *commonentity.ChainUser,
	txId string, withRw bool) (errCode response.ErrCode,
	errMsg string, blockInfo *chainmakercommon.BlockInfo) {
	// 调用 sdk 接口
	// user.ClientKey
	sdkClientPool := sync.GetSdkClientPool()
	if sdkClientPool == nil {
		logger.GetBlockLogger().Errorf(">>> 通用-[区块查询]-[链客户端初始化失败, 请重新登录]\n")
		return response.ErrorParamWrong, "链客户端初始化失败, 请重新登录", nil
	}

	sdkClient, ok := sdkClientPool.SdkClients[user.GetClientKey()]
	if !ok {
		logger.GetBlockLogger().Errorf(">>> 通用-[区块查询]-[链客户端获取失败, 请重新登录]\n")
		return response.ErrorParamWrong, "链客户端获取失败, 请重新登录", nil
	}

	// 启用证书压缩（开启证书压缩可以减小交易包大小，提升处理性能）
	err := sdkClient.ChainClient.EnableCertHash()
	if err != nil {
		logger.GetBlockLogger().Errorf(">>> 通用-[区块查询]-[启用证书压缩失败]-[err:%v]\n", err.Error())
		return response.ErrorHandleFailure, "启用证书压缩失败:" + err.Error(), nil
	}

	// 链上查询
	blockInfo, err = sdkClient.ChainClient.GetBlockByTxId(txId, withRw)
	if err != nil {
		logger.GetBlockLogger().Errorf(">>> 通用-[区块查询]-[调用智能合约失败]-[err:%v]-[kvs:%v]\n", err.Error(), map[string]interface{}{"txId": txId, "withRw": withRw})
		return response.ErrorHandleFailure, "查询失败:" + err.Error(), blockInfo
	}
	return response.ErrCodeOK, "", blockInfo
}
