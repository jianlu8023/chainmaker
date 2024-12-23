/*
Copyright (C) BABEC. All rights reserved.


SPDX-License-Identifier: Apache-2.0
*/

// Package archive_utils define common utils
package archive_utils

import "sync"

var (
	// GlobalServerLatch 同步使用
	GlobalServerLatch sync.WaitGroup
)
