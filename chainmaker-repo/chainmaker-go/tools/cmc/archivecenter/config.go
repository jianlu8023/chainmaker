package archivecenter

import (
	"fmt"

	"github.com/spf13/viper"
)

var (
	archiveCenterCFG *ArchiveCenterConfig
)

type ArchiveCenterConfig struct {
	ChainGenesisHash       string    `mapstructure:"chain_genesis_hash"`
	RpcAddress             string    `mapstructure:"rpc_address"`
	HttpAddress            string    `mapstructure:"archive_center_http_url"`
	HttpRequestSecondLimit int       `mapstructure:"request_second_limit"`
	TlsEnable              bool      `mapstructure:"tls_enable"`
	Tls                    TlsConfig `mapstructure:"tls"`
	MaxSendMsgSize         int       `mapstructure:"max_send_msg_size"`
	MaxRecvMsgSize         int       `mapstructure:"max_recv_msg_size"`
}

type TlsConfig struct {
	ServerName  string   `mapstructure:"server_name"`
	PrivKeyFile string   `mapstructure:"priv_key_file"`
	CertFile    string   `mapstructure:"cert_file"`
	TrustCaList []string `mapstructure:"trust_ca_list"`
}

func ReadConfigFile(confPath string) error {
	var (
		err       error
		confViper *viper.Viper
	)
	if confViper, err = initViper(confPath); err != nil {
		return fmt.Errorf("load sdk config failed, %s", err)
	}
	archiveCenterCFG = &ArchiveCenterConfig{}
	if err = confViper.Unmarshal(archiveCenterCFG); err != nil {
		return fmt.Errorf("unmarshal config file failed, %s", err)
	}
	//fmt.Fprintf(os.Stdout, "config is : %+v ", archiveCenterCFG)

	if err != nil {
		return err
	}
	return nil
}

// initViper 加载配置文件，解析
// @param string
// @return *viper.Viper
// @return error
func initViper(confPath string) (*viper.Viper, error) {
	cmViper := viper.New()
	cmViper.SetConfigFile(confPath)
	err := cmViper.ReadInConfig()
	if err != nil {
		return nil, err
	}

	return cmViper, nil
}
