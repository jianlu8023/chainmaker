/*
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package chainmaker_sdk_go

import (
	"fmt"
	"io/ioutil"
	"os"
	"testing"

	"chainmaker.org/chainmaker/common/v2/crypto"
	bcx509 "chainmaker.org/chainmaker/common/v2/crypto/x509"
	"chainmaker.org/chainmaker/pb-go/v2/common"
	"github.com/gogo/protobuf/proto"
	"github.com/stretchr/testify/require"
)

func TestSignPayloadWithPath(t *testing.T) {
	tests := []struct {
		name            string
		unsignedPayload *common.Payload
		wantErr         bool
	}{
		{
			"good",
			&common.Payload{
				ChainId: "chain1",
			},
			false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cli, err := newMockChainClient(nil, nil, WithConfPath(sdkConfigPathForUT))
			require.Nil(t, err)
			defer cli.Stop()

			e, err := SignPayloadWithPath(cli.ConfigModel.ChainClientConfig.UserSignKeyFilePath,
				cli.ConfigModel.ChainClientConfig.UserSignCrtFilePath, tt.unsignedPayload)
			require.Equal(t, tt.wantErr, err != nil)

			payloadBz, err := proto.Marshal(tt.unsignedPayload)
			require.Nil(t, err)

			var opts crypto.SignOpts
			hashalgo, err := bcx509.GetHashFromSignatureAlgorithm(cli.userCrt.SignatureAlgorithm)
			require.Nil(t, err)

			opts.Hash = hashalgo
			opts.UID = crypto.CRYPTO_DEFAULT_UID

			verified, err := cli.userCrt.PublicKey.VerifyWithOpts(payloadBz, e.Signature, &opts)
			require.Nil(t, err)
			require.True(t, verified)
		})
	}
}

func TestEVMAddress(t *testing.T) {
	tests := []struct {
		name       string
		certPEM    string
		pubKeyHex  string
		pubKeyPEM  string
		privKeyPEM string
		hashType   crypto.HashType
		expect     string
	}{
		{
			"test parse evm address",
			`-----BEGIN CERTIFICATE-----
MIICdzCCAh6gAwIBAgIDDE8jMAoGCCqGSM49BAMCMIGKMQswCQYDVQQGEwJDTjEQ
MA4GA1UECBMHQmVpamluZzEQMA4GA1UEBxMHQmVpamluZzEfMB0GA1UEChMWd3gt
b3JnMS5jaGFpbm1ha2VyLm9yZzESMBAGA1UECxMJcm9vdC1jZXJ0MSIwIAYDVQQD
ExljYS53eC1vcmcxLmNoYWlubWFrZXIub3JnMB4XDTIyMDgwNTA3MjMwMloXDTI3
MDgwNDA3MjMwMlowgZExCzAJBgNVBAYTAkNOMRAwDgYDVQQIEwdCZWlqaW5nMRAw
DgYDVQQHEwdCZWlqaW5nMR8wHQYDVQQKExZ3eC1vcmcxLmNoYWlubWFrZXIub3Jn
MQ8wDQYDVQQLEwZjbGllbnQxLDAqBgNVBAMTI2NsaWVudDEuc2lnbi53eC1vcmcx
LmNoYWlubWFrZXIub3JnMFkwEwYHKoZIzj0CAQYIKoZIzj0DAQcDQgAE7+Szqb9f
7oHxobnS+D92ADt6jmDe2XglbjecXcvPJwXLeAd9FxDu/UBFoM+saQO4hvWzNCmR
E3lOFD7spSU9RaNqMGgwDgYDVR0PAQH/BAQDAgbAMCkGA1UdDgQiBCCu+njHmEJm
5j/qsEa1nHK1IF2IEu4tCKKjp5/ossJvyjArBgNVHSMEJDAigCC7A23XtvVNLr0L
cyi3A4jFqiOmkI7DV9VL2ShhPFZ7zzAKBggqhkjOPQQDAgNHADBEAiBtOtyOojJm
u1wNOCOjDiGCcPcKjtQbsma+fMqkd8nYAwIgWSXedyOvUEyO8browF+5UAwOpzjO
0deYCoRcxyWnJqU=
-----END CERTIFICATE-----`,
			`3059301306072a8648ce3d020106082a8648ce3d03010703420004efe4b3a9bf5fee81f1a1b9d2f83f76003b7a8e60ded978256e379c5dcbcf2705cb78077d1710eefd4045a0cfac6903b886f5b334299113794e143eeca5253d45`,
			`-----BEGIN PUBLIC KEY-----
MFkwEwYHKoZIzj0CAQYIKoZIzj0DAQcDQgAE7+Szqb9f7oHxobnS+D92ADt6jmDe
2XglbjecXcvPJwXLeAd9FxDu/UBFoM+saQO4hvWzNCmRE3lOFD7spSU9RQ==
-----END PUBLIC KEY-----`,
			`-----BEGIN EC PRIVATE KEY-----
MHcCAQEEIAG7Q35sgdRJallUZ8OepYje26P6CwdckSUBccjM5g7woAoGCCqGSM49
AwEHoUQDQgAE7+Szqb9f7oHxobnS+D92ADt6jmDe2XglbjecXcvPJwXLeAd9FxDu
/UBFoM+saQO4hvWzNCmRE3lOFD7spSU9RQ==
-----END EC PRIVATE KEY-----`,
			crypto.HASH_TYPE_SHA256,
			"d111b71769c5f0178ede4c0a06af744887fda53d",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			certTmpFile, err := os.CreateTemp("", "")
			require.Nil(t, err)
			_, err = certTmpFile.Write([]byte(tt.certPEM))
			require.Nil(t, err)
			defer os.Remove(certTmpFile.Name())

			addr, err := GetEVMAddressFromCertPath(certTmpFile.Name())
			require.Nil(t, err)
			require.Equal(t, tt.expect, addr)
			addr, err = GetEVMAddressFromCertBytes([]byte(tt.certPEM))
			require.Nil(t, err)
			require.Equal(t, tt.expect, addr)

			privKeyTmpFile, err := os.CreateTemp("", "")
			require.Nil(t, err)
			_, err = privKeyTmpFile.Write([]byte(tt.privKeyPEM))
			require.Nil(t, err)
			defer os.Remove(privKeyTmpFile.Name())

			addr, err = GetEVMAddressFromPrivateKeyPath(privKeyTmpFile.Name(), tt.hashType)
			require.Nil(t, err)
			require.Equal(t, tt.expect, addr)
			addr, err = GetEVMAddressFromPrivateKeyBytes([]byte(tt.privKeyPEM), tt.hashType)
			require.Nil(t, err)
			require.Equal(t, tt.expect, addr)
			addr, err = GetEVMAddressFromPKHex(tt.pubKeyHex, tt.hashType)
			require.Nil(t, err)
			require.Equal(t, tt.expect, addr)
			addr, err = GetEVMAddressFromPKPEM(tt.pubKeyPEM, tt.hashType)
			require.Nil(t, err)
			require.Equal(t, tt.expect, addr)
		})
	}
}

func TestZXAddress(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		expect string
	}{
		{
			"pkHex",
			"3059301306072a8648ce3d020106082a811ccf5501822d034200044a4c24cf037b0c7a027e634b994a5fdbcd0faa718ce9053e3f75fcb9a865523a605aff92b5f99e728f51a924d4f18d5819c42f9b626bdf6eea911946efe7442d",
			"ZXaaa6f45415493ffb832ca28faa14bef5c357f5f0",
		},
		{
			"pkPEM",
			`-----BEGIN PUBLIC KEY-----
MFkwEwYHKoZIzj0CAQYIKoEcz1UBgi0DQgAESkwkzwN7DHoCfmNLmUpf280PqnGM
6QU+P3X8uahlUjpgWv+Stfmeco9RqSTU8Y1YGcQvm2Jr327qkRlG7+dELQ==
-----END PUBLIC KEY-----`,
			"ZXaaa6f45415493ffb832ca28faa14bef5c357f5f0",
		},
		{
			"certPEM",
			`-----BEGIN CERTIFICATE-----
MIICzjCCAi+gAwIBAgIDCzLUMAoGCCqGSM49BAMCMGoxCzAJBgNVBAYTAkNOMRAw
DgYDVQQIEwdCZWlqaW5nMRAwDgYDVQQHEwdCZWlqaW5nMRAwDgYDVQQKEwd3eC1v
cmcxMRAwDgYDVQQLEwdyb290LWNhMRMwEQYDVQQDEwp3eC1vcmcxLWNhMB4XDTIw
MTAyOTEzMzgxMFoXDTMwMTAyNzEzMzgxMFowcDELMAkGA1UEBhMCQ04xEDAOBgNV
BAgTB0JlaWppbmcxEDAOBgNVBAcTB0JlaWppbmcxEDAOBgNVBAoTB3d4LW9yZzEx
EzARBgNVBAsTCkNoYWluTWFrZXIxFjAUBgNVBAMTDXVzZXIxLnd4LW9yZzEwgZsw
EAYHKoZIzj0CAQYFK4EEACMDgYYABAGLEJZriYzK9Se/vMGfkwjhU55eEZsM2iKM
emSZICh/HY37uR0BFAVUjMYEj84tJBzEEzlpD+AUAe44/b11b+GCMwDXPKcsjHK0
jsAPrN5LH7uptXsjMFpN2bbOqvj6sAIDfTV9chuF91LxCjYnh+Lya0ikextGkpbp
HOvi5eQ/yUHSQaN7MHkwDgYDVR0PAQH/BAQDAgGmMA8GA1UdJQQIMAYGBFUdJQAw
KQYDVR0OBCIEIAp+6tWmoiE0KmdtpLFBZpBj1Ni7JH8g2XPgoQwhQS8qMCsGA1Ud
IwQkMCKAIMsnP+UWEyGuyEHBn7JkJzb+tfBqsRCBUIPyMZH4h1HPMAoGCCqGSM49
BAMCA4GMADCBiAJCAIENc8ip2BP4yJpj9SdR9pvZc4/qbBzKucZQaD/GT2sj0FxH
hp8YLjSflgw1+uWlMb/WCY60MyxZr/RRsTYpHu7FAkIBSMAVxw5RYySsf4J3bpM0
CpIO2ZrxkJ1Nm/FKZzMLQjp7Dm//xEMkpCbqqC6koOkRP2MKGSnEGXGfRr1QgBvr
8H8=
-----END CERTIFICATE-----`,
			"ZX0787b8affa4cbdb9994548010c80d9741113ae78",
		},
		{
			"certPath",
			`-----BEGIN CERTIFICATE-----
MIICzjCCAi+gAwIBAgIDCzLUMAoGCCqGSM49BAMCMGoxCzAJBgNVBAYTAkNOMRAw
DgYDVQQIEwdCZWlqaW5nMRAwDgYDVQQHEwdCZWlqaW5nMRAwDgYDVQQKEwd3eC1v
cmcxMRAwDgYDVQQLEwdyb290LWNhMRMwEQYDVQQDEwp3eC1vcmcxLWNhMB4XDTIw
MTAyOTEzMzgxMFoXDTMwMTAyNzEzMzgxMFowcDELMAkGA1UEBhMCQ04xEDAOBgNV
BAgTB0JlaWppbmcxEDAOBgNVBAcTB0JlaWppbmcxEDAOBgNVBAoTB3d4LW9yZzEx
EzARBgNVBAsTCkNoYWluTWFrZXIxFjAUBgNVBAMTDXVzZXIxLnd4LW9yZzEwgZsw
EAYHKoZIzj0CAQYFK4EEACMDgYYABAGLEJZriYzK9Se/vMGfkwjhU55eEZsM2iKM
emSZICh/HY37uR0BFAVUjMYEj84tJBzEEzlpD+AUAe44/b11b+GCMwDXPKcsjHK0
jsAPrN5LH7uptXsjMFpN2bbOqvj6sAIDfTV9chuF91LxCjYnh+Lya0ikextGkpbp
HOvi5eQ/yUHSQaN7MHkwDgYDVR0PAQH/BAQDAgGmMA8GA1UdJQQIMAYGBFUdJQAw
KQYDVR0OBCIEIAp+6tWmoiE0KmdtpLFBZpBj1Ni7JH8g2XPgoQwhQS8qMCsGA1Ud
IwQkMCKAIMsnP+UWEyGuyEHBn7JkJzb+tfBqsRCBUIPyMZH4h1HPMAoGCCqGSM49
BAMCA4GMADCBiAJCAIENc8ip2BP4yJpj9SdR9pvZc4/qbBzKucZQaD/GT2sj0FxH
hp8YLjSflgw1+uWlMb/WCY60MyxZr/RRsTYpHu7FAkIBSMAVxw5RYySsf4J3bpM0
CpIO2ZrxkJ1Nm/FKZzMLQjp7Dm//xEMkpCbqqC6koOkRP2MKGSnEGXGfRr1QgBvr
8H8=
-----END CERTIFICATE-----`,
			"ZX0787b8affa4cbdb9994548010c80d9741113ae78",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var (
				err  error
				addr string
			)

			if tt.name == "pkHex" {
				addr, err = GetZXAddressFromPKHex(tt.input, crypto.HASH_TYPE_SM3)
			} else if tt.name == "pkPEM" {
				addr, err = GetZXAddressFromPKPEM(tt.input, crypto.HASH_TYPE_SM3)
			} else if tt.name == "certPEM" {
				addr, err = GetZXAddressFromCertPEM(tt.input)
			} else if tt.name == "certPath" {
				var tmpFile *os.File
				tmpFile, err = ioutil.TempFile(os.TempDir(), "zx-*.crt")
				require.Nil(t, err)

				defer os.Remove(tmpFile.Name())

				_, err = tmpFile.Write([]byte(tt.input))
				require.Nil(t, err)

				addr, err = GetZXAddressFromCertPath(tmpFile.Name())
			}

			fmt.Printf("ZX address: %s\n", addr)
			require.Nil(t, err)
			require.Equal(t, tt.expect, addr)
		})
	}
}

func TestCMAddress(t *testing.T) {
	type input struct {
		data     string
		hashType crypto.HashType
	}

	tests := []struct {
		name   string
		input  input
		expect string
	}{
		{
			"CMPkHex",
			input{
				"3059301306072a8648ce3d020106082a811ccf5501822d034200044a4c24cf037b0c7a027e634b994a5fdbcd0faa718ce9053e3f75fcb9a865523a605aff92b5f99e728f51a924d4f18d5819c42f9b626bdf6eea911946efe7442d",
				crypto.HASH_TYPE_SHA256,
			},
			"4cd0b5e8f6d6df38ecdc06c7431a48dd0265cb1e",
		},
		{
			"CMPkPEM",
			input{
				`-----BEGIN PUBLIC KEY-----
MFkwEwYHKoZIzj0CAQYIKoEcz1UBgi0DQgAESkwkzwN7DHoCfmNLmUpf280PqnGM
6QU+P3X8uahlUjpgWv+Stfmeco9RqSTU8Y1YGcQvm2Jr327qkRlG7+dELQ==
-----END PUBLIC KEY-----`,
				crypto.HASH_TYPE_SHA256,
			},
			"4cd0b5e8f6d6df38ecdc06c7431a48dd0265cb1e",
		},
		{
			"CMCertPEM",
			input{
				`-----BEGIN CERTIFICATE-----
MIICzjCCAi+gAwIBAgIDCzLUMAoGCCqGSM49BAMCMGoxCzAJBgNVBAYTAkNOMRAw
DgYDVQQIEwdCZWlqaW5nMRAwDgYDVQQHEwdCZWlqaW5nMRAwDgYDVQQKEwd3eC1v
cmcxMRAwDgYDVQQLEwdyb290LWNhMRMwEQYDVQQDEwp3eC1vcmcxLWNhMB4XDTIw
MTAyOTEzMzgxMFoXDTMwMTAyNzEzMzgxMFowcDELMAkGA1UEBhMCQ04xEDAOBgNV
BAgTB0JlaWppbmcxEDAOBgNVBAcTB0JlaWppbmcxEDAOBgNVBAoTB3d4LW9yZzEx
EzARBgNVBAsTCkNoYWluTWFrZXIxFjAUBgNVBAMTDXVzZXIxLnd4LW9yZzEwgZsw
EAYHKoZIzj0CAQYFK4EEACMDgYYABAGLEJZriYzK9Se/vMGfkwjhU55eEZsM2iKM
emSZICh/HY37uR0BFAVUjMYEj84tJBzEEzlpD+AUAe44/b11b+GCMwDXPKcsjHK0
jsAPrN5LH7uptXsjMFpN2bbOqvj6sAIDfTV9chuF91LxCjYnh+Lya0ikextGkpbp
HOvi5eQ/yUHSQaN7MHkwDgYDVR0PAQH/BAQDAgGmMA8GA1UdJQQIMAYGBFUdJQAw
KQYDVR0OBCIEIAp+6tWmoiE0KmdtpLFBZpBj1Ni7JH8g2XPgoQwhQS8qMCsGA1Ud
IwQkMCKAIMsnP+UWEyGuyEHBn7JkJzb+tfBqsRCBUIPyMZH4h1HPMAoGCCqGSM49
BAMCA4GMADCBiAJCAIENc8ip2BP4yJpj9SdR9pvZc4/qbBzKucZQaD/GT2sj0FxH
hp8YLjSflgw1+uWlMb/WCY60MyxZr/RRsTYpHu7FAkIBSMAVxw5RYySsf4J3bpM0
CpIO2ZrxkJ1Nm/FKZzMLQjp7Dm//xEMkpCbqqC6koOkRP2MKGSnEGXGfRr1QgBvr
8H8=
-----END CERTIFICATE-----`,
				crypto.HASH_TYPE_SHA256,
			},

			"305f98514f3c2f6fcaeb8247ed147bacf99990f8",
		},
		{
			"CMCertPath",
			input{
				`-----BEGIN CERTIFICATE-----
MIICzjCCAi+gAwIBAgIDCzLUMAoGCCqGSM49BAMCMGoxCzAJBgNVBAYTAkNOMRAw
DgYDVQQIEwdCZWlqaW5nMRAwDgYDVQQHEwdCZWlqaW5nMRAwDgYDVQQKEwd3eC1v
cmcxMRAwDgYDVQQLEwdyb290LWNhMRMwEQYDVQQDEwp3eC1vcmcxLWNhMB4XDTIw
MTAyOTEzMzgxMFoXDTMwMTAyNzEzMzgxMFowcDELMAkGA1UEBhMCQ04xEDAOBgNV
BAgTB0JlaWppbmcxEDAOBgNVBAcTB0JlaWppbmcxEDAOBgNVBAoTB3d4LW9yZzEx
EzARBgNVBAsTCkNoYWluTWFrZXIxFjAUBgNVBAMTDXVzZXIxLnd4LW9yZzEwgZsw
EAYHKoZIzj0CAQYFK4EEACMDgYYABAGLEJZriYzK9Se/vMGfkwjhU55eEZsM2iKM
emSZICh/HY37uR0BFAVUjMYEj84tJBzEEzlpD+AUAe44/b11b+GCMwDXPKcsjHK0
jsAPrN5LH7uptXsjMFpN2bbOqvj6sAIDfTV9chuF91LxCjYnh+Lya0ikextGkpbp
HOvi5eQ/yUHSQaN7MHkwDgYDVR0PAQH/BAQDAgGmMA8GA1UdJQQIMAYGBFUdJQAw
KQYDVR0OBCIEIAp+6tWmoiE0KmdtpLFBZpBj1Ni7JH8g2XPgoQwhQS8qMCsGA1Ud
IwQkMCKAIMsnP+UWEyGuyEHBn7JkJzb+tfBqsRCBUIPyMZH4h1HPMAoGCCqGSM49
BAMCA4GMADCBiAJCAIENc8ip2BP4yJpj9SdR9pvZc4/qbBzKucZQaD/GT2sj0FxH
hp8YLjSflgw1+uWlMb/WCY60MyxZr/RRsTYpHu7FAkIBSMAVxw5RYySsf4J3bpM0
CpIO2ZrxkJ1Nm/FKZzMLQjp7Dm//xEMkpCbqqC6koOkRP2MKGSnEGXGfRr1QgBvr
8H8=
-----END CERTIFICATE-----`,
				crypto.HASH_TYPE_SHA256,
			},

			"305f98514f3c2f6fcaeb8247ed147bacf99990f8",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var (
				err  error
				addr string
			)

			if tt.name == "CMPkHex" {
				addr, err = GetCMAddressFromPKHex(tt.input.data, tt.input.hashType)
			} else if tt.name == "CMPkPEM" {
				addr, err = GetCMAddressFromPKPEM(tt.input.data, tt.input.hashType)
			} else if tt.name == "CMCertPEM" {
				addr, err = GetCMAddressFromCertPEM(tt.input.data)
			} else if tt.name == "CMCertPath" {
				var tmpFile *os.File
				tmpFile, err = ioutil.TempFile(os.TempDir(), "zx-*.crt")
				require.Nil(t, err)

				defer os.Remove(tmpFile.Name())

				_, err = tmpFile.Write([]byte(tt.input.data))
				require.Nil(t, err)

				addr, err = GetCMAddressFromCertPath(tmpFile.Name())
			}

			fmt.Printf("CM address: %s\n", addr)
			require.Nil(t, err)
			require.Equal(t, tt.expect, addr)
		})
	}
}
