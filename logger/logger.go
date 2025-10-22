package logger

import (
	"fmt"
	"os"
	"time"
)

// 不匹配记录
type MismatchRecord struct {
	TxID           string
	CSVAmount      float64
	APIAmount      float64
	Difference     float64
	Row            int
	CheckTime      time.Time
}

// 查询失败记录
type FailedRecord struct {
	TxID      string
	Row       int
	Reason    string
	CheckTime time.Time
}

// 日志记录器
type Logger struct {
	logFile string
	file    *os.File
}

func NewLogger(logFile string) (*Logger, error) {
	file, err := os.OpenFile(logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return nil, fmt.Errorf("创建日志文件失败: %v", err)
	}

	logger := &Logger{
		logFile: logFile,
		file:    file,
	}

	// 写入日志头
	logger.writeHeader()

	return logger, nil
}

func (l *Logger) writeHeader() {
	header := fmt.Sprintf("\n=== 金额核对日志 开始时间: %s ===\n\n", time.Now().Format("2006-01-02 15:04:05"))
	l.file.WriteString(header)
}

// 记录不匹配的交易
func (l *Logger) LogMismatch(record MismatchRecord) error {
	if _, err := l.file.WriteString("【金额不匹配】\n"); err != nil {
		return err
	}
	
	logLine := fmt.Sprintf("  交易ID: %s\n", record.TxID)
	logLine += fmt.Sprintf("  CSV金额: %.8f\n", record.CSVAmount)
	logLine += fmt.Sprintf("  API金额: %.8f\n", record.APIAmount)
	logLine += fmt.Sprintf("  差额: %.8f\n", record.Difference)
	logLine += fmt.Sprintf("  行号: 第%d行\n", record.Row)
	logLine += fmt.Sprintf("  检查时间: %s\n\n", record.CheckTime.Format("2006-01-02 15:04:05"))
	
	_, err := l.file.WriteString(logLine)
	return err
}

// 记录查询失败的交易
func (l *Logger) LogFailed(record FailedRecord) error {
	if _, err := l.file.WriteString("【查询失败】\n"); err != nil {
		return err
	}
	
	logLine := fmt.Sprintf("  交易ID: %s\n", record.TxID)
	logLine += fmt.Sprintf("  失败原因: %s\n", record.Reason)
	logLine += fmt.Sprintf("  行号: 第%d行\n", record.Row)
	logLine += fmt.Sprintf("  检查时间: %s\n\n", record.CheckTime.Format("2006-01-02 15:04:05"))
	
	_, err := l.file.WriteString(logLine)
	return err
}

// 写入汇总信息
func (l *Logger) WriteSummary(total, success, mismatch, failed int) error {
	summary := fmt.Sprintf("\n=== 核对完成 ===\n")
	summary += fmt.Sprintf("总交易数: %d\n", total)
	summary += fmt.Sprintf("成功核对: %d\n", success)
	summary += fmt.Sprintf("金额不匹配: %d\n", mismatch)
	summary += fmt.Sprintf("查询失败: %d\n", failed)
	if success > 0 {
		summary += fmt.Sprintf("匹配率: %.2f%%\n", float64(success-mismatch)/float64(success)*100)
	}
	summary += fmt.Sprintf("结束时间: %s\n\n", time.Now().Format("2006-01-02 15:04:05"))
	
	_, err := l.file.WriteString(summary)
	return err
}

// 关闭日志文件
func (l *Logger) Close() error {
	if l.file != nil {
		return l.file.Close()
	}
	return nil
}



