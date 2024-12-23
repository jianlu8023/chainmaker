# cmc 归档中心命令

## cmc归档数据命令
```powershell
./cmc archivecenter dump --sdk-conf-path ./testdata/sdk_config.yml --chain-id chain1 --archive-conf-path ./archivecenter/config.yml --archive-begin-height 1 --archive-end-height 610 --mode quick

```
## cmc查询归档中心当前已归档数据状态命令
```powershell
./cmc archivecenter  query archived-status --archive-conf-path archivecenter/config.yml
```

## cmc查询归档中心,根据高度查区块命令
```powershell
./cmc archivecenter query block-by-height 20 --archive-conf-path ./archivecenter/config.yml

```

## cmc查询归档中心,根据tx查询区块命令
```powershell
./cmc archivecenter query block-by-txid 17221b132a25209aca52fdfc07218265e4377ef0099d46a49edfd032001fc2be --archive-conf-path ./archivecenter/config.yml
```

## cmc查询归档中心,根据tx查询事务命令
```powershell
./cmc archivecenter query tx 17221b132a25209aca52fdfc07218265e4377ef0099d46a49edfd032001fc2be --archive-conf-path ./archivecenter/config.yml

```

## cmc查询归档中心,根据hash查询区块命令

```powershell
./cmc archivecenter query block-by-hash KBHde9w/uHoYV+747CS3d6AzLjJ+FQ7ZlX6zEvjqG30=  --archive-conf-path ./archivecenter/config.yml

```