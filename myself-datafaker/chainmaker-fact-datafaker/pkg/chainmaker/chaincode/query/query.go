package query

import (
	chainmakercommon "chainmaker.org/chainmaker/pb-go/v2/common"

	commonentity "tools/web/entity"

	"tools/response"

	"tools/chain/sync"
	"tools/internal/logger"
)

// CommonQueryChain 通用数据查询
// @param user: 链用户
// @param contractName: 合约名称
// @param methodName: 合约方法名
// @param kvs: 合约方法参数
// @param timeOut: 超时时间
// @param syncResult: 是否同步结果
// @return errCode: 错误码
// @return errMsg: 错误信息
// @return txResponse: 交易响应
func CommonQueryChain(user *commonentity.ChainUser,
	contractName, methodName string, kvs []*chainmakercommon.KeyValuePair,
	timeOut int64, syncResult bool) (errCode response.ErrCode,
	errMsg string, txResponse *chainmakercommon.TxResponse) {

	// 调用 sdk 接口
	// user.ClientKey
	sdkClientPool := sync.GetSdkClientPool()
	if sdkClientPool == nil {
		logger.GetChainCodeLogger().Errorf(">>> 通用-[数据查询]-[链客户端初始化失败, 请重新登录]\n")
		return response.ErrorParamWrong, "链客户端初始化失败, 请重新登录", nil
	}

	sdkClient, ok := sdkClientPool.SdkClients[user.GetClientKey()]
	if !ok {
		logger.GetChainCodeLogger().Errorf(">>> 通用-[数据查询]-[链客户端获取失败, 请重新登录]\n")
		return response.ErrorParamWrong, "链客户端获取失败, 请重新登录", nil
	}

	// 启用证书压缩（开启证书压缩可以减小交易包大小，提升处理性能）
	err := sdkClient.ChainClient.EnableCertHash()
	if err != nil {
		logger.GetChainCodeLogger().Errorf(">>> 通用-[数据查询]-[启用证书压缩失败]-[err:%v]\n", err.Error())
		return response.ErrorParamWrong, "启用证书压缩失败:" + err.Error(), nil
	}

	txResponse, err = sdkClient.ChainClient.QueryContract(contractName, methodName, kvs, timeOut)
	if err != nil {
		logger.GetChainCodeLogger().Errorf(">>> 通用-[数据查询]-[调用智能合约失败]-[err:%v]-[kvs:%v]\n", err.Error(), kvs)
		return response.ErrorHandleFailure, "智能合约执行失败:" + err.Error(), txResponse
	}
	if txResponse.Code != chainmakercommon.TxStatusCode_SUCCESS {
		logger.GetChainCodeLogger().Errorf(">>> 通用-[数据查询]-[调用智能合约失败]-[code:%d]-[msg:%s]-[contractMsg:%v]\n", txResponse.Code, txResponse.Message, txResponse.ContractResult.Message)
		return response.ErrorHandleFailure, "智能合约执行失败:" + txResponse.Message, txResponse
	}

	if syncResult {
		logger.GetChainCodeLogger().Debugf(">>> 通用-[数据查询]-[智能合约执行成功]-resp:[code:%d]-[msg:%s]-[txId:%s]\n", txResponse.Code, txResponse.Message, txResponse.ContractResult.Result)
	} else {
		logger.GetChainCodeLogger().Debugf(">>> 通用-[数据查询]-[智能合约执行成功]-resp:[code:%d]-[msg:%s]-[contractResult:%s]\n", txResponse.Code, txResponse.Message, txResponse.ContractResult)
	}

	// 返回结果
	logger.GetChainCodeLogger().Infof(">>> 通用-[数据查询]-[智能合约执行成功]-[TxId:%s]\n", txResponse.TxId)
	logger.GetChainCodeLogger().Infof(">>> 通用-[数据查询]-[智能合约执行成功]-[txResponse:%v]\n", txResponse)
	logger.GetChainCodeLogger().Infof(">>> 通用-[数据查询]-[智能合约执行成功]-[Code:%v]\n", txResponse.Code)
	logger.GetChainCodeLogger().Infof(">>> 通用-[数据查询]-[智能合约执行成功]-[ContractResult:%v]\n", txResponse.ContractResult)
	if syncResult {
		logger.GetChainCodeLogger().Infof(">>> 通用-[数据查询]-[智能合约执行成功]-[Message:%v]\n", txResponse.ContractResult.Message)
	}
	return response.ErrCodeOK, "", txResponse
}
