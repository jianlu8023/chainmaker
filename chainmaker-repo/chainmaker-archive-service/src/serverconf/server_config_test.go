/*
Copyright (C) BABEC. All rights reserved.


SPDX-License-Identifier: Apache-2.0
*/

// Package serverconf define config
package serverconf

import "testing"

func TestLoadCFG(t *testing.T) {
	readError := ReadConfigFile("../../configs/config.yml")
	if readError != nil {
		t.Errorf("read config file got error %s ", readError.Error())
	}
}
