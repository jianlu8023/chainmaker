package block

import (
	"chainmaker-fact-datafaker/internal/logger"
	chainmakercommon "chainmaker.org/chainmaker/pb-go/v2/common"
	chainmakersdkgo "chainmaker.org/chainmaker/sdk-go/v2"
)

// CommonBlockQueryChain 通用区块查询
// @param cc: 链客户端
// @param txId: 交易ID
// @param withRw: 是否返回读写集
// @return blockInfo: 区块信息
// @return err: 错误信息
func CommonBlockQueryChain(cc *chainmakersdkgo.ChainClient, txId string, withRw bool) (blockInfo *chainmakercommon.BlockInfo, err error) {

	// 链上查询
	blockInfo, err = cc.GetBlockByTxId(txId, withRw)
	if err != nil {
		logger.GetAppLogger().Errorf(">>> 通用-[区块查询]-[调用智能合约失败]-[err:%v]-[kvs:%v]\n", err.Error(), map[string]interface{}{"txId": txId, "withRw": withRw})
		return nil, err
	}
	return blockInfo, nil
}
