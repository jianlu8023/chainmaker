package main

import (
	"fmt"
	"os"

	"archive-center/thridparty/bfdb"
)

func main() {

	// 1. 获取bfdb需要的配置文件
	bfdbConfPath := "./configs/cleanbfdb/config.yml"

	configError := bfdb.ReadConfigFile(bfdbConfPath)
	if configError != nil {
		fmt.Fprintf(os.Stderr, "load config file error %s ", configError.Error())
		return
	}

	// 2. 初始化logger
	bfdb.InitLogger(&bfdb.GlobalServerCFG.LogCFG)

	// 3. 执行清理操作
	if bfdb.GlobalServerCFG.Mode == 1 {
		// 只扫描文件
		fmt.Println("begin scan files")
		bfdb.RunScanFilesNoBfdb(bfdb.GlobalServerCFG.CleanHeight, bfdb.GlobalServerCFG.BfdbPath)
	} else {
		// 扫描文件并清理
		bfdb.RunCleanFiles(bfdb.GlobalServerCFG.BfdbPath, bfdb.GlobalServerCFG.CleanStub)
		fmt.Println("begin clean files")
	}
}
