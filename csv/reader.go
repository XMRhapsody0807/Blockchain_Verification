package csv

import (
	"encoding/csv"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
)

// CSV记录结构
type Transaction struct {
	TxID   string
	Amount float64
	Row    int // 记录所在行号，方便定位
}

// CSV读取器
type Reader struct {
	directory string
}

func NewReader(directory string) *Reader {
	return &Reader{directory: directory}
}

// 读取目录下所有CSV文件
func (r *Reader) ReadAllCSV() ([]Transaction, error) {
	var allTransactions []Transaction

	// 遍历目录
	files, err := filepath.Glob(filepath.Join(r.directory, "*.csv"))
	if err != nil {
		return nil, fmt.Errorf("读取目录失败: %v", err)
	}

	if len(files) == 0 {
		return nil, fmt.Errorf("目录 %s 下没有找到CSV文件", r.directory)
	}

	// 逐个处理CSV文件
	for _, file := range files {
		transactions, err := r.readSingleCSV(file)
		if err != nil {
			return nil, fmt.Errorf("读取文件 %s 失败: %v", file, err)
		}
		allTransactions = append(allTransactions, transactions...)
	}

	return allTransactions, nil
}

// 读取单个CSV文件
func (r *Reader) readSingleCSV(filename string) ([]Transaction, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	reader := csv.NewReader(file)
	records, err := reader.ReadAll()
	if err != nil {
		return nil, err
	}

	if len(records) == 0 {
		return nil, fmt.Errorf("CSV文件为空")
	}

	// 查找txid和amount列的索引
	header := records[0]
	txidIndex := -1
	amountIndex := -1

	for i, col := range header {
		if col == "tx_id" {
			txidIndex = i
		} else if col == "amount" {
			amountIndex = i
		}
	}

	if txidIndex == -1 || amountIndex == -1 {
		return nil, fmt.Errorf("CSV文件缺少必需的列，需要txid和amount列")
	}

	// 解析数据行
	var transactions []Transaction
	for i := 1; i < len(records); i++ {
		record := records[i]
		
		if len(record) <= txidIndex || len(record) <= amountIndex {
			continue
		}

		amount, err := strconv.ParseFloat(record[amountIndex], 64)
		if err != nil {
			// 跳过无效的金额
			continue
		}

		transactions = append(transactions, Transaction{
			TxID:   record[txidIndex],
			Amount: amount,
			Row:    i + 1, // Excel行号，从1开始且包含表头
		})
	}

	return transactions, nil
}


