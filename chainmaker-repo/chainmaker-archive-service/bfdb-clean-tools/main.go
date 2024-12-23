/*
Copyright (C) BABEC. All rights reserved.


SPDX-License-Identifier: Apache-2.0
*/

package main

import (
	"flag"
	"fmt"
	"os"
	"time"
)

func main() {
	var defaultConfigPath string
	flag.StringVar(&defaultConfigPath, "i", "./config.yml", "config file path")
	flag.Parse()
	configError := ReadConfigFile(defaultConfigPath)
	if configError != nil {
		fmt.Fprintf(os.Stderr, "load config file error %s ", configError.Error())
		return
	}
	InitLogger(&GlobalServerCFG.LogCFG)
	if GlobalServerCFG.Mode == 1 {
		fmt.Println("begin scan files")
		runScanFilesNoBfdb(GlobalServerCFG.CleanHeight, GlobalServerCFG.BfdbPath)
	} else {
		runCleanFiles(GlobalServerCFG.BfdbPath, GlobalServerCFG.CleanStub)
		fmt.Println("begin clean files")
	}
	fmt.Printf("job done ! \n")

}

// runCleanFiles 删除文件
// @param pathDir
// @param cleanStub
func runCleanFiles(pathDir string, cleanStub string) {
	oFile, oReader := openCSVFile(cleanStub)
	defer oFile.Close()
	summaryF, summaryFErr := os.Create(fmt.Sprintf("%s.result", cleanStub))
	if summaryFErr != nil {
		panic("runCleanFiles create error " + summaryFErr.Error())
	}
	defer summaryF.Close()
	for {
		record := readFileRecord(oReader)
		if record == nil {
			break
		}
		if record.CanClean == "y" {
			// todo rm file
			delErr := os.Remove(fmt.Sprintf("%s/%s", pathDir, record.FileName))
			if delErr != nil {
				fmt.Printf("delete file %s got error %s ", record.FileName, delErr.Error())
				continue
			}
			fmt.Fprintln(summaryF, record.FileName)
			_ = summaryF.Sync()
		}

	}
	_ = summaryF.Sync()
}

// runScanFilesNoBfdb 根据块高度扫描区块文件
// @param height
// @param pathDir
func runScanFilesNoBfdb(height uint64, pathDir string) {
	archiveStatus, queryErr := getArchiveStatus()
	if queryErr != nil {
		msg := fmt.Sprintf("runScanFilesNoBfdb getArchiveStatus error [%+v]",
			queryErr)
		fmt.Println(msg)
		return
	}
	if archiveStatus < height {
		msg := fmt.Sprintf("archiveCenter max height [%d] < clean_height [%d] ,please modify clean_height",
			archiveStatus, height)
		fmt.Println(msg)
		return
	}
	// first get all config height
	configHeights, configHeightsErr := getAllConfigsUnderHeight(height)
	if configHeightsErr != nil {
		GlobalLogger.Errorf("runScanFilesNoBfdb getAllConfigsUnderHeight [%d] error %s",
			height, configHeightsErr.Error())
		panic(fmt.Sprintf("runScanFilesNoBfdb getAllConfigsUnderHeight [%d] error %s",
			height, configHeightsErr.Error()))
	}
	// second, scan all files
	nameEndMp, nameIdArrays, _, scanError := scanBfdbPathDir(pathDir)
	if scanError != nil {
		GlobalLogger.Errorf("runScanFilesNoBfdb scanBfdbPathDir [%s] error %s",
			pathDir, scanError.Error())
		panic(fmt.Sprintf("runScanFilesNoBfdb scanBfdbPathDir [%s] error %s",
			pathDir, scanError.Error()))
	}
	// output csv
	ts := time.Now()
	fName := getCsvFileName(ts)
	ofile, oWriter := createCSVFile(fName)
	defer ofile.Close()
	for i := 0; i < len(nameIdArrays); i++ {
		tempBeginId := nameIdArrays[i]
		tempBeginHeight := tempBeginId - 1      // 文件开始高度
		tempEndHeight := nameEndMp[tempBeginId] // 文件结束高度
		if height < tempBeginHeight {           // 过高的区块文件不再扫描
			break
		}
		if height < tempEndHeight {
			break
		}
		var record BfdbFileInfo
		record.BeginHeight = tempBeginHeight
		record.EndHeight = tempEndHeight
		record.FileName = getBFDBName(tempBeginId)
		for _, confHeight := range configHeights {
			if confHeight >= tempBeginHeight && confHeight <= tempEndHeight {
				record.ConfigHeights = append(record.ConfigHeights, confHeight)
			}
		}
		if len(record.ConfigHeights) > 0 {
			record.CanClean = "n"
		} else {
			record.CanClean = "y"
		}
		writeFileRecord(&record, oWriter)
	}

}

// getCsvFileName 计算csv文件名称
// @param ts
// @return string
func getCsvFileName(ts time.Time) string {
	return fmt.Sprintf("stub%d%02d%02d%02d%02d%02d.csv",
		ts.Year(), ts.Month(), ts.Day(), ts.Hour(), ts.Minute(), ts.Second())
}

// getBFDBName 计算fdb文件名
// @param heightId
// @return string
func getBFDBName(heightId uint64) string {
	return fmt.Sprintf("%020d.fdb", heightId)
}
