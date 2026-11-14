package crypto

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type DetectedPayment struct {
	TxHash          string    `json:"tx_hash"`
	ToAddress       string    `json:"to_address"`
	FromAddress     string    `json:"from_address"`
	ContractAddress string    `json:"contract_address,omitempty"`
	Amount          float64   `json:"amount"`
	BlockNumber     int64     `json:"block_number"`
	Confirmations   int       `json:"confirmations"`
	Confirmed       bool      `json:"confirmed"`
	Timestamp       time.Time `json:"timestamp"`
}

type MultiChainRpcClient struct {
	httpClient *http.Client
}

func NewMultiChainRpcClient() *MultiChainRpcClient {
	return &MultiChainRpcClient{
		httpClient: &http.Client{
			Timeout: 8 * time.Second,
		},
	}
}

// ----------------------------------------------------------------------------
// EVM (Ethereum, BSC, Polygon) JSON-RPC
// ----------------------------------------------------------------------------

func (c *MultiChainRpcClient) callEVM(ctx context.Context, rpcURL, method string, params []interface{}) (json.RawMessage, error) {
	reqBody := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  method,
		"params":  params,
	}
	data, _ := json.Marshal(reqBody)
	req, err := http.NewRequestWithContext(ctx, "POST", rpcURL, bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var rpcResp struct {
		Result json.RawMessage `json:"result"`
		Error  *struct {
			Code    int    `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&rpcResp); err != nil {
		return nil, err
	}
	if rpcResp.Error != nil {
		return nil, fmt.Errorf("evm rpc error (%d): %s", rpcResp.Error.Code, rpcResp.Error.Message)
	}
	return rpcResp.Result, nil
}

func (c *MultiChainRpcClient) GetEVMCurrentBlock(ctx context.Context, rpcURL string) (int64, error) {
	raw, err := c.callEVM(ctx, rpcURL, "eth_blockNumber", []interface{}{})
	if err != nil {
		return 0, err
	}
	var hexStr string
	if err := json.Unmarshal(raw, &hexStr); err != nil {
		return 0, err
	}
	clean := strings.TrimPrefix(hexStr, "0x")
	return strconv.ParseInt(clean, 16, 64)
}

// ScanEVMTrc20OrErc20 checks for ERC20/BEP20 Transfer events to the deposit address.
func (c *MultiChainRpcClient) ScanEVMTokenDeposits(ctx context.Context, rpcURL, contractAddr, depositAddr string, decimals int, fromBlock int64) ([]*DetectedPayment, error) {
	currentBlock, err := c.GetEVMCurrentBlock(ctx, rpcURL)
	if err != nil {
		return nil, err
	}

	cleanContract := strings.ToLower(strings.TrimSpace(contractAddr))
	cleanDeposit := strings.ToLower(strings.TrimPrefix(strings.TrimSpace(depositAddr), "0x"))

	// Pad deposit address to 32 bytes for topic[2] (indexed recipient)
	paddedDeposit := "0x" + strings.Repeat("0", 64-len(cleanDeposit)) + cleanDeposit
	transferTopic := "0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef"

	// Look back up to 2000 blocks
	scanFrom := fromBlock
	if scanFrom <= 0 || currentBlock-scanFrom > 2000 {
		scanFrom = currentBlock - 2000
		if scanFrom < 0 {
			scanFrom = 0
		}
	}

	filterParams := map[string]interface{}{
		"address":   cleanContract,
		"fromBlock": fmt.Sprintf("0x%x", scanFrom),
		"toBlock":   "latest",
		"topics": []interface{}{
			transferTopic,
			nil, // any sender
			paddedDeposit,
		},
	}

	rawLogs, err := c.callEVM(ctx, rpcURL, "eth_getLogs", []interface{}{filterParams})
	if err != nil {
		return nil, err
	}

	var logs []struct {
		Address          string `json:"address"`
		BlockNumber      string `json:"blockNumber"`
		TransactionHash  string `json:"transactionHash"`
		Data             string `json:"data"`
		Topics           []string `json:"topics"`
	}
	if err := json.Unmarshal(rawLogs, &logs); err != nil {
		return nil, err
	}

	payments := make([]*DetectedPayment, 0, len(logs))
	for _, l := range logs {
		if !strings.EqualFold(l.Address, cleanContract) {
			continue
		}
		blk, _ := strconv.ParseInt(strings.TrimPrefix(l.BlockNumber, "0x"), 16, 64)
		confs := int(currentBlock - blk + 1)
		if confs < 0 {
			confs = 0
		}

		rawHex := strings.TrimPrefix(l.Data, "0x")
		valBig := new(big.Int)
		valBig.SetString(rawHex, 16)

		// Convert token units to float using decimals
		fVal := new(big.Float).SetInt(valBig)
		divider := new(big.Float).SetFloat64(float64(new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(decimals)), nil).Int64()))
		tokenAmount, _ := new(big.Float).Quo(fVal, divider).Float64()

		fromAddr := ""
		if len(l.Topics) >= 2 {
			fromRaw := strings.TrimPrefix(l.Topics[1], "0x")
			if len(fromRaw) >= 40 {
				fromAddr = "0x" + fromRaw[len(fromRaw)-40:]
			}
		}

		payments = append(payments, &DetectedPayment{
			TxHash:          l.TransactionHash,
			ToAddress:       depositAddr,
			FromAddress:     fromAddr,
			ContractAddress: contractAddr,
			Amount:          tokenAmount,
			BlockNumber:     blk,
			Confirmations:   confs,
			Confirmed:       confs >= 2,
			Timestamp:       time.Now().UTC(),
		})
	}

	return payments, nil
}

// ----------------------------------------------------------------------------
// TRON (TRC20 USDT & TRX)
// ----------------------------------------------------------------------------

func (c *MultiChainRpcClient) ScanTronTRC20Deposits(ctx context.Context, rpcURL, contractAddr, depositAddr string, minConfirmations int) ([]*DetectedPayment, error) {
	baseURL := rpcURL
	if baseURL == "" {
		baseURL = "https://api.trongrid.io"
	}
	url := fmt.Sprintf("%s/v1/accounts/%s/transactions/trc20?limit=20&contract_address=%s", baseURL, depositAddr, contractAddr)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var trc20Resp struct {
		Success bool `json:"success"`
		Data    []struct {
			TransactionID string `json:"transaction_id"`
			TokenInfo     struct {
				Symbol   string `json:"symbol"`
				Address  string `json:"address"`
				Decimals int    `json:"decimals"`
			} `json:"token_info"`
			From            string `json:"from"`
			To              string `json:"to"`
			Type            string `json:"type"`
			Value           string `json:"value"`
			BlockTimestamp  int64  `json:"block_timestamp"`
		} `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&trc20Resp); err != nil {
		return nil, err
	}

	payments := make([]*DetectedPayment, 0, len(trc20Resp.Data))
	for _, tx := range trc20Resp.Data {
		if !strings.EqualFold(tx.To, depositAddr) {
			continue
		}
		if contractAddr != "" && !strings.EqualFold(tx.TokenInfo.Address, contractAddr) {
			continue
		}

		valBig := new(big.Int)
		valBig.SetString(tx.Value, 10)
		decimals := tx.TokenInfo.Decimals
		if decimals == 0 {
			decimals = 6
		}

		fVal := new(big.Float).SetInt(valBig)
		divider := new(big.Float).SetFloat64(float64(new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(decimals)), nil).Int64()))
		tokenAmount, _ := new(big.Float).Quo(fVal, divider).Float64()

		txTime := time.UnixMilli(tx.BlockTimestamp).UTC()
		// TRON blocks are produced every 3s. Confirmations can be estimated or queried.
		// If transaction is in TronGrid, it has at least 1 confirmation. After ~60 seconds it is solid (19+ blocks).
		confs := int(time.Since(txTime).Seconds() / 3)
		if confs < 1 {
			confs = 1
		}

		payments = append(payments, &DetectedPayment{
			TxHash:          tx.TransactionID,
			ToAddress:       depositAddr,
			FromAddress:     tx.From,
			ContractAddress: tx.TokenInfo.Address,
			Amount:          tokenAmount,
			BlockNumber:     0,
			Confirmations:   confs,
			Confirmed:       confs >= minConfirmations,
			Timestamp:       txTime,
		})
	}

	return payments, nil
}

// ----------------------------------------------------------------------------
// Bitcoin & Litecoin (Mempool / Blockstream API)
// ----------------------------------------------------------------------------

func (c *MultiChainRpcClient) ScanBitcoinDeposits(ctx context.Context, apiURL, depositAddr string, minConfirmations int) ([]*DetectedPayment, error) {
	baseURL := apiURL
	if baseURL == "" {
		baseURL = "https://blockstream.info/api"
	}
	url := fmt.Sprintf("%s/address/%s/txs", baseURL, depositAddr)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var txs []struct {
		TxID   string `json:"txid"`
		Status struct {
			Confirmed   bool  `json:"confirmed"`
			BlockHeight int64 `json:"block_height"`
			BlockTime   int64 `json:"block_time"`
		} `json:"status"`
		Vout []struct {
			ScriptPubKeyAddress string `json:"scriptpubkey_address"`
			Value               int64  `json:"value"` // in satoshis
		} `json:"vout"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&txs); err != nil {
		return nil, err
	}

	payments := make([]*DetectedPayment, 0)
	for _, tx := range txs {
		var receivedSatoshis int64
		for _, out := range tx.Vout {
			if strings.EqualFold(out.ScriptPubKeyAddress, depositAddr) {
				receivedSatoshis += out.Value
			}
		}
		if receivedSatoshis <= 0 {
			continue
		}

		amountBTC := float64(receivedSatoshis) / 1e8
		confs := 0
		if tx.Status.Confirmed {
			confs = 3 // Standard confirmed block
		}

		payments = append(payments, &DetectedPayment{
			TxHash:        tx.TxID,
			ToAddress:     depositAddr,
			Amount:        amountBTC,
			BlockNumber:   tx.Status.BlockHeight,
			Confirmations: confs,
			Confirmed:     tx.Status.Confirmed && confs >= minConfirmations,
			Timestamp:     time.Unix(tx.Status.BlockTime, 0).UTC(),
		})
	}

	return payments, nil
}
