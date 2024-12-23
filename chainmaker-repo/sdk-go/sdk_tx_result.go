/*
Copyright (C) BABEC. All rights reserved.
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package chainmaker_sdk_go

import (
	"context"
	"errors"
	"sync"
	"time"

	"chainmaker.org/chainmaker/pb-go/v2/common"
)

type txResult struct {
	Result        *common.Result
	TxTimestamp   int64
	TxBlockHeight uint64
}

type txResultDispatcher struct {
	startBlock int64
	cc         *ChainClient
	stopC      chan struct{}
	txC        chan *transaction

	mux             sync.Mutex // mux protect txRegistrations
	txRegistrations map[string]chan *txResult
}

func newTxResultDispatcher(cc *ChainClient) *txResultDispatcher {
	return &txResultDispatcher{
		startBlock:      -1,
		cc:              cc,
		stopC:           make(chan struct{}),
		txC:             make(chan *transaction, 1),
		txRegistrations: make(map[string]chan *txResult),
	}
}

// register registers for transaction result events.
// Note that unregister must be called when the registration is no longer needed.
//   - txId is the transaction ID for which events are to be received
//   - Returns the channel that is used to receive result. The channel
//     is closed when unregister is called.
func (d *txResultDispatcher) register(txId string) chan *txResult {
	d.mux.Lock()
	defer d.mux.Unlock()
	if txResultC, exists := d.txRegistrations[txId]; exists {
		return txResultC
	}
	txResultC := make(chan *txResult, 1)
	d.txRegistrations[txId] = txResultC
	return txResultC
}

// unregister removes the given registration and closes the event channel.
func (d *txResultDispatcher) unregister(txId string) {
	d.mux.Lock()
	defer d.mux.Unlock()
	if txResultC, exists := d.txRegistrations[txId]; exists {
		delete(d.txRegistrations, txId)
		close(txResultC)
	}
}

func (d *txResultDispatcher) start() {
	go d.autoSubscribe()
	for {
		select {
		case tx := <-d.txC:
			d.mux.Lock()
			if txResultC, exists := d.txRegistrations[tx.Transaction.Payload.TxId]; exists {
				// non-blocking write to channel to ignore txResultC buffer is full in extreme cases
				result := &txResult{
					Result:        tx.Transaction.Result,
					TxTimestamp:   tx.Transaction.Payload.Timestamp,
					TxBlockHeight: tx.BlockHeight,
				}
				select {
				case txResultC <- result:
				default:
				}
			}
			d.mux.Unlock()
		case <-d.stopC:
			return
		}
	}
}

func (d *txResultDispatcher) stop() {
	close(d.stopC)
}

type transaction struct {
	Transaction *common.Transaction
	BlockHeight uint64
}

func (d *txResultDispatcher) subscribe() error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	dataC, err := d.cc.SubscribeBlock(ctx, d.startBlock, -1, false, false)
	if err != nil {
		return err
	}
	d.cc.logger.Debugf("txResultDispatcher subscribe success, start block[%d]", d.startBlock)

	for {
		select {
		case block, ok := <-dataC:
			if !ok {
				return errors.New("chan is closed")
			}

			blockInfo, _ := block.(*common.BlockInfo)
			d.cc.logger.Debugf("received block height: %d tx count: %d",
				blockInfo.Block.Header.BlockHeight, len(blockInfo.Block.Txs))
			for _, tx := range blockInfo.Block.Txs {
				d.txC <- &transaction{
					Transaction: tx,
					BlockHeight: blockInfo.Block.Header.BlockHeight,
				}
			}
			d.startBlock = int64(blockInfo.Block.Header.BlockHeight)
		case <-d.stopC:
			return nil
		}
	}
}

func (d *txResultDispatcher) autoSubscribe() {
	for {
		if err := d.subscribe(); err != nil {
			d.cc.logger.Debugf("txResultDispatcher subscribe failed, %s", err)
			d.cc.logger.Debug("txResultDispatcher will resubscribing after one second")
			time.Sleep(time.Second)
		} else {
			d.cc.logger.Debug("txResultDispatcher subscribe stopped")
			return
		}
	}
}
