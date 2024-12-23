module chainmaker.org/chainmaker-archive-service

go 1.16

replace chainmaker.org/chainmaker/pb-go/v2 v2.3.0 => chainmaker.org/chainmaker/pb-go/v2 v2.3.2-0.20221212031024-fc4ec021d4ca

require (
	chainmaker.org/chainmaker/common/v2 v2.3.0
	chainmaker.org/chainmaker/lws v1.1.0
	chainmaker.org/chainmaker/pb-go/v2 v2.3.0
	chainmaker.org/chainmaker/protocol/v2 v2.3.0
	chainmaker.org/chainmaker/store-leveldb/v2 v2.3.1
	chainmaker.org/chainmaker/store/v2 v2.3.1
	chainmaker.org/chainmaker/utils/v2 v2.3.0
	github.com/gin-contrib/gzip v0.0.6
	github.com/gin-gonic/gin v1.8.1
	github.com/gogo/protobuf v1.3.2
	github.com/google/uuid v1.3.0 // indirect
	github.com/grpc-ecosystem/go-grpc-middleware v1.3.0
	github.com/mitchellh/mapstructure v1.4.2
	github.com/pkg/errors v0.9.1
	github.com/prometheus/client_golang v1.9.0
	github.com/spf13/viper v1.9.0
	github.com/stretchr/testify v1.8.0
	github.com/syndtr/goleveldb v1.0.1-0.20200815110645-5c35d600f0ca
	github.com/tidwall/tinylru v1.1.0
	go.uber.org/zap v1.17.0
	golang.org/x/text v0.3.7 // indirect
	golang.org/x/time v0.0.0-20220722155302-e5dcc9cfc0b9
	google.golang.org/grpc v1.40.0
	gopkg.in/natefinch/lumberjack.v2 v2.0.0
)
