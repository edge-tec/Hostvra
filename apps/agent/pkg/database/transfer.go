package database

import (
	"bufio"
	"bytes"
	"context"
	"database/sql"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"time"
)

// StreamExport streams database or table data in SQL, CSV, or JSON format.
func (m *Manager) StreamExport(ctx context.Context, dbName, format string, tables []string, includeData, includeStructure bool, w io.Writer) error {
	db, err := m.pool.GetDB(ctx, dbName)
	if err != nil {
		return err
	}

	// If tables not specified, fetch all tables in database
	if len(tables) == 0 {
		tableDetails, err := m.GetLiveTablesDetails(ctx, dbName)
		if err != nil {
			return err
		}
		for _, t := range tableDetails {
			tables = append(tables, t.Name)
		}
	}

	bw := bufio.NewWriter(w)
	defer bw.Flush()

	switch strings.ToLower(format) {
	case "csv":
		csvWriter := csv.NewWriter(bw)
		defer csvWriter.Flush()

		for _, tbl := range tables {
			rows, err := db.QueryContext(ctx, fmt.Sprintf("SELECT * FROM %s", SafeQuoteIdentifier(tbl)))
			if err != nil {
				continue
			}
			cols, _ := rows.Columns()
			_ = csvWriter.Write(cols)

			values := make([]interface{}, len(cols))
			scanArgs := make([]interface{}, len(cols))
			for i := range values {
				scanArgs[i] = &values[i]
			}

			for rows.Next() {
				if err := rows.Scan(scanArgs...); err == nil {
					record := make([]string, len(cols))
					for i, v := range values {
						if v == nil {
							record[i] = "NULL"
						} else {
							switch val := v.(type) {
							case []byte:
								record[i] = string(val)
							default:
								record[i] = fmt.Sprintf("%v", val)
							}
						}
					}
					_ = csvWriter.Write(record)
				}
			}
			rows.Close()
		}
		return nil

	case "json":
		allData := make(map[string][]map[string]interface{})
		for _, tbl := range tables {
			rows, err := db.QueryContext(ctx, fmt.Sprintf("SELECT * FROM %s LIMIT 5000", SafeQuoteIdentifier(tbl)))
			if err != nil {
				continue
			}
			cols, _ := rows.Columns()
			var tableRows []map[string]interface{}

			values := make([]interface{}, len(cols))
			scanArgs := make([]interface{}, len(cols))
			for i := range values {
				scanArgs[i] = &values[i]
			}

			for rows.Next() {
				if err := rows.Scan(scanArgs...); err == nil {
					rowMap := make(map[string]interface{})
					for i, col := range cols {
						val := values[i]
						if b, ok := val.([]byte); ok {
							rowMap[col] = string(b)
						} else {
							rowMap[col] = val
						}
					}
					tableRows = append(tableRows, rowMap)
				}
			}
			rows.Close()
			allData[tbl] = tableRows
		}
		encoder := json.NewEncoder(bw)
		encoder.SetIndent("", "  ")
		return encoder.Encode(allData)

	default: // SQL Dump
		fmt.Fprintf(bw, "-- Hostvra Production Database Dump\n")
		fmt.Fprintf(bw, "-- Database: `%s`\n", dbName)
		fmt.Fprintf(bw, "-- Generated at: %s\n\n", time.Now().Format("2006-01-02 15:04:05"))
		fmt.Fprintf(bw, "SET NAMES utf8mb4;\nSET FOREIGN_KEY_CHECKS = 0;\n\n")

		for _, tbl := range tables {
			quotedTbl := SafeQuoteIdentifier(tbl)

			if includeStructure {
				var dummyTbl, createStmt string
				err := db.QueryRowContext(ctx, fmt.Sprintf("SHOW CREATE TABLE %s", quotedTbl)).Scan(&dummyTbl, &createStmt)
				if err == nil {
					fmt.Fprintf(bw, "-- Table structure for %s\n", quotedTbl)
					fmt.Fprintf(bw, "DROP TABLE IF EXISTS %s;\n", quotedTbl)
					fmt.Fprintf(bw, "%s;\n\n", createStmt)
				}
			}

			if includeData {
				rows, err := db.QueryContext(ctx, fmt.Sprintf("SELECT * FROM %s", quotedTbl))
				if err != nil {
					continue
				}

				cols, _ := rows.Columns()
				var quotedCols []string
				for _, c := range cols {
					quotedCols = append(quotedCols, SafeQuoteIdentifier(c))
				}

				values := make([]interface{}, len(cols))
				scanArgs := make([]interface{}, len(cols))
				for i := range values {
					scanArgs[i] = &values[i]
				}

				var insertBatch []string
				batchCount := 0

				fmt.Fprintf(bw, "-- Dumping data for table %s\n", quotedTbl)

				for rows.Next() {
					if err := rows.Scan(scanArgs...); err == nil {
						var valStrs []string
						for _, v := range values {
							if v == nil {
								valStrs = append(valStrs, "NULL")
							} else {
								switch val := v.(type) {
								case []byte:
									escaped := strings.ReplaceAll(string(val), "\\", "\\\\")
									escaped = strings.ReplaceAll(escaped, "'", "\\'")
									escaped = strings.ReplaceAll(escaped, "\n", "\\n")
									escaped = strings.ReplaceAll(escaped, "\r", "\\r")
									valStrs = append(valStrs, fmt.Sprintf("'%s'", escaped))
								case string:
									escaped := strings.ReplaceAll(val, "\\", "\\\\")
									escaped = strings.ReplaceAll(escaped, "'", "\\'")
									escaped = strings.ReplaceAll(escaped, "\n", "\\n")
									escaped = strings.ReplaceAll(escaped, "\r", "\\r")
									valStrs = append(valStrs, fmt.Sprintf("'%s'", escaped))
								case time.Time:
									valStrs = append(valStrs, fmt.Sprintf("'%s'", val.Format("2006-01-02 15:04:05")))
								default:
									valStrs = append(valStrs, fmt.Sprintf("%v", val))
								}
							}
						}
						insertBatch = append(insertBatch, fmt.Sprintf("(%s)", strings.Join(valStrs, ", ")))
						batchCount++

						if batchCount >= 100 {
							fmt.Fprintf(bw, "INSERT INTO %s (%s) VALUES\n%s;\n", quotedTbl, strings.Join(quotedCols, ", "), strings.Join(insertBatch, ",\n"))
							insertBatch = nil
							batchCount = 0
						}
					}
				}
				rows.Close()

				if len(insertBatch) > 0 {
					fmt.Fprintf(bw, "INSERT INTO %s (%s) VALUES\n%s;\n\n", quotedTbl, strings.Join(quotedCols, ", "), strings.Join(insertBatch, ",\n"))
				}
			}
		}

		fmt.Fprintf(bw, "SET FOREIGN_KEY_CHECKS = 1;\n")
		return nil
	}
}

// ImportResult captures metrics from an SQL import operation.
type ImportResult struct {
	Successful int      `json:"successful"`
	Failed     int      `json:"failed"`
	Total      int      `json:"total"`
	Tables     int      `json:"tables"`
	Errors     []string `json:"errors,omitempty"`
}

// ImportSQL executes SQL statements read from a stream against the target database.
// It supports files up to 128 MB with automatic temporary file buffering,
// fast-path native CLI execution (mysql/mariadb), and a streaming quote-aware
// fallback parser with utf8mb4 encoding, foreign-key isolation, and comment preservation.
func (m *Manager) ImportSQL(ctx context.Context, dbName string, r io.Reader) (*ImportResult, error) {
	if strings.TrimSpace(dbName) == "" {
		return nil, errors.New("target database name is required")
	}

	// 1. Buffer stream to secure temporary file (max 128 MB)
	tmpFile, err := os.CreateTemp("", fmt.Sprintf("hostvra_import_%s_*.sql", dbName))
	if err != nil {
		return nil, fmt.Errorf("failed to create temporary file for import: %w", err)
	}
	tmpPath := tmpFile.Name()
	defer func() {
		_ = os.Remove(tmpPath)
	}()
	defer tmpFile.Close()

	const maxImportBytes = 128 * 1024 * 1024 // 128 MB max upload size
	written, err := io.Copy(tmpFile, io.LimitReader(r, maxImportBytes+1))
	if err != nil {
		return nil, fmt.Errorf("failed to read import stream: %w", err)
	}
	if written > maxImportBytes {
		return nil, fmt.Errorf("import file exceeds maximum allowed size of 128 MB")
	}
	if written == 0 {
		return nil, errors.New("import file is empty (0 bytes received)")
	}
	_ = tmpFile.Sync()

	// 2. Attempt fast-path execution via native mysql/mariadb CLI if available
	var cliErr error
	binPath, err := exec.LookPath("mysql")
	if err != nil {
		binPath, err = exec.LookPath("mariadb")
	}

	if binPath != "" {
		args := []string{
			"--default-character-set=utf8mb4",
			"--binary-mode=1",
			"--max_allowed_packet=128M",
		}
		if m.rootPassword != "" {
			args = append(args, "-u", "root", fmt.Sprintf("-p%s", m.rootPassword))
		} else {
			args = append(args, "-u", "root")
		}
		if m.pool != nil && m.pool.socketPath != "" {
			if _, sErr := os.Stat(m.pool.socketPath); sErr == nil {
				args = append(args, fmt.Sprintf("--socket=%s", m.pool.socketPath))
			}
		}
		args = append(args, dbName)

		cmd := exec.CommandContext(ctx, binPath, args...)
		inFile, openErr := os.Open(tmpPath)
		if openErr == nil {
			cmd.Stdin = inFile
			var stderrBuf bytes.Buffer
			cmd.Stderr = &stderrBuf

			cliErr = cmd.Run()
			_ = inFile.Close()

			if cliErr == nil {
				// CLI import succeeded! Count live tables to verify import.
				tableCount := 0
				if db, dbErr := m.pool.GetDB(ctx, dbName); dbErr == nil {
					if rows, qErr := db.QueryContext(ctx, "SHOW TABLES"); qErr == nil {
						for rows.Next() {
							tableCount++
						}
						_ = rows.Close()
					}
				}

				stmtCount := countStatements(tmpPath)
				if stmtCount <= 0 {
					stmtCount = 1
				}

				return &ImportResult{
					Successful: stmtCount,
					Failed:     0,
					Total:      stmtCount,
					Tables:     tableCount,
				}, nil
			}

			// CLI failed, capture stderr and fall back to Go engine
			cliErrStr := strings.TrimSpace(stderrBuf.String())
			if cliErrStr != "" {
				cliErr = fmt.Errorf("mysql cli error: %s (%w)", cliErrStr, cliErr)
			}
		}
	}

	// 3. Fallback to Go streaming SQL executor
	inFile, err := os.Open(tmpPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open import file for execution: %w", err)
	}
	defer inFile.Close()

	db, err := m.pool.GetDB(ctx, dbName)
	if err != nil {
		if cliErr != nil {
			return nil, fmt.Errorf("%v; database pool connection failed: %w", cliErr, err)
		}
		return nil, fmt.Errorf("database pool connection failed: %w", err)
	}

	// Use a dedicated connection so session variables apply to ALL statements
	conn, err := db.Conn(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to obtain dedicated database connection: %w", err)
	}
	defer func() {
		// Restore constraint checks on clean exit or error
		_, _ = conn.ExecContext(context.Background(), "SET FOREIGN_KEY_CHECKS = 1")
		_, _ = conn.ExecContext(context.Background(), "SET UNIQUE_CHECKS = 1")
		_ = conn.Close()
	}()

	// Configure session for reliable bulk import
	_, _ = conn.ExecContext(ctx, "SET NAMES utf8mb4")
	_, _ = conn.ExecContext(ctx, "SET FOREIGN_KEY_CHECKS = 0")
	_, _ = conn.ExecContext(ctx, "SET UNIQUE_CHECKS = 0")
	_, _ = conn.ExecContext(ctx, "SET SQL_MODE = 'NO_AUTO_VALUE_ON_ZERO'")
	_, _ = conn.ExecContext(ctx, "SET AUTOCOMMIT = 1")

	result := &ImportResult{}
	if err := parseAndExecuteSQL(ctx, inFile, conn, result); err != nil && result.Successful == 0 {
		return nil, err
	}

	// Count live tables after import
	if rows, qErr := conn.QueryContext(ctx, "SHOW TABLES"); qErr == nil {
		for rows.Next() {
			result.Tables++
		}
		_ = rows.Close()
	}
	result.Total = result.Successful + result.Failed

	return result, nil
}

// countStatements scans a file to estimate statement count without loading into RAM.
func countStatements(filePath string) int {
	f, err := os.Open(filePath)
	if err != nil {
		return 0
	}
	defer f.Close()

	reader := bufio.NewReaderSize(f, 64*1024)
	count := 0
	inQuote := false
	var quoteChar byte
	inLineComment := false
	inBlockComment := false

	for {
		b, err := reader.ReadByte()
		if err != nil {
			break
		}
		if inLineComment {
			if b == '\n' {
				inLineComment = false
			}
			continue
		}
		if inBlockComment {
			if b == '*' {
				if next, pErr := reader.Peek(1); pErr == nil && len(next) > 0 && next[0] == '/' {
					_, _ = reader.ReadByte()
					inBlockComment = false
				}
			}
			continue
		}
		if inQuote {
			if b == '\\' {
				_, _ = reader.ReadByte()
				continue
			}
			if b == quoteChar {
				inQuote = false
			}
			continue
		}
		if b == '\'' || b == '"' || b == '`' {
			inQuote = true
			quoteChar = b
			continue
		}
		if b == '-' {
			if next, pErr := reader.Peek(1); pErr == nil && len(next) > 0 && next[0] == '-' {
				inLineComment = true
				continue
			}
		}
		if b == '#' {
			inLineComment = true
			continue
		}
		if b == '/' {
			if next, pErr := reader.Peek(1); pErr == nil && len(next) > 0 && next[0] == '*' {
				_, _ = reader.ReadByte()
				inBlockComment = true
				continue
			}
		}
		if b == ';' {
			count++
		}
	}
	return count
}

// parseAndExecuteSQL streams and parses SQL statements with quote-awareness,
// comment-preservation (preserving executable /*!... */ conditional comments),
// DELIMITER switching, and error recording.
func parseAndExecuteSQL(ctx context.Context, r io.Reader, conn *sql.Conn, res *ImportResult) error {
	reader := bufio.NewReaderSize(r, 64*1024)
	var stmt strings.Builder
	stmt.Grow(4096)

	delimiter := ";"
	inSingleQuote := false
	inDoubleQuote := false
	inBacktick := false
	inLineComment := false
	inBlockComment := false
	isConditional := false

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		b, err := reader.ReadByte()
		if err != nil {
			if err == io.EOF {
				break
			}
			return err
		}

		// 1. Line comment (-- or #)
		if inLineComment {
			if b == '\n' {
				inLineComment = false
			}
			continue
		}

		// 2. Block comment (/* ... */)
		if inBlockComment {
			if b == '*' {
				if next, pErr := reader.Peek(1); pErr == nil && len(next) > 0 && next[0] == '/' {
					_, _ = reader.ReadByte()
					inBlockComment = false
				}
			}
			continue
		}

		// 3. MySQL conditional comment (/*! ... */) - executable SQL
		if isConditional {
			stmt.WriteByte(b)
			if b == '*' {
				if next, pErr := reader.Peek(1); pErr == nil && len(next) > 0 && next[0] == '/' {
					slash, _ := reader.ReadByte()
					stmt.WriteByte(slash)
					isConditional = false
				}
			}
			continue
		}

		// 4. Single quote literal
		if inSingleQuote {
			stmt.WriteByte(b)
			if b == '\\' {
				if nextByte, rErr := reader.ReadByte(); rErr == nil {
					stmt.WriteByte(nextByte)
				}
				continue
			}
			if b == '\'' {
				if next, pErr := reader.Peek(1); pErr == nil && len(next) > 0 && next[0] == '\'' {
					nextQuote, _ := reader.ReadByte()
					stmt.WriteByte(nextQuote)
					continue
				}
				inSingleQuote = false
			}
			continue
		}

		// 5. Double quote literal
		if inDoubleQuote {
			stmt.WriteByte(b)
			if b == '\\' {
				if nextByte, rErr := reader.ReadByte(); rErr == nil {
					stmt.WriteByte(nextByte)
				}
				continue
			}
			if b == '"' {
				inDoubleQuote = false
			}
			continue
		}

		// 6. Backtick identifier
		if inBacktick {
			stmt.WriteByte(b)
			if b == '`' {
				inBacktick = false
			}
			continue
		}

		// 7. Not inside any quotes or comments:
		// Check for comment starts
		if b == '-' {
			if next, pErr := reader.Peek(1); pErr == nil && len(next) >= 1 && next[0] == '-' {
				_, _ = reader.ReadByte() // consume second '-'
				inLineComment = true
				continue
			}
		}
		if b == '#' {
			inLineComment = true
			continue
		}

		if b == '/' {
			if next, pErr := reader.Peek(1); pErr == nil && len(next) > 0 && next[0] == '*' {
				_, _ = reader.ReadByte() // consume '*'
				if peekThird, tErr := reader.Peek(1); tErr == nil && len(peekThird) > 0 && peekThird[0] == '!' {
					isConditional = true
					stmt.WriteString("/*")
					continue
				} else {
					inBlockComment = true
					continue
				}
			}
		}

		// Check for string openers
		if b == '\'' {
			inSingleQuote = true
			stmt.WriteByte(b)
			continue
		}
		if b == '"' {
			inDoubleQuote = true
			stmt.WriteByte(b)
			continue
		}
		if b == '`' {
			inBacktick = true
			stmt.WriteByte(b)
			continue
		}

		// Check for DELIMITER keyword
		trimmedCurrent := strings.TrimSpace(stmt.String())
		if trimmedCurrent == "" && (b == 'D' || b == 'd') {
			peekRest, pErr := reader.Peek(9)
			if pErr == nil && strings.EqualFold(string(peekRest), "elimiter ") {
				for i := 0; i < 9; i++ {
					_, _ = reader.ReadByte()
				}
				var newDelim strings.Builder
				for {
					dByte, dErr := reader.ReadByte()
					if dErr != nil || dByte == '\n' || dByte == '\r' {
						break
					}
					newDelim.WriteByte(dByte)
				}
				newDelimStr := strings.TrimSpace(newDelim.String())
				if newDelimStr != "" {
					delimiter = newDelimStr
				}
				stmt.Reset()
				continue
			}
		}

		stmt.WriteByte(b)

		// Check if statement buffer ends with current delimiter
		if len(delimiter) == 1 {
			if b == delimiter[0] {
				raw := stmt.String()
				query := strings.TrimSpace(raw[:len(raw)-1])
				stmt.Reset()

				if query != "" {
					if _, execErr := conn.ExecContext(ctx, query); execErr != nil {
						res.Failed++
						if len(res.Errors) < 20 {
							snippet := query
							if len(snippet) > 100 {
								snippet = snippet[:100] + "..."
							}
							res.Errors = append(res.Errors, fmt.Sprintf("%v (query: %s)", execErr, snippet))
						}
					} else {
						res.Successful++
					}
				}
			}
		} else {
			if b == delimiter[len(delimiter)-1] && stmt.Len() >= len(delimiter) {
				raw := stmt.String()
				if strings.HasSuffix(raw, delimiter) {
					query := strings.TrimSpace(raw[:len(raw)-len(delimiter)])
					stmt.Reset()

					if query != "" {
						if _, execErr := conn.ExecContext(ctx, query); execErr != nil {
							res.Failed++
							if len(res.Errors) < 20 {
								snippet := query
								if len(snippet) > 100 {
									snippet = snippet[:100] + "..."
								}
								res.Errors = append(res.Errors, fmt.Sprintf("%v (query: %s)", execErr, snippet))
							}
						} else {
							res.Successful++
						}
					}
				}
			}
		}
	}

	// Trailing statement without final delimiter
	if stmt.Len() > 0 {
		query := strings.TrimSpace(stmt.String())
		if query != "" {
			if _, execErr := conn.ExecContext(ctx, query); execErr != nil {
				res.Failed++
				if len(res.Errors) < 20 {
					snippet := query
					if len(snippet) > 100 {
						snippet = snippet[:100] + "..."
					}
					res.Errors = append(res.Errors, fmt.Sprintf("%v (query: %s)", execErr, snippet))
				}
			} else {
				res.Successful++
			}
		}
	}

	return nil
}

