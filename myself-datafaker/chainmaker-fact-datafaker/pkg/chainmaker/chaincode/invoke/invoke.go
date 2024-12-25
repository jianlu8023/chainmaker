package invoke

import (
	chainmakercommon "chainmaker.org/chainmaker/pb-go/v2/common"

	"tools/response"

	"tools/internal/logger"

	"tools/chain/sync"
)

// CommonUpChain 通用数据上链
// @param contractName: 合约名称
// @param methodName: 合约方法
// @param txId: 交易ID
// @param kvs: 合约参数
// @param timeOut: 超时时间
// @param syncResult: 是否同步
// @return errCode: 错误码
// @return errMsg: 错误信息
// @return txResponse: 交易响应
func CommonUpChain(contractName, methodName, txId string,
	kvs []*chainmakercommon.KeyValuePair,
	timeOut int64, syncResult bool) (txResponse *chainmakercommon.TxResponse) {

	var err error
	// 调用 sdk 接口
	// user.ClientKey
	sdkClientPool := sync.GetSdkClientPool()
	if sdkClientPool == nil {
		logger.GetChainCodeLogger().Errorf(">>> 通用-[数据上链]-[链客户端初始化失败, 请重新登录]\n")
		return nil
	}

	sdkClient, ok := sdkClientPool.SdkClients[user.GetClientKey()]
	if !ok {
		logger.GetChainCodeLogger().Errorf(">>> 通用-[数据上链]-[链客户端获取失败, 请重新登录]\n")
		return response.ErrorParamWrong, "链客户端获取失败, 请重新登录", nil
	}
	// 启用证书压缩（开启证书压缩可以减小交易包大小，提升处理性能）
	err = sdkClient.ChainClient.EnableCertHash()
	if err != nil {
		logger.GetChainCodeLogger().Errorf(">>> 通用-[数据上链]-[启用证书压缩失败]-[err:%v]\n", err.Error())
		return response.ErrorParamWrong, "启用证书压缩失败:" + err.Error(), nil
	}

	// 数据上链
	// txId: 交易ID 格式要求：长度为64字节，字符在a-z0-9, 可为空，若为空字符串，将自动生成txId
	// kvs: 合约参数
	txResponse, err = sdkClient.ChainClient.InvokeContract(contractName, methodName, txId, kvs, timeOut, syncResult)
	if err != nil {
		logger.GetChainCodeLogger().Errorf(">>> 通用-[数据上链]-[调用智能合约失败]-[err:%v]-[kvs:%v]\n", err.Error(), kvs)
		return response.ErrorHandleFailure, "智能合约执行失败:" + err.Error(), txResponse
	}
	if txResponse.Code != chainmakercommon.TxStatusCode_SUCCESS {
		logger.GetChainCodeLogger().Errorf(">>> 通用-[数据上链]-[调用智能合约失败]-[code:%d]-[msg:%s]-[contractMsg:%v]\n", txResponse.Code, txResponse.Message, txResponse.ContractResult.Message)
		return response.ErrorHandleFailure, "智能合约执行失败:" + txResponse.Message, txResponse
	}
	if syncResult {
		logger.GetChainCodeLogger().Debugf(">>> 通用-[数据上链]-[智能合约执行成功]-resp:[code:%d]-[msg:%s]-[txId:%s]\n", txResponse.Code, txResponse.Message, txResponse.ContractResult.Result)
	} else {
		logger.GetChainCodeLogger().Debugf(">>> 通用-[数据上链]-[智能合约执行成功]-resp:[code:%d]-[msg:%s]-[contractResult:%s]\n", txResponse.Code, txResponse.Message, txResponse.ContractResult)
	}

	// 返回结果
	logger.GetChainCodeLogger().Debugf(">>> 通用-[数据上链]-[智能合约执行成功]-[TxId:%s]\n", txResponse.TxId)
	logger.GetChainCodeLogger().Debugf(">>> 通用-[数据上链]-[智能合约执行成功]-[txResponse:%v]\n", txResponse)
	logger.GetChainCodeLogger().Debugf(">>> 通用-[数据上链]-[智能合约执行成功]-[Code:%v]\n", txResponse.Code)
	logger.GetChainCodeLogger().Debugf(">>> 通用-[数据上链]-[智能合约执行成功]-[ContractResult:%v]\n", txResponse.ContractResult)
	if syncResult {
		logger.GetChainCodeLogger().Debugf(">>> 通用-[数据上链]-[智能合约执行成功]-[Message:%v]\n", txResponse.ContractResult.Message)
	}
	return response.ErrCodeOK, "", txResponse
}
