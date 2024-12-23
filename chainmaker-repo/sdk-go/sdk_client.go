/*
Copyright (C) BABEC. All rights reserved.
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package chainmaker_sdk_go

import (
	"encoding/hex"
	"encoding/pem"
	"errors"
	"fmt"
	"time"

	"chainmaker.org/chainmaker/common/v2/crypto"
	"chainmaker.org/chainmaker/common/v2/crypto/asym"
	bcx509 "chainmaker.org/chainmaker/common/v2/crypto/x509"
	"chainmaker.org/chainmaker/pb-go/v2/accesscontrol"
	"chainmaker.org/chainmaker/pb-go/v2/common"
	"chainmaker.org/chainmaker/sdk-go/v2/utils"
	"github.com/Rican7/retry"
	"github.com/Rican7/retry/strategy"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	errStringFormat    = "%s failed, %s"
	sdkErrStringFormat = "[SDK] %s"
)

var _ SDKInterface = (*ChainClient)(nil)

// ChainClient define chainmaker chain client to interact with node
type ChainClient struct {
	// common config
	logger                  utils.Logger
	pool                    ConnectionPool
	canonicalTxFetcherPools map[string]ConnectionPool
	txResultDispatcher      *txResultDispatcher
	chainId                 string
	orgId                   string

	userCrtBytes []byte
	userCrt      *bcx509.Certificate
	privateKey   crypto.PrivateKey
	publicKey    crypto.PublicKey
	pkBytes      []byte

	// cert hash config
	enabledCrtHash bool
	userCrtHash    []byte

	// archive config
	archiveConfig *ArchiveConfig

	// grpc client config
	rpcClientConfig *RPCClientConfig

	// pkcs11 config
	pkcs11Config *Pkcs11Config

	hashType crypto.HashType
	authType AuthType
	// retry config
	retryLimit    int // if <=0 then use DefaultRetryLimit
	retryInterval int // if <=0 then use DefaultRetryInterval

	// alias support
	enabledAlias bool
	alias        string

	// default TimestampKey , true NormalKey support
	enableNormalKey bool

	// enable tx result dispatcher
	enableTxResultDispatcher bool
	// enable sync canonical tx result
	enableSyncCanonicalTxResult bool

	ConfigModel *utils.ChainClientConfigModel

	archiveCenterConfig *ArchiveCenterHttpConfig
}

// NewNodeConfig new node config, returns *NodeConfig
func NewNodeConfig(opts ...NodeOption) *NodeConfig {
	config := &NodeConfig{}
	for _, opt := range opts {
		opt(config)
	}

	return config
}

// NewConnPoolWithOptions new conn pool with optioins, returns *ClientConnectionPool
func NewConnPoolWithOptions(opts ...ChainClientOption) (*ClientConnectionPool, error) {
	config, err := generateConfig(opts...)
	if err != nil {
		return nil, err
	}

	return NewConnPool(config)
}

// NewArchiveConfig new archive config
func NewArchiveConfig(opts ...ArchiveOption) *ArchiveConfig {
	config := &ArchiveConfig{}
	for _, opt := range opts {
		opt(config)
	}

	return config
}

// NewRPCClientConfig new rpc client config
func NewRPCClientConfig(opts ...RPCClientOption) *RPCClientConfig {
	config := &RPCClientConfig{}
	for _, opt := range opts {
		opt(config)
	}
	return config
}

// NewPkcs11Config new pkcs11 config
func NewPkcs11Config(enabled bool, typ, libPath, label, password string,
	sessionCacheSize int, hashAlgo string) *Pkcs11Config {
	return &Pkcs11Config{
		Enabled:          enabled,
		Type:             typ,
		Library:          libPath,
		Label:            label,
		Password:         password,
		SessionCacheSize: sessionCacheSize,
		Hash:             hashAlgo,
	}
}

// NewCryptoConfig 根据传入参数创建新的CryptoConfig对象
// @param opts
// @return *CryptoConfig
func NewCryptoConfig(opts ...CryptoOption) *CryptoConfig {
	config := &CryptoConfig{}
	for _, opt := range opts {
		opt(config)
	}
	return config
}

// NewArchiveCenterConfig 根据传入参数创建新的归档中心对象
// @param httpUrl
// @param chainGenesisHash
// @param requestSecondLimit
// @return *ArchiveCenterHttpConfig
func NewArchiveCenterConfig(httpUrl, chainGenesisHash string,
	requestSecondLimit int) *ArchiveCenterHttpConfig {
	ret := &ArchiveCenterHttpConfig{
		ChainGenesisHash:     chainGenesisHash,
		ArchiveCenterHttpUrl: httpUrl,
		ReqeustSecondLimit:   httpRequestDuration,
	}
	if requestSecondLimit > 0 {
		ret.ReqeustSecondLimit = requestSecondLimit
	}
	return ret
}

// NewChainClient new chain client
func NewChainClient(opts ...ChainClientOption) (*ChainClient, error) {
	config, err := generateConfig(opts...)
	if err != nil {
		return nil, err
	}

	pool, err := NewConnPool(config)
	if err != nil {
		return nil, err
	}

	var hashType crypto.HashType
	if config.authType == PermissionedWithKey || config.authType == Public {
		hashType = crypto.HashAlgoMap[config.crypto.hash]
	} else {
		hashType, err = bcx509.GetHashFromSignatureAlgorithm(config.userCrt.SignatureAlgorithm)
		if err != nil {
			return nil, err
		}
	}

	var publicKey crypto.PublicKey
	var pkBytes []byte
	var pkPem string
	publicKey = config.userPk
	pkPem, err = publicKey.String()
	if err != nil {
		return nil, err
	}
	pkBytes = []byte(pkPem)

	cc := &ChainClient{
		pool:            pool,
		logger:          config.logger,
		chainId:         config.chainId,
		orgId:           config.orgId,
		alias:           config.alias,
		userCrtBytes:    config.userSignCrtBytes,
		userCrt:         config.userCrt,
		privateKey:      config.privateKey,
		archiveConfig:   config.archiveConfig,
		rpcClientConfig: config.rpcClientConfig,
		pkcs11Config:    config.pkcs11Config,

		publicKey: publicKey,
		hashType:  hashType,
		authType:  config.authType,
		pkBytes:   pkBytes,

		retryLimit:    config.retryLimit,
		retryInterval: config.retryInterval,

		enableNormalKey:             config.enableNormalKey,
		enableTxResultDispatcher:    config.enableTxResultDispatcher,
		enableSyncCanonicalTxResult: config.enableSyncCanonicalTxResult,

		ConfigModel: config.ConfigModel,
	}

	// 若设置了别名，便启用
	if config.authType == PermissionedWithCert && len(cc.alias) > 0 {
		if err := cc.EnableAlias(); err != nil {
			return nil, err
		}
	}

	// 启动 异步订阅交易结果
	if cc.enableTxResultDispatcher {
		cc.txResultDispatcher = newTxResultDispatcher(cc)
		go cc.txResultDispatcher.start()
	} else if cc.enableSyncCanonicalTxResult {
		cc.canonicalTxFetcherPools, err = NewCanonicalTxFetcherPools(config)
		if err != nil {
			return nil, err
		}
	}

	// 若设置了归档中心配置,则启用
	if config.archiveCenterHttpConfig != nil && len(config.archiveCenterHttpConfig.ArchiveCenterHttpUrl) > 0 {
		cc.archiveCenterConfig = config.archiveCenterHttpConfig
	}
	return cc, nil
}

// IsEnableNormalKey whether to use normal key
func (cc *ChainClient) IsEnableNormalKey() bool {
	return cc.enableNormalKey
}

// Stop stop chain client
func (cc *ChainClient) Stop() error {
	if cc.txResultDispatcher != nil {
		cc.txResultDispatcher.stop()
	}
	for _, pool := range cc.canonicalTxFetcherPools {
		pool.Close()
	}
	return cc.pool.Close()
}

func (cc *ChainClient) proposalRequest(payload *common.Payload,
	endorsers []*common.EndorsementEntry) (*common.TxResponse, error) {
	return cc.proposalRequestWithTimeout(payload, endorsers, -1)
}

func (cc *ChainClient) proposalRequestWithTimeout(payload *common.Payload, endorsers []*common.EndorsementEntry,
	timeout int64) (*common.TxResponse, error) {

	req, err := cc.GenerateTxRequest(payload, endorsers)
	if err != nil {
		return nil, err
	}

	return cc.sendTxRequest(req, timeout)
}

func (cc *ChainClient) newAccessMember() *accesscontrol.Member {
	var member *accesscontrol.Member
	if cc.authType == PermissionedWithCert {
		if cc.enabledAlias && len(cc.alias) > 0 {
			member = &accesscontrol.Member{
				OrgId:      cc.orgId,
				MemberInfo: []byte(cc.alias),
				MemberType: accesscontrol.MemberType_ALIAS,
			}
		} else if cc.enabledCrtHash && len(cc.userCrtHash) > 0 {
			member = &accesscontrol.Member{
				OrgId:      cc.orgId,
				MemberInfo: cc.userCrtHash,
				MemberType: accesscontrol.MemberType_CERT_HASH,
			}
		} else {
			member = &accesscontrol.Member{
				OrgId:      cc.orgId,
				MemberInfo: cc.userCrtBytes,
				MemberType: accesscontrol.MemberType_CERT,
			}
		}
	} else {
		member = &accesscontrol.Member{
			OrgId:      cc.orgId,
			MemberInfo: cc.pkBytes,
			MemberType: accesscontrol.MemberType_PUBLIC_KEY,
		}
	}
	return member
}

// GenerateTxRequest sign payload and generate *common.TxRequest
func (cc *ChainClient) GenerateTxRequest(payload *common.Payload,
	endorsers []*common.EndorsementEntry) (*common.TxRequest, error) {
	req := &common.TxRequest{
		Payload: payload,
		Sender: &common.EndorsementEntry{
			Signer: cc.newAccessMember(),
		},
		Endorsers: endorsers,
	}
	var err error
	req.Sender.Signature, err = utils.SignPayloadWithHashType(cc.privateKey, cc.hashType, payload)
	if err != nil {
		return nil, err
	}
	return req, nil
}

// GenerateTxRequestBySigner sign payload and generate *common.TxRequest
// use signer to sign payload if it is not nil.
// use cc.privateKey to sign payload if signer is nil.
func (cc *ChainClient) GenerateTxRequestBySigner(payload *common.Payload, endorsers []*common.EndorsementEntry,
	signer Signer) (*common.TxRequest, error) {
	if signer == nil {
		return cc.GenerateTxRequest(payload, endorsers)
	}

	sig, err := signer.Sign(payload)
	if err != nil {
		return nil, err
	}

	member, err := signer.NewMember()
	if err != nil {
		return nil, err
	}

	return &common.TxRequest{
		Payload: payload,
		Sender: &common.EndorsementEntry{
			Signer:    member,
			Signature: sig,
		},
		Endorsers: endorsers,
	}, nil
}

func (cc *ChainClient) sendTxRequest(txRequest *common.TxRequest, timeout int64) (*common.TxResponse, error) {

	var (
		errMsg string
	)

	if timeout <= 0 {
		if txRequest.Payload.TxType == common.TxType_QUERY_CONTRACT {
			timeout = cc.rpcClientConfig.rpcClientGetTxTimeout
		} else {
			timeout = cc.rpcClientConfig.rpcClientSendTxTimeout
		}
	}

	ignoreAddrs := make(map[string]struct{})
	for {
		client, err := cc.pool.getClientWithIgnoreAddrs(ignoreAddrs)
		if err != nil {
			return nil, err
		}

		if len(ignoreAddrs) > 0 {
			cc.logger.Debugf("[SDK] begin try to connect node [%s]", client.ID)
		}

		resp, err := client.sendRequestWithTimeout(txRequest, timeout)
		if err != nil {
			resp := &common.TxResponse{
				Message: err.Error(),
				TxId:    txRequest.Payload.TxId,
			}

			statusErr, ok := status.FromError(err)
			if ok && (statusErr.Code() == codes.DeadlineExceeded ||
				// desc = "transport: Error while dialing dial tcp 127.0.0.1:12301: connect: connection refused"
				statusErr.Code() == codes.Unavailable) {

				resp.Code = common.TxStatusCode_TIMEOUT
				errMsg = fmt.Sprintf("call [%s] meet network error, try to connect another node if has, %s",
					client.ID, err.Error())

				cc.logger.Errorf(sdkErrStringFormat, errMsg)
				ignoreAddrs[client.ID] = struct{}{}
				continue
			}

			cc.logger.Errorf("statusErr.Code() : %s", statusErr.Code())

			resp.Code = common.TxStatusCode_INTERNAL_ERROR
			errMsg = fmt.Sprintf("client.call failed, %+v", err)
			cc.logger.Errorf(sdkErrStringFormat, errMsg)
			return resp, fmt.Errorf(errMsg)
		}

		resp.TxId = txRequest.Payload.TxId
		cc.logger.Debugf("[SDK] proposalRequest resp: %+v", resp)
		return resp, nil
	}
}

// EnableCertHash Cert Hash logic
func (cc *ChainClient) EnableCertHash() error {
	var (
		err error
	)

	// 优先使用别名，如果开启了别名，直接忽略压缩证书
	if cc.enabledAlias {
		return nil
	}

	if cc.GetAuthType() != PermissionedWithCert {
		return errors.New("cert hash is not supported")
	}

	// 0.已经启用压缩证书
	if cc.enabledCrtHash {
		return nil
	}

	// 1.如尚未获取证书Hash，便进行获取
	if len(cc.userCrtHash) == 0 {
		// 获取证书Hash
		cc.userCrtHash, err = cc.GetCertHash()
		if err != nil {
			errMsg := fmt.Sprintf("get cert hash failed, %s", err.Error())
			cc.logger.Errorf(sdkErrStringFormat, errMsg)
			return errors.New(errMsg)
		}
	}

	// 2.链上查询证书是否存在
	ok, err := cc.getCheckCertHash()
	if err != nil {
		errMsg := fmt.Sprintf("enable cert hash, get and check cert hash failed, %s", err.Error())
		cc.logger.Errorf(sdkErrStringFormat, errMsg)
		return errors.New(errMsg)
	}

	// 3.1 若证书已经上链，直接返回
	if ok {
		cc.enabledCrtHash = true
		return nil
	}

	// 3.2 若证书未上链，添加证书
	resp, err := cc.AddCert()
	if err != nil {
		errMsg := fmt.Sprintf("enable cert hash AddCert failed, %s", err.Error())
		cc.logger.Errorf(sdkErrStringFormat, errMsg)
		return errors.New(errMsg)
	}

	if err = utils.CheckProposalRequestResp(resp, true); err != nil {
		errMsg := fmt.Sprintf("enable cert hash AddCert got invalid resp, %s", err.Error())
		cc.logger.Errorf(sdkErrStringFormat, errMsg)
		return errors.New(errMsg)
	}

	// 循环检查证书是否成功上链
	err = cc.checkUserCertOnChain()
	if err != nil {
		errMsg := fmt.Sprintf("check user cert on chain failed, %s", err.Error())
		cc.logger.Errorf(sdkErrStringFormat, errMsg)
		return errors.New(errMsg)
	}

	cc.enabledCrtHash = true

	return nil
}

// DisableCertHash disable cert hash logic
func (cc *ChainClient) DisableCertHash() error {
	cc.enabledCrtHash = false
	return nil
}

// GetEnabledCrtHash check whether the cert hash logic is enabled
func (cc *ChainClient) GetEnabledCrtHash() bool {
	return cc.enabledCrtHash
}

// GetUserCrtHash get user cert hash of cc
func (cc *ChainClient) GetUserCrtHash() []byte {
	return cc.userCrtHash
}

// GetHashType get hash type of cc
func (cc *ChainClient) GetHashType() crypto.HashType {
	return cc.hashType
}

// GetAuthType get auth type of cc
func (cc *ChainClient) GetAuthType() AuthType {
	return cc.authType
}

// GetPublicKey get public key of cc
func (cc *ChainClient) GetPublicKey() crypto.PublicKey {
	return cc.publicKey
}

// GetPrivateKey get private key of cc
func (cc *ChainClient) GetPrivateKey() crypto.PrivateKey {
	return cc.privateKey
}

// GetCertPEM get cert pem of cc
func (cc *ChainClient) GetCertPEM() []byte {
	return cc.userCrtBytes
}

// GetLocalCertAlias get local cert alias of cc
func (cc *ChainClient) GetLocalCertAlias() string {
	return cc.alias
}

// ChangeSigner change ChainClient siger. signerCrt passes nil in Public or PermissionedWithKey mode
// publicModeHashType must be set in Public mode else set to zero value.
func (cc *ChainClient) ChangeSigner(signerPrivKey crypto.PrivateKey, signerCrt *bcx509.Certificate,
	publicModeHashType crypto.HashType) error {
	signerPubKey := signerPrivKey.PublicKey()
	pkPem, err := signerPubKey.String()
	if err != nil {
		return err
	}

	cc.pkBytes = []byte(pkPem)
	cc.publicKey = signerPubKey
	cc.privateKey = signerPrivKey

	if signerCrt != nil {
		crtPem := pem.EncodeToMemory(&pem.Block{Bytes: signerCrt.Raw, Type: "CERTIFICATE"})
		cc.userCrtBytes = crtPem
		cc.userCrt = signerCrt
	} else {
		cc.hashType = publicModeHashType
	}
	return nil
}

// 检查证书是否成功上链
func (cc *ChainClient) checkUserCertOnChain() error {
	err := retry.Retry(func(uint) error {
		ok, err := cc.getCheckCertHash()
		if err != nil {
			errMsg := fmt.Sprintf("check user cert on chain, get and check cert hash failed, %s", err.Error())
			cc.logger.Errorf(sdkErrStringFormat, errMsg)
			return errors.New(errMsg)
		}

		if !ok {
			errMsg := "user cert havenot on chain yet, and try again"
			cc.logger.Debugf(sdkErrStringFormat, errMsg)
			return errors.New(errMsg)
		}

		return nil
	}, strategy.Limit(10), strategy.Wait(time.Second))

	if err != nil {
		errMsg := fmt.Sprintf("check user upload cert on chain failed, try again later, %s", err.Error())
		cc.logger.Errorf(sdkErrStringFormat, errMsg)
		return errors.New(errMsg)
	}

	return nil
}

func (cc *ChainClient) getCheckCertHash() (bool, error) {
	// 根据已缓存证书Hash，查链上是否存在
	certInfo, err := cc.QueryCert([]string{hex.EncodeToString(cc.userCrtHash)})
	if err != nil {
		errMsg := fmt.Sprintf("QueryCert failed, %s", err.Error())
		cc.logger.Errorf(sdkErrStringFormat, errMsg)
		return false, errors.New(errMsg)
	}

	if len(certInfo.CertInfos) == 0 {
		return false, nil
	}

	// 返回链上证书列表长度不为1，即报错
	if len(certInfo.CertInfos) > 1 {
		errMsg := "CertInfos != 1"
		cc.logger.Errorf(sdkErrStringFormat, errMsg)
		return false, errors.New(errMsg)
	}

	// 如果链上证书Hash不为空
	if len(certInfo.CertInfos[0].Cert) > 0 {
		// 如果和缓存的证书Hash不一致则报错
		if hex.EncodeToString(cc.userCrtHash) != certInfo.CertInfos[0].Hash {
			errMsg := fmt.Sprintf("not equal certHash, [expected:%s]/[actual:%s]",
				cc.userCrtHash, certInfo.CertInfos[0].Hash)
			cc.logger.Errorf(sdkErrStringFormat, errMsg)
			return false, errors.New(errMsg)
		}

		// 如果和缓存的证书Hash一致，则说明已经上传好了证书，具备提交压缩证书交易的能力
		return true, nil
	}

	return false, nil
}

// Pkcs11Config get pkcs11 config of cc
func (cc *ChainClient) Pkcs11Config() *Pkcs11Config {
	return cc.pkcs11Config
}

// ArchiveCenterConfig 获取归档中心的配置信息
func (cc *ChainClient) ArchiveCenterConfig() *ArchiveCenterHttpConfig {
	return cc.archiveCenterConfig
}

// CreateChainClient create chain client and init chain client, returns *ChainClient
func CreateChainClient(pool ConnectionPool, userCrtBytes, privKey, userCrtHash []byte, orgId, chainId string,
	enabledCrtHash int) (*ChainClient, error) {
	cert, err := utils.ParseCert(userCrtBytes)
	if err != nil {
		return nil, err
	}

	priv, err := asym.PrivateKeyFromPEM(privKey, nil)
	if err != nil {
		return nil, err
	}

	chain := &ChainClient{
		pool:         pool,
		logger:       pool.getLogger(),
		chainId:      chainId,
		orgId:        orgId,
		userCrtBytes: userCrtBytes,
		userCrt:      cert,
		privateKey:   priv,
	}

	return chain, nil
}

// EnableAlias enable cert alias logic
func (cc *ChainClient) EnableAlias() error {
	var (
		err error
	)

	// 已经启用别名，直接返回
	if cc.enabledAlias {
		return nil
	}

	// 查询别名是否上链
	ok, err := cc.getCheckAlias()
	if err != nil {
		errMsg := fmt.Sprintf("enable alias, get and check alias failed, %s", err.Error())
		cc.logger.Debugf(sdkErrStringFormat, errMsg)
		//return errors.New(errMsg)
	}

	// 别名已上链
	if ok {
		cc.enabledAlias = true
		return nil
	}

	// 添加别名
	resp, err := cc.AddAlias()
	if err != nil {
		errMsg := fmt.Sprintf("enable alias AddAlias failed, %s", err.Error())
		cc.logger.Errorf(sdkErrStringFormat, errMsg)
		return errors.New(errMsg)
	}

	if err = utils.CheckProposalRequestResp(resp, true); err != nil {
		errMsg := fmt.Sprintf("enable alias AddAlias got invalid resp, %s", err.Error())
		cc.logger.Errorf(sdkErrStringFormat, errMsg)
		return errors.New(errMsg)
	}

	// 循环检查别名是否成功上链
	err = cc.checkAliasOnChain()
	if err != nil {
		errMsg := fmt.Sprintf("check alias on chain failed, %s", err.Error())
		cc.logger.Errorf(sdkErrStringFormat, errMsg)
		return errors.New(errMsg)
	}

	cc.enabledAlias = true

	return nil
}

func (cc *ChainClient) getCheckAlias() (bool, error) {
	aliasInfos, err := cc.QueryCertsAlias([]string{cc.alias})
	if err != nil {
		errMsg := fmt.Sprintf("QueryCertsAlias failed, %s", err.Error())
		cc.logger.Errorf(sdkErrStringFormat, errMsg)
		return false, errors.New(errMsg)
	}

	if len(aliasInfos.AliasInfos) != 1 {
		return false, errors.New("alias not found")
	}

	if aliasInfos.AliasInfos[0].Alias != cc.alias {
		return false, errors.New("alias not equal")
	}

	if aliasInfos.AliasInfos[0].NowCert.Cert == nil {
		return false, errors.New("alias has been deleted")
	}

	return true, nil
}

func (cc *ChainClient) checkAliasOnChain() error {
	err := retry.Retry(func(uint) error {
		ok, err := cc.getCheckAlias()
		if err != nil {
			errMsg := fmt.Sprintf("check alias on chain, get and check alias failed, %s", err.Error())
			cc.logger.Errorf(sdkErrStringFormat, errMsg)
			return errors.New(errMsg)
		}

		if !ok {
			errMsg := "alias havenot on chain yet, and try again"
			cc.logger.Debugf(sdkErrStringFormat, errMsg)
			return errors.New(errMsg)
		}

		return nil
	}, strategy.Limit(10), strategy.Wait(time.Second))

	if err != nil {
		errMsg := fmt.Sprintf("check upload alias on chain failed, try again later, %s", err.Error())
		cc.logger.Errorf(sdkErrStringFormat, errMsg)
		return errors.New(errMsg)
	}

	return nil
}
