# Blockchain_Verification

### This tool is used to communicate with the blockchain platform API to verify whether the hash value and amount of some of your own blockchain projects are correct.

## How to use?

### 1. Download

```
git clone https://github.com/XMRhapsody0807/Blockchain_Verification.git
```

### 2. Run

```
go run .
```

## Config.go Configuration
```
    APIEndpoint:  "https://services.tokenview.io/vipapi/tx",  //使用的区块链浏览器平台API
		APIKey:       "", //APIKEY
		ChainSymbol:  "bsc", // 默认比特币，可根据需要修改为eth、trx等  //hash地址是哪种类型
		CSVDirectory: "./data",  //读取csv文件的目录
		LogFile:      "./mismatch.log", //输出的log , 用于查询有哪些校验未通过
```
