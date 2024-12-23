/*
Copyright (C) BABEC. All rights reserved.


SPDX-License-Identifier: Apache-2.0
*/

// Package archive_utils define common utils
package archive_utils

import "math/rand"

var letters = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")

// RandSeq 计算n位随机数
func RandSeq(n int) string {
	b := make([]rune, n)
	for i := range b {
		//nolint
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}
