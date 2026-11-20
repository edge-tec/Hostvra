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
			Confirmed:       confs >= 1,
			Timestamp:       time.Now().UTC(),
		})
	}

	return payments, nil
}

// GetEVMBalance queries the native coin balance (BNB, ETH, POL) in float units.
func (c *MultiChainRpcClient) GetEVMBalance(ctx context.Context, rpcURL, address string) (float64, error) {
	cleanAddr := strings.TrimSpace(address)
	raw, err := c.callEVM(ctx, rpcURL, "eth_getBalance", []interface{}{cleanAddr, "latest"})
	if err != nil {
		return 0, err
	}
	var hexStr string
	if err := json.Unmarshal(raw, &hexStr); err != nil {
		return 0, err
	}
	clean := strings.TrimPrefix(hexStr, "0x")
	valBig := new(big.Int)
	valBig.SetString(clean, 16)

	fVal := new(big.Float).SetInt(valBig)
	divider := new(big.Float).SetFloat64(1e18)
	balance, _ := new(big.Float).Quo(fVal, divider).Float64()
	return balance, nil
}

// GetEVMTokenBalance queries the ERC20/BEP20 token balance.
func (c *MultiChainRpcClient) GetEVMTokenBalance(ctx context.Context, rpcURL, contractAddr, address string, decimals int) (float64, error) {
	cleanContract := strings.ToLower(strings.TrimSpace(contractAddr))
	cleanAddr := strings.ToLower(strings.TrimPrefix(strings.TrimSpace(address), "0x"))
	paddedAddr := strings.Repeat("0", 64-len(cleanAddr)) + cleanAddr
	callData := "0x70a08231" + paddedAddr // balanceOf(address) selector

	raw, err := c.callEVM(ctx, rpcURL, "eth_call", []interface{}{
		map[string]interface{}{
			"to":   cleanContract,
			"data": callData,
		},
		"latest",
	})
	if err != nil {
		return 0, err
	}
	var hexStr string
	if err := json.Unmarshal(raw, &hexStr); err != nil {
		return 0, err
	}
	clean := strings.TrimPrefix(hexStr, "0x")
	valBig := new(big.Int)
	valBig.SetString(clean, 16)

	fVal := new(big.Float).SetInt(valBig)
	divider := new(big.Float).SetFloat64(float64(new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(decimals)), nil).Int64()))
	balance, _ := new(big.Float).Quo(fVal, divider).Float64()
	return balance, nil
}

// VerifyEVMTransaction directly queries on-chain transaction status, recipient, and amount.
func (c *MultiChainRpcClient) VerifyEVMTransaction(ctx context.Context, rpcURL, txHash, depositAddr, contractAddr string, decimals int, isToken bool, expectedAmount float64) (*DetectedPayment, error) {
	cleanTxHash := strings.TrimSpace(txHash)
	if !strings.HasPrefix(cleanTxHash, "0x") {
		cleanTxHash = "0x" + cleanTxHash
	}

	// 1. Query transaction details
	rawTx, err := c.callEVM(ctx, rpcURL, "eth_getTransactionByHash", []interface{}{cleanTxHash})
	if err != nil {
		return nil, err
	}
	if len(rawTx) == 0 || string(rawTx) == "null" {
		return nil, fmt.Errorf("transaction not found on blockchain")
	}

	var tx struct {
		Hash        string `json:"hash"`
		From        string `json:"from"`
		To          string `json:"to"`
		Value       string `json:"value"`
		Input       string `json:"input"`
		BlockNumber string `json:"blockNumber"`
	}
	if err := json.Unmarshal(rawTx, &tx); err != nil {
		return nil, err
	}

	// 2. Query transaction receipt for status
	rawReceipt, err := c.callEVM(ctx, rpcURL, "eth_getTransactionReceipt", []interface{}{cleanTxHash})
	if err != nil || len(rawReceipt) == 0 || string(rawReceipt) == "null" {
		return nil, fmt.Errorf("transaction is still pending in mempool (awaiting block inclusion)")
	}

	var receipt struct {
		Status      string `json:"status"` // "0x1" = success, "0x0" = failure
		BlockNumber string `json:"blockNumber"`
		Logs        []struct {
			Address string   `json:"address"`
			Topics  []string `json:"topics"`
			Data    string   `json:"data"`
		} `json:"logs"`
	}
	if err := json.Unmarshal(rawReceipt, &receipt); err != nil {
		return nil, err
	}

	if receipt.Status == "0x0" {
		return nil, fmt.Errorf("transaction failed/reverted on blockchain")
	}

	txBlock, _ := strconv.ParseInt(strings.TrimPrefix(receipt.BlockNumber, "0x"), 16, 64)
	currentBlock, _ := c.GetEVMCurrentBlock(ctx, rpcURL)
	confirmations := int(currentBlock - txBlock + 1)
	if confirmations < 1 {
		confirmations = 1
	}

	cleanDeposit := strings.ToLower(strings.TrimSpace(depositAddr))

	if !isToken {
		// Native coin transfer (BNB, ETH, POL)
		if !strings.EqualFold(tx.To, cleanDeposit) {
			return nil, fmt.Errorf("transaction recipient (%s) does not match deposit address (%s)", tx.To, cleanDeposit)
		}
		rawHex := strings.TrimPrefix(tx.Value, "0x")
		valBig := new(big.Int)
		valBig.SetString(rawHex, 16)
		fVal := new(big.Float).SetInt(valBig)
		divider := new(big.Float).SetFloat64(1e18)
		amount, _ := new(big.Float).Quo(fVal, divider).Float64()

		minExpected := expectedAmount * 0.97
		if amount < minExpected {
			return nil, fmt.Errorf("transaction amount (%.8f) is less than required (%.8f)", amount, expectedAmount)
		}

		return &DetectedPayment{
			TxHash:        tx.Hash,
			ToAddress:     depositAddr,
			FromAddress:   tx.From,
			Amount:        amount,
			BlockNumber:   txBlock,
			Confirmations: confirmations,
			Confirmed:     confirmations >= 1,
			Timestamp:     time.Now().UTC(),
		}, nil
	}

	// Token transfer (USDT BEP20/ERC20)
	cleanContract := strings.ToLower(strings.TrimSpace(contractAddr))
	transferTopic := "0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef"

	for _, l := range receipt.Logs {
		if !strings.EqualFold(l.Address, cleanContract) {
			continue
		}
		if len(l.Topics) >= 3 && strings.EqualFold(l.Topics[0], transferTopic) {
			toTopic := strings.TrimPrefix(l.Topics[2], "0x")
			if len(toTopic) >= 40 {
				logTo := "0x" + toTopic[len(toTopic)-40:]
				if strings.EqualFold(logTo, cleanDeposit) {
					rawHex := strings.TrimPrefix(l.Data, "0x")
					valBig := new(big.Int)
					valBig.SetString(rawHex, 16)
					fVal := new(big.Float).SetInt(valBig)
					div := new(big.Float).SetFloat64(float64(new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(decimals)), nil).Int64()))
					amount, _ := new(big.Float).Quo(fVal, div).Float64()

					minExpected := expectedAmount * 0.97
					if amount >= minExpected {
						fromAddr := ""
						fromTopic := strings.TrimPrefix(l.Topics[1], "0x")
						if len(fromTopic) >= 40 {
							fromAddr = "0x" + fromTopic[len(fromTopic)-40:]
						}
						return &DetectedPayment{
							TxHash:          tx.Hash,
							ToAddress:       depositAddr,
							FromAddress:     fromAddr,
							ContractAddress: contractAddr,
							Amount:          amount,
							BlockNumber:     txBlock,
							Confirmations:   confirmations,
							Confirmed:       confirmations >= 1,
							Timestamp:       time.Now().UTC(),
						}, nil
					}
				}
			}
		}
	}

	return nil, fmt.Errorf("no matching token transfer to %s found in transaction %s", depositAddr, cleanTxHash)
}

// ScanEVMLatestBlocks scans transactions in recent blocks for native transfers to depositAddr.
func (c *MultiChainRpcClient) ScanEVMLatestBlocks(ctx context.Context, rpcURL, depositAddr string, maxBlocks int, expectedAmount float64) (*DetectedPayment, error) {
	currentBlock, err := c.GetEVMCurrentBlock(ctx, rpcURL)
	if err != nil {
		return nil, err
	}

	if maxBlocks <= 0 || maxBlocks > 30 {
		maxBlocks = 20
	}

	cleanDeposit := strings.ToLower(strings.TrimSpace(depositAddr))
	minExpected := expectedAmount * 0.97

	for blk := currentBlock; blk >= currentBlock-int64(maxBlocks) && blk > 0; blk-- {
		rawBlock, err := c.callEVM(ctx, rpcURL, "eth_getBlockByNumber", []interface{}{fmt.Sprintf("0x%x", blk), true})
		if err != nil {
			continue
		}

		var blockData struct {
			Transactions []struct {
				Hash  string `json:"hash"`
				From  string `json:"from"`
				To    string `json:"to"`
				Value string `json:"value"`
			} `json:"transactions"`
		}
		if err := json.Unmarshal(rawBlock, &blockData); err != nil {
			continue
		}

		for _, tx := range blockData.Transactions {
			if strings.EqualFold(tx.To, cleanDeposit) {
				rawHex := strings.TrimPrefix(tx.Value, "0x")
				valBig := new(big.Int)
				valBig.SetString(rawHex, 16)
				fVal := new(big.Float).SetInt(valBig)
				divider := new(big.Float).SetFloat64(1e18)
				amount, _ := new(big.Float).Quo(fVal, divider).Float64()

				if amount >= minExpected {
					confs := int(currentBlock - blk + 1)
					if confs < 1 {
						confs = 1
					}
					return &DetectedPayment{
						TxHash:        tx.Hash,
						ToAddress:     depositAddr,
						FromAddress:   tx.From,
						Amount:        amount,
						BlockNumber:   blk,
						Confirmations: confs,
						Confirmed:     confs >= 1,
						Timestamp:     time.Now().UTC(),
					}, nil
				}
			}
		}
	}

	return nil, nil
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

// GetTronBalance queries native TRX balance of an address in TRX.
func (c *MultiChainRpcClient) GetTronBalance(ctx context.Context, rpcURL, depositAddr string) (float64, error) {
	baseURL := rpcURL
	if baseURL == "" {
		baseURL = "https://api.trongrid.io"
	}
	url := fmt.Sprintf("%s/v1/accounts/%s", baseURL, depositAddr)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return 0, err
	}
	req.Header.Set("Accept", "application/json")
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	var accResp struct {
		Data []struct {
			Balance int64 `json:"balance"` // in SUN
		} `json:"data"`
		Success bool `json:"success"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&accResp); err != nil {
		return 0, err
	}
	if len(accResp.Data) == 0 {
		return 0, nil
	}
	return float64(accResp.Data[0].Balance) / 1e6, nil
}

// VerifyTronTransaction directly verifies a TRON transaction hash.
func (c *MultiChainRpcClient) VerifyTronTransaction(ctx context.Context, rpcURL, txHash, depositAddr, contractAddr string, isToken bool, expectedAmount float64) (*DetectedPayment, error) {
	baseURL := rpcURL
	if baseURL == "" {
		baseURL = "https://api.trongrid.io"
	}

	cleanTx := strings.TrimSpace(txHash)
	url := fmt.Sprintf("%s/wallet/gettransactionbyid", baseURL)
	body, _ := json.Marshal(map[string]string{"value": cleanTx})
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var txInfo struct {
		TxID    string `json:"txID"`
		RawData struct {
			Contract []struct {
				Parameter struct {
					Value struct {
						Amount          int64  `json:"amount"` // in SUN
						ToAddress       string `json:"to_address"`
						OwnerAddress     string `json:"owner_address"`
						ContractAddress string `json:"contract_address"`
						Data            string `json:"data"`
					} `json:"value"`
				} `json:"parameter"`
				Type string `json:"type"`
			} `json:"contract"`
			Timestamp int64 `json:"timestamp"`
		} `json:"raw_data"`
		Ret []struct {
			ContractRet string `json:"contractRet"`
		} `json:"ret"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&txInfo); err != nil {
		return nil, err
	}

	if txInfo.TxID == "" {
		return nil, fmt.Errorf("tron transaction not found on-chain")
	}

	if len(txInfo.Ret) > 0 && txInfo.Ret[0].ContractRet != "SUCCESS" {
		return nil, fmt.Errorf("tron transaction failed on blockchain: %s", txInfo.Ret[0].ContractRet)
	}

	txTime := time.UnixMilli(txInfo.RawData.Timestamp).UTC()
	minExpected := expectedAmount * 0.97

	if isToken {
		detected, _ := c.ScanTronTRC20Deposits(ctx, rpcURL, contractAddr, depositAddr, 1)
		for _, d := range detected {
			if strings.EqualFold(d.TxHash, cleanTx) {
				return d, nil
			}
		}
	} else {
		for _, cItem := range txInfo.RawData.Contract {
			if cItem.Type == "TransferContract" {
				trxAmount := float64(cItem.Parameter.Value.Amount) / 1e6
				if trxAmount >= minExpected {
					return &DetectedPayment{
						TxHash:        txInfo.TxID,
						ToAddress:     depositAddr,
						FromAddress:   cItem.Parameter.Value.OwnerAddress,
						Amount:        trxAmount,
						BlockNumber:   0,
						Confirmations: 3,
						Confirmed:     true,
						Timestamp:     txTime,
					}, nil
				}
			}
		}
	}

	return nil, fmt.Errorf("transaction %s does not match expected deposit to %s", cleanTx, depositAddr)
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

// ScanLitecoinDeposits checks for native LTC deposits using litecoinspace/blockchair API.
func (c *MultiChainRpcClient) ScanLitecoinDeposits(ctx context.Context, apiURL, depositAddr string, minConfirmations int) ([]*DetectedPayment, error) {
	baseURL := apiURL
	if baseURL == "" {
		baseURL = "https://litecoinspace.org/api"
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
			Value               int64  `json:"value"` // in litoshis
		} `json:"vout"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&txs); err != nil {
		return nil, err
	}

	payments := make([]*DetectedPayment, 0)
	for _, tx := range txs {
		var receivedLitoshis int64
		for _, out := range tx.Vout {
			if strings.EqualFold(out.ScriptPubKeyAddress, depositAddr) {
				receivedLitoshis += out.Value
			}
		}
		if receivedLitoshis <= 0 {
			continue
		}

		amountLTC := float64(receivedLitoshis) / 1e8
		confs := 0
		if tx.Status.Confirmed {
			confs = 3
		}

		payments = append(payments, &DetectedPayment{
			TxHash:        tx.TxID,
			ToAddress:     depositAddr,
			Amount:        amountLTC,
			BlockNumber:   tx.Status.BlockHeight,
			Confirmations: confs,
			Confirmed:     tx.Status.Confirmed && confs >= minConfirmations,
			Timestamp:     time.Unix(tx.Status.BlockTime, 0).UTC(),
		})
	}

	return payments, nil
}

// ScanDogecoinDeposits checks for native DOGE deposits.
func (c *MultiChainRpcClient) ScanDogecoinDeposits(ctx context.Context, apiURL, depositAddr string, minConfirmations int) ([]*DetectedPayment, error) {
	baseURL := apiURL
	if baseURL == "" {
		baseURL = "https://dogechain.info/api/v1"
	}
	url := fmt.Sprintf("%s/address/received/%s", baseURL, depositAddr)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var res struct {
		Success  int    `json:"success"`
		Received string `json:"received"`
		Error    string `json:"error"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return nil, err
	}

	var receivedDoge float64
	_, _ = fmt.Sscanf(res.Received, "%f", &receivedDoge)

	if receivedDoge <= 0 {
		return nil, nil
	}

	payments := []*DetectedPayment{
		{
			TxHash:        fmt.Sprintf("doge_dep_%s_%d", depositAddr[:8], time.Now().Unix()/60),
			ToAddress:     depositAddr,
			Amount:        receivedDoge,
			BlockNumber:   0,
			Confirmations: 3,
			Confirmed:     true,
			Timestamp:     time.Now().UTC(),
		},
	}

	return payments, nil
}


// ----------------------------------------------------------------------------
// EVM Signing & Broadcast Helpers (Nonce, Gas Price, Raw Tx Broadcast)
// ----------------------------------------------------------------------------

func (c *MultiChainRpcClient) GetEVMNonce(ctx context.Context, rpcURL, address string) (uint64, error) {
	raw, err := c.callEVM(ctx, rpcURL, "eth_getTransactionCount", []interface{}{strings.TrimSpace(address), "pending"})
	if err != nil {
		return 0, err
	}
	var hexStr string
	if err := json.Unmarshal(raw, &hexStr); err != nil {
		return 0, err
	}
	return strconv.ParseUint(strings.TrimPrefix(hexStr, "0x"), 16, 64)
}

func (c *MultiChainRpcClient) GetEVMGasPrice(ctx context.Context, rpcURL string) (*big.Int, error) {
	raw, err := c.callEVM(ctx, rpcURL, "eth_gasPrice", []interface{}{})
	if err != nil {
		return nil, err
	}
	var hexStr string
	if err := json.Unmarshal(raw, &hexStr); err != nil {
		return nil, err
	}
	val := new(big.Int)
	val.SetString(strings.TrimPrefix(hexStr, "0x"), 16)
	return val, nil
}

func (c *MultiChainRpcClient) SendEVMRawTransaction(ctx context.Context, rpcURL, rawTxHex string) (string, error) {
	raw, err := c.callEVM(ctx, rpcURL, "eth_sendRawTransaction", []interface{}{strings.TrimSpace(rawTxHex)})
	if err != nil {
		return "", err
	}
	var txHash string
	if err := json.Unmarshal(raw, &txHash); err != nil {
		return "", err
	}
	return txHash, nil
}

// ----------------------------------------------------------------------------
// Solana JSON-RPC (SOL Native Transfers & Balance Monitoring)
// ----------------------------------------------------------------------------

func (c *MultiChainRpcClient) GetSolanaBalance(ctx context.Context, rpcURL, address string) (float64, error) {
	endpoint := rpcURL
	if endpoint == "" {
		endpoint = "https://api.mainnet-beta.solana.com"
	}
	raw, err := c.callEVM(ctx, endpoint, "getBalance", []interface{}{strings.TrimSpace(address), map[string]string{"commitment": "finalized"}})
	if err != nil {
		return 0, err
	}
	var res struct {
		Value uint64 `json:"value"` // lamports
	}
	if err := json.Unmarshal(raw, &res); err != nil {
		return 0, err
	}
	return float64(res.Value) / 1e9, nil
}

func (c *MultiChainRpcClient) VerifySolanaTransaction(ctx context.Context, rpcURL, signature, depositAddr string, expectedAmount float64) (*DetectedPayment, error) {
	endpoint := rpcURL
	if endpoint == "" {
		endpoint = "https://api.mainnet-beta.solana.com"
	}
	raw, err := c.callEVM(ctx, endpoint, "getTransaction", []interface{}{
		strings.TrimSpace(signature),
		map[string]interface{}{"encoding": "json", "commitment": "finalized", "maxSupportedTransactionVersion": 0},
	})
	if err != nil {
		return nil, err
	}
	if len(raw) == 0 || string(raw) == "null" {
		return nil, fmt.Errorf("solana transaction not found or not yet finalized")
	}

	var txRes struct {
		Slot int64 `json:"slot"`
		Meta struct {
			Err          interface{} `json:"err"`
			PreBalances  []uint64    `json:"preBalances"`
			PostBalances []uint64    `json:"postBalances"`
		} `json:"meta"`
		Transaction struct {
			Message struct {
				AccountKeys []string `json:"accountKeys"`
			} `json:"message"`
		} `json:"transaction"`
	}
	if err := json.Unmarshal(raw, &txRes); err != nil {
		return nil, err
	}
	if txRes.Meta.Err != nil {
		return nil, fmt.Errorf("solana transaction failed on-chain")
	}

	minExpected := expectedAmount * 0.97
	for idx, key := range txRes.Transaction.Message.AccountKeys {
		if strings.EqualFold(key, strings.TrimSpace(depositAddr)) {
			if idx < len(txRes.Meta.PreBalances) && idx < len(txRes.Meta.PostBalances) && txRes.Meta.PostBalances[idx] > txRes.Meta.PreBalances[idx] {
				deltaLamports := txRes.Meta.PostBalances[idx] - txRes.Meta.PreBalances[idx]
				amountSOL := float64(deltaLamports) / 1e9
				if amountSOL >= minExpected {
					return &DetectedPayment{
						TxHash:        strings.TrimSpace(signature),
						ToAddress:     depositAddr,
						Amount:        amountSOL,
						BlockNumber:   txRes.Slot,
						Confirmations: 1,
						Confirmed:     true,
						Timestamp:     time.Now().UTC(),
					}, nil
				}
			}
		}
	}
	return nil, fmt.Errorf("no matching SOL deposit to %s found in signature %s", depositAddr, signature)
}

// ----------------------------------------------------------------------------
// TRON, Bitcoin & Solana Broadcast Helpers (Phase 4 Multi-Chain Auto-Sweep)
// ----------------------------------------------------------------------------

func (c *MultiChainRpcClient) CreateTronNativeTx(ctx context.Context, rpcURL, fromAddr, toAddr string, amountSun int64) (map[string]interface{}, string, error) {
	baseURL := rpcURL
	if baseURL == "" {
		baseURL = "https://api.trongrid.io"
	}
	payload := map[string]interface{}{
		"owner_address": fromAddr,
		"to_address":    toAddr,
		"amount":        amountSun,
		"visible":       true,
	}
	body, _ := json.Marshal(payload)
	req, err := http.NewRequestWithContext(ctx, "POST", baseURL+"/wallet/createtransaction", bytes.NewReader(body))
	if err != nil {
		return nil, "", err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, "", err
	}
	defer resp.Body.Close()
	var txObj map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&txObj); err != nil {
		return nil, "", err
	}
	txID, _ := txObj["txID"].(string)
	if txID == "" {
		return nil, "", fmt.Errorf("trongrid createtransaction failed: %v", txObj["Error"])
	}
	return txObj, txID, nil
}

func (c *MultiChainRpcClient) BroadcastTronSignedTx(ctx context.Context, rpcURL string, signedTx map[string]interface{}) (string, error) {
	baseURL := rpcURL
	if baseURL == "" {
		baseURL = "https://api.trongrid.io"
	}
	body, _ := json.Marshal(signedTx)
	req, err := http.NewRequestWithContext(ctx, "POST", baseURL+"/wallet/broadcasttransaction", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	var res struct {
		Result  bool   `json:"result"`
		TxID    string `json:"txid"`
		Message string `json:"message"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return "", err
	}
	if !res.Result {
		return "", fmt.Errorf("tron broadcast rejected: %s", res.Message)
	}
	if res.TxID != "" {
		return res.TxID, nil
	}
	txID, _ := signedTx["txID"].(string)
	return txID, nil
}

func (c *MultiChainRpcClient) GetBitcoinUTXOs(ctx context.Context, apiURL, address string) ([]BitcoinUTXO, error) {
	baseURL := apiURL
	if baseURL == "" {
		baseURL = "https://blockstream.info/api"
	}
	req, err := http.NewRequestWithContext(ctx, "GET", fmt.Sprintf("%s/address/%s/utxo", baseURL, address), nil)
	if err != nil {
		return nil, err
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var raw []struct {
		TxID   string `json:"txid"`
		Vout   uint32 `json:"vout"`
		Value  int64  `json:"value"`
		Status struct {
			Confirmed bool `json:"confirmed"`
		} `json:"status"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return nil, err
	}
	out := make([]BitcoinUTXO, 0, len(raw))
	for _, u := range raw {
		if u.Status.Confirmed && u.Value > 0 {
			out = append(out, BitcoinUTXO{TxID: u.TxID, Vout: u.Vout, Amount: u.Value})
		}
	}
	return out, nil
}

func (c *MultiChainRpcClient) BroadcastBitcoinRawTx(ctx context.Context, apiURL, rawTxHex string) (string, error) {
	baseURL := apiURL
	if baseURL == "" {
		baseURL = "https://blockstream.info/api"
	}
	req, err := http.NewRequestWithContext(ctx, "POST", baseURL+"/tx", strings.NewReader(strings.TrimSpace(rawTxHex)))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "text/plain")
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	var buf bytes.Buffer
	_, _ = buf.ReadFrom(resp.Body)
	txt := strings.TrimSpace(buf.String())
	if resp.StatusCode >= 400 {
		return "", fmt.Errorf("bitcoin broadcast failed (%d): %s", resp.StatusCode, txt)
	}
	return txt, nil
}

func (c *MultiChainRpcClient) GetSolanaRecentBlockhash(ctx context.Context, rpcURL string) (string, error) {
	endpoint := rpcURL
	if endpoint == "" {
		endpoint = "https://api.mainnet-beta.solana.com"
	}
	raw, err := c.callEVM(ctx, endpoint, "getLatestBlockhash", []interface{}{map[string]string{"commitment": "finalized"}})
	if err != nil {
		return "", err
	}
	var res struct {
		Value struct {
			Blockhash string `json:"blockhash"`
		} `json:"value"`
	}
	if err := json.Unmarshal(raw, &res); err != nil {
		return "", err
	}
	if res.Value.Blockhash == "" {
		return "", fmt.Errorf("empty solana recent blockhash")
	}
	return res.Value.Blockhash, nil
}

func (c *MultiChainRpcClient) SendSolanaRawTransaction(ctx context.Context, rpcURL, base64Tx string) (string, error) {
	endpoint := rpcURL
	if endpoint == "" {
		endpoint = "https://api.mainnet-beta.solana.com"
	}
	raw, err := c.callEVM(ctx, endpoint, "sendTransaction", []interface{}{
		base64Tx,
		map[string]interface{}{"encoding": "base64", "preflightCommitment": "confirmed"},
	})
	if err != nil {
		return "", err
	}
	var sig string
	if err := json.Unmarshal(raw, &sig); err != nil {
		return "", err
	}
	return sig, nil
}

// CreateTronTRC20Tx builds an unsigned TRC20 transfer(address,uint256) TriggerSmartContract transaction via TronGrid.
func (c *MultiChainRpcClient) CreateTronTRC20Tx(ctx context.Context, rpcURL, fromAddr, contractAddr, toAddr string, amountSmallestUnits *big.Int, feeLimitSun int64) (map[string]interface{}, string, error) {
	baseURL := rpcURL
	if baseURL == "" {
		baseURL = "https://api.trongrid.io"
	}
	paramHex, err := EncodeTronTRC20TransferParam(toAddr, amountSmallestUnits)
	if err != nil {
		return nil, "", err
	}
	if feeLimitSun <= 0 || feeLimitSun > 50_000_000 {
		feeLimitSun = 30_000_000 // 30 TRX hard safety cap
	}
	payload := map[string]interface{}{
		"owner_address":     fromAddr,
		"contract_address":  contractAddr,
		"function_selector": "transfer(address,uint256)",
		"parameter":         paramHex,
		"fee_limit":         feeLimitSun,
		"call_value":        0,
		"visible":           true,
	}
	body, _ := json.Marshal(payload)
	req, err := http.NewRequestWithContext(ctx, "POST", baseURL+"/wallet/triggersmartcontract", bytes.NewReader(body))
	if err != nil {
		return nil, "", err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, "", err
	}
	defer resp.Body.Close()

	var res struct {
		Result struct {
			Result  bool   `json:"result"`
			Message string `json:"message"`
		} `json:"result"`
		Transaction map[string]interface{} `json:"transaction"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return nil, "", err
	}
	if !res.Result.Result || res.Transaction == nil {
		return nil, "", fmt.Errorf("trongrid triggersmartcontract failed: %s", res.Result.Message)
	}
	txID, _ := res.Transaction["txID"].(string)
	if txID == "" {
		return nil, "", fmt.Errorf("empty txID from trongrid triggersmartcontract")
	}
	return res.Transaction, txID, nil
}
