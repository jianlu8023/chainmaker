package query

import (
	chainmakercommon "chainmaker.org/chainmaker/pb-go/v2/common"
	chainmakersdkgo "chainmaker.org/chainmaker/sdk-go/v2"

	"chainmaker-fact-datafaker/internal/logger"
)

// CommonQueryChain 通用数据查询
// @param cc: 链客户端
// @param contractName: 合约名称
// @param methodName: 合约方法名
// @param kvs: 合约方法参数
// @param timeOut: 超时时间
// @param syncResult: 是否同步结果
// @return txResponse: 交易响应
// @return err: 错误信息
func CommonQueryChain(cc *chainmakersdkgo.ChainClient,
	contractName, methodName string, kvs []*chainmakercommon.KeyValuePair,
	timeOut int64, syncResult bool) (txResponse *chainmakercommon.TxResponse, err error) {

	txResponse, err = cc.QueryContract(contractName, methodName, kvs, timeOut)
	if err != nil {
		logger.GetAppLogger().Errorf(">>> 通用-[数据查询]-[调用智能合约失败]-[err:%v]-[kvs:%v]\n", err.Error(), kvs)
		return nil, err
	}
	if txResponse.Code != chainmakercommon.TxStatusCode_SUCCESS {
		logger.GetAppLogger().Errorf(">>> 通用-[数据查询]-[调用智能合约失败]-[code:%d]-[msg:%s]-[contractMsg:%v]\n", txResponse.Code, txResponse.Message, txResponse.ContractResult.Message)
		return nil, err
	}

	if syncResult {
		logger.GetAppLogger().Debugf(">>> 通用-[数据查询]-[智能合约执行成功]-resp:[code:%d]-[msg:%s]-[txId:%s]\n", txResponse.Code, txResponse.Message, txResponse.ContractResult.Result)
	} else {
		logger.GetAppLogger().Debugf(">>> 通用-[数据查询]-[智能合约执行成功]-resp:[code:%d]-[msg:%s]-[contractResult:%s]\n", txResponse.Code, txResponse.Message, txResponse.ContractResult)
	}

	// 返回结果
	logger.GetAppLogger().Infof(">>> 通用-[数据查询]-[智能合约执行成功]-[TxId:%s]\n", txResponse.TxId)
	logger.GetAppLogger().Infof(">>> 通用-[数据查询]-[智能合约执行成功]-[txResponse:%v]\n", txResponse)
	logger.GetAppLogger().Infof(">>> 通用-[数据查询]-[智能合约执行成功]-[Code:%v]\n", txResponse.Code)
	logger.GetAppLogger().Infof(">>> 通用-[数据查询]-[智能合约执行成功]-[ContractResult:%v]\n", txResponse.ContractResult)
	if syncResult {
		logger.GetAppLogger().Infof(">>> 通用-[数据查询]-[智能合约执行成功]-[Message:%v]\n", txResponse.ContractResult.Message)
	}
	return txResponse, nil
}
