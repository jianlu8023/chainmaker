package archivecenter

import (
	"context"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"chainmaker.org/chainmaker-go/tools/cmc/util"
	"chainmaker.org/chainmaker/pb-go/v2/archivecenter"
	"chainmaker.org/chainmaker/pb-go/v2/common"
	"chainmaker.org/chainmaker/pb-go/v2/store"
	"github.com/hokaccha/go-prettyjson"
	"github.com/spf13/cobra"
)

var (
	errorNoBlock = errors.New("query got no block")
)

func newQueryCMD() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "query",
		Short: "query off-chain blockchain data",
		Long:  "query off-chain blockchain data",
	}
	cmd.AddCommand(newQueryBlockByTxIdCMD())
	cmd.AddCommand(newQueryTxOffChainCMD())
	cmd.AddCommand(newQueryBlockByHashCMD())
	cmd.AddCommand(newQueryArchiveStatus())
	cmd.AddCommand(newQueryBlockByHeightCMD())
	return cmd
}

func newQueryTxOffChainCMD() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "tx [txid]",
		Short: "query off-chain tx by txid",
		Long:  "query off-chain tx by txid",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			archiveClient, err := createArchiveCenerGrpcClient(archiveCenterConfPath)
			if err != nil {
				return err
			}
			defer archiveClient.Stop()
			ctx, ctxCancel := context.WithTimeout(context.Background(),
				time.Duration(grpcTimeoutSeconds)*time.Second)
			defer ctxCancel()
			grpcResp, respErr := archiveClient.client.GetBlockByTxId(ctx,
				&archivecenter.BlockByTxIdRequest{
					ChainUnique: archiveCenterCFG.ChainGenesisHash,
					TxId:        args[0],
				}, archiveClient.GrpcCallOption()...)
			validErr := validGrpcBlock(grpcResp, respErr)
			if validErr != nil {
				if validErr == errorNoBlock {
					fmt.Println("query off-chain data got no block")
					return nil
				}
				return validErr
			}
			outStr, outErr := transferBlockToTransactionString(grpcResp.BlockData, args[0])
			if outErr != nil {
				return fmt.Errorf("marshal transaction error %s", outErr.Error())
			}
			fmt.Println(outStr)
			return nil
		},
	}
	util.AttachAndRequiredFlags(cmd, flags, []string{
		flagArchiveConfPath,
	})
	return cmd
}

func newQueryBlockByHeightCMD() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "block-by-height [height]",
		Short: "query off-chain block by height",
		Long:  "query off-chain block by height",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			height, transferErr := strconv.ParseUint(args[0], 10, 64)
			if transferErr != nil {
				return fmt.Errorf("height must be positive integer , but got %s", args[0])
			}
			archiveClient, err := createArchiveCenerGrpcClient(archiveCenterConfPath)
			if err != nil {
				return err
			}
			defer archiveClient.Stop()
			ctx, ctxCancel := context.WithTimeout(context.Background(),
				time.Duration(grpcTimeoutSeconds)*time.Second)
			defer ctxCancel()
			grpcResp, respErr := archiveClient.client.GetBlockByHeight(ctx,
				&archivecenter.BlockByHeightRequest{
					ChainUnique: archiveCenterCFG.ChainGenesisHash,
					Height:      height,
					Operation:   archivecenter.OperationByHeight_OperationGetBlockByHeight,
				}, archiveClient.GrpcCallOption()...)
			validErr := validGrpcBlock(grpcResp, respErr)
			if validErr != nil {
				if validErr == errorNoBlock {
					fmt.Println("query off-chain data got no block")
					return nil
				}
				return validErr
			}
			blkStr, blkErr := transferBlockToString(grpcResp.BlockData)
			if blkErr != nil {
				return fmt.Errorf("query block-by-height marshal error %s",
					blkErr.Error())
			}
			fmt.Println(blkStr)
			return nil
		},
	}
	util.AttachAndRequiredFlags(cmd, flags, []string{
		flagArchiveConfPath,
	})
	return cmd
}

func newQueryBlockByHashCMD() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "block-by-hash [block hash in hex]",
		Short: "query off-chain block by hash",
		Long:  "query off-chain block by hash",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			hashStr, hashErr := transferHashToHexString(args[0])
			if hashErr != nil {
				return fmt.Errorf("transfer block hash %s got error %s",
					args[0], hashErr.Error())
			}
			archiveClient, err := createArchiveCenerGrpcClient(archiveCenterConfPath)
			if err != nil {
				return err
			}
			defer archiveClient.Stop()
			ctx, ctxCancel := context.WithTimeout(context.Background(),
				time.Duration(grpcTimeoutSeconds)*time.Second)
			defer ctxCancel()
			grpcResp, respErr := archiveClient.client.GetBlockByHash(ctx,
				&archivecenter.BlockByHashRequest{
					ChainUnique: archiveCenterCFG.ChainGenesisHash,
					BlockHash:   hashStr,
					Operation:   archivecenter.OperationByHash_OperationGetBlockByHash,
				}, archiveClient.GrpcCallOption()...)
			validErr := validGrpcBlock(grpcResp, respErr)
			if validErr != nil {
				if validErr == errorNoBlock {
					fmt.Println("query off-chain data got no block")
					return nil
				}
				return validErr
			}
			blkStr, blkErr := transferBlockToString(grpcResp.BlockData)
			if blkErr != nil {
				return fmt.Errorf("query block-by-hash marshal error %s", blkErr.Error())
			}
			fmt.Println(blkStr)
			return nil
		},
	}
	util.AttachAndRequiredFlags(cmd, flags, []string{
		flagArchiveConfPath,
	})
	return cmd
}

func transferHashToHexString(hashStr string) (string, error) {
	if strings.Contains(hashStr, "+") ||
		strings.Contains(hashStr, "=") ||
		strings.Contains(hashStr, "/") {
		hashBytes, decodeErr := base64.StdEncoding.DecodeString(hashStr)
		if decodeErr != nil {
			return hashStr, decodeErr
		}
		return hex.EncodeToString(hashBytes), nil
	}
	return hashStr, nil
}

func newQueryBlockByTxIdCMD() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "block-by-txid [txid]",
		Short: "query off-chain block by txid",
		Long:  "query off-chain block by txid",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			archiveClient, err := createArchiveCenerGrpcClient(archiveCenterConfPath)
			if err != nil {
				return err
			}
			defer archiveClient.Stop()
			ctx, ctxCancel := context.WithTimeout(context.Background(),
				time.Duration(grpcTimeoutSeconds)*time.Second)
			defer ctxCancel()
			grpcResp, respErr := archiveClient.client.GetBlockByTxId(ctx,
				&archivecenter.BlockByTxIdRequest{
					ChainUnique: archiveCenterCFG.ChainGenesisHash,
					TxId:        args[0],
				}, archiveClient.GrpcCallOption()...)
			validErr := validGrpcBlock(grpcResp, respErr)
			if validErr != nil {
				if validErr == errorNoBlock {
					fmt.Println("query off-chain data got no block")
					return nil
				}
				return validErr
			}
			blkStr, blkErr := transferBlockToString(grpcResp.BlockData)
			if blkErr != nil {
				return fmt.Errorf("query block-by-txid marshal error %s",
					blkErr.Error())
			}
			fmt.Println(blkStr)
			return nil
		},
	}
	util.AttachAndRequiredFlags(cmd, flags, []string{
		flagArchiveConfPath,
	})
	return cmd
}

func validGrpcBlock(grpcResp *archivecenter.BlockWithRWSetResp,
	respErr error) error {
	if respErr != nil {
		return fmt.Errorf("validGrpcBlock rpcerror %s",
			respErr.Error())
	}
	if grpcResp == nil {
		return fmt.Errorf("validGrpcBlock rpc nothing")
	}
	if grpcResp.Code > 0 {
		return fmt.Errorf("validGrpcBlock rpc code %d, message %s",
			grpcResp.Code, grpcResp.Message)
	}
	if grpcResp.BlockData == nil ||
		grpcResp.BlockData.Block == nil ||
		grpcResp.BlockData.Block.Header == nil {
		return errorNoBlock
	}
	return nil
}

type OutPutBlockHeader struct {
	BlockHeader *common.BlockHeader `json:"block_header"`
	BlockHash   string              `json:"block_hash"`
}
type OutPutBlock struct {
	Block  *common.Block      `json:"block"`
	Header *OutPutBlockHeader `json:"header"`
}

type OutPutBlockWithRWSet struct {
	BlockWithRWSet *store.BlockWithRWSet
	Block          *OutPutBlock
}

func transferBlockToString(blk *store.BlockWithRWSet) (string, error) {
	block := &OutPutBlockWithRWSet{
		BlockWithRWSet: blk,
		Block: &OutPutBlock{
			Block: blk.Block,
			Header: &OutPutBlockHeader{
				BlockHeader: blk.Block.Header,
				BlockHash:   hex.EncodeToString(blk.Block.Header.BlockHash),
			},
		},
	}
	output, err := prettyjson.Marshal(block)
	if err != nil {
		return "", err
	}
	return string(output), nil
}

func transferBlockToTransactionString(blk *store.BlockWithRWSet,
	txId string) (string, error) {
	retTxInfo := &common.TransactionInfo{
		BlockHeight: blk.Block.Header.BlockHeight,
		BlockHash:   blk.Block.Header.BlockHash,
	}
	for idx, tx := range blk.Block.Txs {
		if tx.Payload.TxId == txId {
			retTxInfo.Transaction = tx
			retTxInfo.TxIndex = uint32(idx)
		}
	}
	out, err := prettyjson.Marshal(retTxInfo)
	if err != nil {
		return "", err
	}
	return string(out), nil
}

func newQueryArchiveStatus() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "archived-status",
		Short: "query off-chain archived status",
		Long:  "query off-chain archived status",
		Args:  cobra.ExactArgs(0),
		RunE: func(cmd *cobra.Command, args []string) error {
			archiveClient, err := createArchiveCenerGrpcClient(archiveCenterConfPath)
			if err != nil {
				return err
			}
			defer archiveClient.Stop()
			ctx, ctxCancel := context.WithTimeout(context.Background(),
				time.Duration(grpcTimeoutSeconds)*time.Second)
			defer ctxCancel()
			grpcResp, respErr := archiveClient.client.GetArchivedStatus(ctx,
				&archivecenter.ArchiveStatusRequest{
					ChainUnique: archiveCenterCFG.ChainGenesisHash,
				})
			if respErr != nil {
				return fmt.Errorf("archived-status rpc error %s", respErr.Error())
			}
			if grpcResp == nil {
				return fmt.Errorf("archived-status rpc nothing")
			}
			if grpcResp.Code != 0 {
				return fmt.Errorf("archived-status rpc code %d ,message %s ",
					grpcResp.Code, grpcResp.Message)
			}
			fmt.Printf("archived-status height %d , inArchive %t \n",
				grpcResp.ArchivedHeight, grpcResp.InArchive)
			return nil
		},
	}
	util.AttachAndRequiredFlags(cmd, flags, []string{
		flagArchiveConfPath,
	})
	return cmd
}
