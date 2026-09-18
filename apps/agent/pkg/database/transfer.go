package database

import (
	"bufio"
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
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

// ImportSQL executes SQL statements read from a stream against the target database.
func (m *Manager) ImportSQL(ctx context.Context, dbName string, r io.Reader) (int, int, error) {
	db, err := m.pool.GetDB(ctx, dbName)
	if err != nil {
		return 0, 0, err
	}

	scanner := bufio.NewScanner(r)
	var currentStmt strings.Builder
	var successful, failed int
	var delimiter = ";"

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Skip comments
		if strings.HasPrefix(line, "--") || strings.HasPrefix(line, "/*") || strings.HasPrefix(line, "#") {
			continue
		}

		// Delimiter changes (e.g. for triggers / stored procedures)
		if strings.HasPrefix(strings.ToUpper(line), "DELIMITER ") {
			delimiter = strings.TrimSpace(line[10:])
			continue
		}

		currentStmt.WriteString(line)
		currentStmt.WriteString("\n")

		if strings.HasSuffix(line, delimiter) {
			raw := currentStmt.String()
			currentStmt.Reset()

			stmt := strings.TrimSuffix(strings.TrimSpace(raw), delimiter)
			if stmt != "" {
				_, err := db.ExecContext(ctx, stmt)
				if err != nil {
					failed++
				} else {
					successful++
				}
			}
		}
	}

	// Handle trailing statement without final delimiter
	if currentStmt.Len() > 0 {
		stmt := strings.TrimSpace(currentStmt.String())
		if stmt != "" {
			_, err := db.ExecContext(ctx, stmt)
			if err != nil {
				failed++
			} else {
				successful++
			}
		}
	}

	return successful, failed, scanner.Err()
}
