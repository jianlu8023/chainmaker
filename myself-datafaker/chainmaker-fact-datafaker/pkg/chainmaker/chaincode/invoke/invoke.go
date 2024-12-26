package invoke

import (
	"fmt"

	chainmakercommon "chainmaker.org/chainmaker/pb-go/v2/common"
	chainmakersdkgo "chainmaker.org/chainmaker/sdk-go/v2"

	"chainmaker-fact-datafaker/internal/logger"
)

// CommonUpChain 通用数据上链
// @param cc: 链客户端
// @param contractName: 合约名称
// @param methodName: 合约方法
// @param txId: 交易ID
// @param kvs: 合约参数
// @param timeOut: 超时时间
// @param syncResult: 是否同步
// @return txResponse: 交易响应
// @return err: 错误信息
func CommonUpChain(cc *chainmakersdkgo.ChainClient, contractName, methodName, txId string,
	kvs []*chainmakercommon.KeyValuePair,
	timeOut int64, syncResult bool) (txResponse *chainmakercommon.TxResponse, err error) {

	// 数据上链
	// txId: 交易ID 格式要求：长度为64字节，字符在a-z0-9, 可为空，若为空字符串，将自动生成txId
	// kvs: 合约参数
	txResponse, err = cc.InvokeContract(contractName, methodName, txId, kvs, timeOut, syncResult)
	if err != nil {
		logger.GetAppLogger().Errorf(">>> 通用-[数据上链]-[调用智能合约失败]-[err:%v]-[kvs:%v]\n", err.Error(), kvs)
		return nil, err
	}
	if txResponse.Code != chainmakercommon.TxStatusCode_SUCCESS {
		logger.GetAppLogger().Errorf(">>> 通用-[数据上链]-[调用智能合约失败]-[code:%d]-[msg:%s]-[contractMsg:%v]\n", txResponse.Code, txResponse.Message, txResponse.ContractResult.Message)
		return nil, fmt.Errorf("txResponse code != TxStatusCode_SUCCESS")
	}
	if syncResult {
		logger.GetAppLogger().Debugf(">>> 通用-[数据上链]-[智能合约执行成功]-resp:[code:%d]-[msg:%s]-[txId:%s]\n", txResponse.Code, txResponse.Message, txResponse.ContractResult.Result)
	} else {
		logger.GetAppLogger().Debugf(">>> 通用-[数据上链]-[智能合约执行成功]-resp:[code:%d]-[msg:%s]-[contractResult:%s]\n", txResponse.Code, txResponse.Message, txResponse.ContractResult)
	}

	// 返回结果
	logger.GetAppLogger().Debugf(">>> 通用-[数据上链]-[智能合约执行成功]-[TxId:%s]\n", txResponse.TxId)
	logger.GetAppLogger().Debugf(">>> 通用-[数据上链]-[智能合约执行成功]-[txResponse:%v]\n", txResponse)
	logger.GetAppLogger().Debugf(">>> 通用-[数据上链]-[智能合约执行成功]-[Code:%v]\n", txResponse.Code)
	logger.GetAppLogger().Debugf(">>> 通用-[数据上链]-[智能合约执行成功]-[ContractResult:%v]\n", txResponse.ContractResult)
	if syncResult {
		logger.GetAppLogger().Debugf(">>> 通用-[数据上链]-[智能合约执行成功]-[Message:%v]\n", txResponse.ContractResult.Message)
	}
	return txResponse, nil
}
