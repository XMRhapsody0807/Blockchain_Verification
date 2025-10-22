package config

type Config struct {
	// API配置
	APIEndpoint string
	APIKey      string
	
	// 公链简称，如btc、eth等
	ChainSymbol string
	
	// CSV文件路径
	CSVDirectory string
	
	// 日志文件路径
	LogFile string
}

// 获取默认配置
func GetDefaultConfig() *Config {
	return &Config{
		APIEndpoint:  "https://services.tokenview.io/vipapi/tx",
		APIKey:       "",
		ChainSymbol:  "bsc", // 默认比特币，可根据需要修改为eth、trx等
		CSVDirectory: "./data",
		LogFile:      "./mismatch.log",
	}
}

