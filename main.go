package main

import (
	"fmt"
	"log"
	"math"
	"os"
	"time"

	"test/api"
	"test/config"
	"test/csv"
	"test/logger"
)

func main() {
	fmt.Println("=== 交易金额核对工具 ===")
	
	// 加载配置
	cfg := config.GetDefaultConfig()
	
	// 检查数据目录是否存在
	if _, err := os.Stat(cfg.CSVDirectory); os.IsNotExist(err) {
		if err := os.MkdirAll(cfg.CSVDirectory, 0755); err != nil {
			log.Fatalf("创建数据目录失败: %v", err)
		}
		fmt.Printf("已创建数据目录: %s\n", cfg.CSVDirectory)
		fmt.Println("请将CSV文件放入该目录后重新运行程序")
		return
	}
	
	// 初始化CSV读取器
	csvReader := csv.NewReader(cfg.CSVDirectory)
	
	// 读取所有CSV文件
	fmt.Printf("正在读取 %s 目录下的CSV文件...\n", cfg.CSVDirectory)
	transactions, err := csvReader.ReadAllCSV()
	if err != nil {
		log.Fatalf("读取CSV文件失败: %v", err)
	}
	fmt.Printf("成功读取 %d 条交易记录\n", len(transactions))
	
	// 初始化API客户端
	apiClient := api.NewClient(cfg.APIEndpoint, cfg.APIKey, cfg.ChainSymbol)
	fmt.Printf("API配置: %s (公链: %s)\n", cfg.APIEndpoint, cfg.ChainSymbol)
	
	// 初始化日志记录器
	logWriter, err := logger.NewLogger(cfg.LogFile)
	if err != nil {
		log.Fatalf("初始化日志记录器失败: %v", err)
	}
	defer logWriter.Close()
	
	// 开始核对
	fmt.Println("\n开始核对交易金额...")
	fmt.Println("按Ctrl+C可随时中断程序\n")
	
	mismatchCount := 0
	failedCount := 0
	successCount := 0
	
	for i, tx := range transactions {
		// 显示进度
		if (i+1)%10 == 0 || i == len(transactions)-1 {
			fmt.Printf("\r进度: %d/%d (成功:%d 不匹配:%d 失败:%d)", 
				i+1, len(transactions), successCount, mismatchCount, failedCount)
		}
		
		// 调用API查询交易
		apiResp, err := apiClient.QueryTransaction(tx.TxID)
		if err != nil {
			// API查询失败时，记录到日志
			failedRecord := logger.FailedRecord{
				TxID:      tx.TxID,
				Row:       tx.Row,
				Reason:    err.Error(),
				CheckTime: time.Now(),
			}
			
			if err := logWriter.LogFailed(failedRecord); err != nil {
				log.Printf("写入日志失败: %v", err)
			}
			
			failedCount++
			continue
		}
		
		// 核对金额，允许小误差
		tolerance := 0.00000001 // 容差值，避免浮点数精度问题
		diff := math.Abs(tx.Amount - apiResp.Amount)
		
		if diff > tolerance {
			// 金额不匹配，记录到日志
			mismatchRecord := logger.MismatchRecord{
				TxID:       tx.TxID,
				CSVAmount:  tx.Amount,
				APIAmount:  apiResp.Amount,
				Difference: tx.Amount - apiResp.Amount,
				Row:        tx.Row,
				CheckTime:  time.Now(),
			}
			
			if err := logWriter.LogMismatch(mismatchRecord); err != nil {
				log.Printf("写入日志失败: %v", err)
			}
			
			mismatchCount++
		}
		
		successCount++
	}
	
	fmt.Println() // 换行
	
	// 写入汇总信息
	if err := logWriter.WriteSummary(len(transactions), successCount, mismatchCount, failedCount); err != nil {
		log.Printf("写入汇总信息失败: %v", err)
	}
	
	// 输出结果
	fmt.Println("\n=== 核对完成 ===")
	fmt.Printf("总交易数: %d\n", len(transactions))
	fmt.Printf("成功核对: %d\n", successCount)
	fmt.Printf("  其中匹配: %d\n", successCount-mismatchCount)
	fmt.Printf("  金额不匹配: %d\n", mismatchCount)
	fmt.Printf("查询失败: %d\n", failedCount)
	
	if mismatchCount > 0 || failedCount > 0 {
		fmt.Printf("\n详细信息已记录到: %s\n", cfg.LogFile)
	} else {
		fmt.Println("\n所有交易金额核对一致！")
	}
}
