package api

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

// TokenView API响应结构
type TokenViewResponse struct {
	Code  int    `json:"code"`
	Msg   string `json:"msg"`
	EnMsg string `json:"enMsg"`
	Data  struct {
		Type          string `json:"type"`
		Network       string `json:"network"`
		BlockNo       int64  `json:"block_no"`
		Height        int64  `json:"height"`
		BlockHash     string `json:"blockHash"`
		Index         int    `json:"index"`
		Time          int64  `json:"time"`
		Txid          string `json:"txid"`
		Fee           string `json:"fee"`
		Confirmations int64  `json:"confirmations"`
		From          string `json:"from"`
		To            string `json:"to"`
		Value         string `json:"value"` // 主交易金额，字符串类型
		GasPrice      int64  `json:"gasPrice"`
		GasLimit      int64  `json:"gasLimit"`
		GasUsed       int64  `json:"gasUsed"`
		TokenTransfer []struct {
			Index         int    `json:"index"`
			Token         string `json:"token"`
			TokenAddr     string `json:"tokenAddr"`
			TokenSymbol   string `json:"tokenSymbol"`
			TokenDecimals string `json:"tokenDecimals"`
			From          string `json:"from"`
			To            string `json:"to"`
			Value         string `json:"value"` // 代币转账金额
		} `json:"tokenTransfer"`
	} `json:"data"`
}

// API响应结构
type TransactionResponse struct {
	TxID   string
	Amount float64
	Status string
}

// API客户端
type Client struct {
	endpoint    string
	apiKey      string
	chainSymbol string
	httpClient  *http.Client
	
	// 速率限制相关
	rateLimiter *RateLimiter
}

// 速率限制器
type RateLimiter struct {
	maxRequests int           // 时间窗口内最大请求数
	timeWindow  time.Duration // 时间窗口
	requests    []time.Time   // 请求时间记录
	mu          sync.Mutex    // 互斥锁
}

// 创建速率限制器
func NewRateLimiter(maxRequests int, timeWindow time.Duration) *RateLimiter {
	return &RateLimiter{
		maxRequests: maxRequests,
		timeWindow:  timeWindow,
		requests:    make([]time.Time, 0),
	}
}

// 等待直到可以发送请求
func (rl *RateLimiter) Wait() {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	
	now := time.Now()
	
	// 清理过期的请求记录
	validRequests := make([]time.Time, 0)
	for _, reqTime := range rl.requests {
		if now.Sub(reqTime) < rl.timeWindow {
			validRequests = append(validRequests, reqTime)
		}
	}
	rl.requests = validRequests
	
	// 如果达到限制，等待直到最早的请求过期
	if len(rl.requests) >= rl.maxRequests {
		oldestReq := rl.requests[0]
		waitTime := rl.timeWindow - now.Sub(oldestReq)
		if waitTime > 0 {
			rl.mu.Unlock()
			time.Sleep(waitTime)
			rl.mu.Lock()
		}
		// 移除最早的请求
		rl.requests = rl.requests[1:]
	}
	
	// 记录当前请求
	rl.requests = append(rl.requests, now)
}

func NewClient(endpoint, apiKey, chainSymbol string) *Client {
	return &Client{
		endpoint:    endpoint,
		apiKey:      apiKey,
		chainSymbol: chainSymbol,
		httpClient: &http.Client{
			Timeout: 30 * time.Second, // 增加到30秒
		},
		// 设置每分钟299次的限制
		rateLimiter: NewRateLimiter(299, time.Minute),
	}
}

// 查询交易信息，带重试机制
func (c *Client) QueryTransaction(txid string) (*TransactionResponse, error) {
	maxRetries := 3
	var lastErr error
	
	for attempt := 0; attempt < maxRetries; attempt++ {
		if attempt > 0 {
			// 重试前等待，使用指数退避
			waitTime := time.Duration(attempt) * 2 * time.Second
			time.Sleep(waitTime)
		}
		
		result, err := c.queryTransactionOnce(txid)
		if err == nil {
			return result, nil
		}
		
		lastErr = err
		// 如果是非超时错误，不重试
		if !isTimeoutError(err) {
			break
		}
	}
	
	return nil, lastErr
}

// 判断是否为超时错误
func isTimeoutError(err error) bool {
	if err == nil {
		return false
	}
	errStr := err.Error()
	return strings.Contains(errStr, "timeout") || 
	       strings.Contains(errStr, "deadline exceeded") ||
	       strings.Contains(errStr, "connection refused")
}

// 单次查询交易信息
func (c *Client) queryTransactionOnce(txid string) (*TransactionResponse, error) {
	// 速率限制，等待直到可以发送请求
	c.rateLimiter.Wait()
	
	// 构建请求URL
	url := fmt.Sprintf("%s/%s/%s?apikey=%s", 
		c.endpoint, 
		c.chainSymbol, 
		txid, 
		c.apiKey)
	
	// 创建请求
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %v", err)
	}
	
	req.Header.Set("Content-Type", "application/json")
	
	// 发送请求
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("API请求失败: %v", err)
	}
	defer resp.Body.Close()
	
	// 读取响应体
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %v", err)
	}
	
	// 检查HTTP状态码
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API返回错误状态码: %d, 响应: %s", resp.StatusCode, string(body))
	}
	
	// 解析JSON响应
	var apiResp TokenViewResponse
	if err := json.Unmarshal(body, &apiResp); err != nil {
		return nil, fmt.Errorf("解析响应失败: %v, 响应内容: %s", err, string(body))
	}
	
	// 检查API返回的code
	if apiResp.Code != 1 {
		return nil, fmt.Errorf("API返回错误: code=%d, msg=%s", apiResp.Code, apiResp.Msg)
	}
	
	var amount float64
	
	// 优先使用代币转账金额，如果存在的话
	if len(apiResp.Data.TokenTransfer) > 0 {
		// 使用第一个代币转账记录
		tokenTransfer := apiResp.Data.TokenTransfer[0]
		
		// 解析代币金额
		tokenValue, err := strconv.ParseFloat(tokenTransfer.Value, 64)
		if err != nil {
			return nil, fmt.Errorf("解析代币金额失败: %v, 原始值: %s", err, tokenTransfer.Value)
		}
		
		// 解析代币精度
		decimals, err := strconv.Atoi(tokenTransfer.TokenDecimals)
		if err != nil {
			return nil, fmt.Errorf("解析代币精度失败: %v, 原始值: %s", err, tokenTransfer.TokenDecimals)
		}
		
		// 转换为实际金额，除以10^decimals
		divisor := 1.0
		for i := 0; i < decimals; i++ {
			divisor *= 10
		}
		amount = tokenValue / divisor
		
	} else {
		// 使用主交易金额
		var err error
		amount, err = strconv.ParseFloat(apiResp.Data.Value, 64)
		if err != nil {
			return nil, fmt.Errorf("解析金额失败: %v, 原始值: %s", err, apiResp.Data.Value)
		}
	}
	
	// 构建返回结果
	result := &TransactionResponse{
		TxID:   apiResp.Data.Txid,
		Amount: amount,
		Status: "success",
	}
	
	return result, nil
}

