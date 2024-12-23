package archivecenter

import (
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

var (
	sdkConfPath           string // sdk-go的配置信息
	archiveCenterConfPath string // 归档中心文件配置路径
	chainId               string // chainid
	archiveBeginHeight    uint64 // 开始归档区块
	archiveEndHeight      uint64 // 结束归档区块
	archiveMode           string // 是否使用快速归档(是的话为quick默认为否)
	flags                 *pflag.FlagSet
)

const (
	flagSdkConfPath        = "sdk-conf-path"
	flagArchiveConfPath    = "archive-conf-path"
	flagArchiveBeginHeight = "archive-begin-height"
	flagArchiveEndHeight   = "archive-end-height"
	flagChainId            = "chain-id"
	flagArchiveMode        = "mode"
	archiveModeQuick       = "quick"
	archivedModeNormal     = "normal"
)

func NewArchiveCenterCMD() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "archivecenter",
		Short: "archivecenter blockchain data",
		Long:  "archivecener blockchain data ",
	}
	cmd.AddCommand(newDumpCMD())
	cmd.AddCommand(newQueryCMD())
	return cmd
}

func init() {
	flags = &pflag.FlagSet{}
	flags.StringVar(&sdkConfPath,
		flagSdkConfPath, "", "specify sdk config path")
	flags.StringVar(&archiveCenterConfPath,
		flagArchiveConfPath, "", "specify archivecenter config path")
	flags.StringVar(&chainId, flagChainId, "", "specify chain id ")
	flags.Uint64Var(&archiveEndHeight, flagArchiveEndHeight,
		0, "specify archive end block height")
	flags.Uint64Var(&archiveBeginHeight, flagArchiveBeginHeight,
		0, "specify archive begin block height")
	flags.StringVar(&archiveMode, flagArchiveMode, "", "specify archive mode ,can be quick or normal ")
}
