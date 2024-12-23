/*
Copyright (C) BABEC. All rights reserved.
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package chainmaker_sdk_go

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewConnPool(t *testing.T) {
	conf, err := generateConfig(WithConfPath(sdkConfigPathForUT))
	require.Nil(t, err)
	_, err = NewConnPool(conf)
	require.Nil(t, err)
}

func TestNewCanonicalTxFetcherPools(t *testing.T) {
	conf, err := generateConfig(WithConfPath(sdkConfigPathForUT))
	require.Nil(t, err)
	_, err = NewCanonicalTxFetcherPools(conf)
	require.Nil(t, err)
}
