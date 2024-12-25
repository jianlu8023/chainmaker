# 学习 整理 归档功能

## 准备工作

1. 启动长安链baas chainmaker-baas
   cd chainmaker-baas && docker-compose up -d
   目前 测试没有使用自带的mysql

2. 部署一条测试链

3. 打包安装fact智能合约
   cd chainmaker-chaincode/fact/ && bash build.sh fact

4. 调用上链程序 在链上造数据

5. archive-center 准备

tips:

* 此部分已经进行调整，可直接在 chainmaker-repo/chainmaker-archive-service/ 执行 make build
* 针对chainmaker-archive-service 添加health接口，用于服务状态检测，进行容器化部署
* make build 后 在 chainmaker-archivecenter 中 修改image版本号 docker-compose up -d 启动即可

```shell
git clone -v v1.0.0_alpha --depth=1 https://git.chainmaker.org.cn/chainmaker/chainmaker-archive-service.git
cd chainmaker-archive-service/src && go build -o chainmaker-archive-service
```

6. 编译cmc工具

```shell
git clone -v v2.3.0_archive_alpha --depth=1 https://git.chainmaker.org.cn/chainmaker/chainmaker-go.git
cd chainmaker-go/tools/cmc && go build -o cmc
```

7. 归档后使用的sdk

```shell
git clone -b v2.3.0_archive_alpha --depth=1 https://git.chainmaker.org.cn/chainmaker/sdk-go.git
```

## 步骤

1. 启动archive-center

```shell
cd chainmaker-archivecenter && docker-compose up -d 
```

2. 获取创世区块

tips:

* 已经使用代码获取 代码位置 myself-archivecenter/archive-center/cmd/genesisblock/main.go

正常情况获取:

git clone -b v2.3.0_archive_alpha --depth=1 https://git.chainmaker.org.cn/chainmaker/chainmaker-go.git
cd chainaker-go/tools/cmc
go build -o cmc

若archivecenter使用默认配置 即可开始后续操作，若archivecenter修改tls配置，则需要将tls的client证书和私钥复制到cmc的archivecenter文件夹下

将测试链的链配置文件拷贝到cmc文件夹下 并解压
完整路经: chainmaker-go/tools/cmc/sdk_configs

调整config文件
必须修改的配置
chain_genesis_hash 创世区块的hash 第0块的区块hash

使用cmc工具获取创世区块hash步骤

pwd:
chainmaker-go/tools/cmc

命令:

// 首先使用cmc从链上查询创世块的blockhash
./cmc query block-by-height 0 --sdk-conf-path ./sdk_configs/sdk_config.yml

// 调用归档中心http接口,获取hex编码的genesis_block_hash,其中[blockhash]即为上面cmc查询出来的base64编码的hash
curl -X POST 'http://127.0.0.1:13119/get_hashhex_by_hashbyte' -d '{"block_hash": [blockhash]}'
curl -X POST http://127.0.0.1:13119/get_hashhex_by_hashbyte -d '{"block_hash":"G58UfXtBIF7aC5hkWmO3V0SQK7ZA4FQ5df6KZOwp8ek="}'

3. 归档数据


