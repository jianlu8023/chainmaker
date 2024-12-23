/*
Copyright (C) BABEC. All rights reserved.


SPDX-License-Identifier: Apache-2.0
*/

// Package filestore define file operation
package filestore

import (
	"bytes"
	"io/ioutil"
	"os"
)

func CompareFile(file1, file2 string) (bool, error) {
	fileRead1, fileRead1Error := os.Open(file1)
	if fileRead1Error != nil {
		return false, fileRead1Error
	}
	defer fileRead1.Close()
	fileRead2, fileRead2Error := os.Open(file2)
	if fileRead2Error != nil {
		return false, fileRead2Error
	}
	defer fileRead2.Close()
	content1, content1Error := ioutil.ReadAll(fileRead1)
	if content1Error != nil {
		return false, content1Error
	}

	content2, content2Error := ioutil.ReadAll(fileRead2)
	if content2Error != nil {
		return false, content2Error
	}
	if bytes.Equal(content1, content2) {
		return true, nil
	}
	return false, nil
}
