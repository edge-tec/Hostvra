package database

import (
	"context"
	"fmt"
	"strings"
	"time"
)

// BrowseRowsParams specifies pagination, sorting, and filtering for table rows.
type BrowseRowsParams struct {
	Database   string `json:"database"`
	Table      string `json:"table"`
	Page       int    `json:"page"`
	Limit      int    `json:"limit"`
	SortColumn string `json:"sort_column"`
	SortOrder  string `json:"sort_order"`
	SearchWord string `json:"search_word"`
}

// BrowseRowsResult holds the paginated row data and pagination metadata.
type BrowseRowsResult struct {
	Columns       []string                 `json:"columns"`
	Rows          []map[string]interface{} `json:"rows"`
	TotalRows     int64                    `json:"total_rows"`
	Page          int                      `json:"page"`
	Limit         int                      `json:"limit"`
	TotalPages    int                      `json:"total_pages"`
	ExecutionTime string                   `json:"execution_time"`
	PrimaryKey    string                   `json:"primary_key"`
}

// BrowseTableRows executes server-side paginated queries against the real database.
func (m *Manager) BrowseTableRows(ctx context.Context, p BrowseRowsParams) (*BrowseRowsResult, error) {
	start := time.Now()
	if p.Page < 1 {
		p.Page = 1
	}
	if p.Limit < 1 || p.Limit > 500 {
		p.Limit = 50
	}
	offset := (p.Page - 1) * p.Limit

	db, err := m.pool.GetDB(ctx, p.Database)
	if err != nil {
		return nil, fmt.Errorf("failed to open database %s: %w", p.Database, err)
	}

	quotedTable := SafeQuoteIdentifier(p.Table)

	// 1. Detect Primary Key for row operations
	var primaryKey string
	pkQuery := `
		SELECT COLUMN_NAME 
		FROM information_schema.COLUMNS 
		WHERE TABLE_SCHEMA = ? AND TABLE_NAME = ? AND COLUMN_KEY = 'PRI' 
		LIMIT 1
	`
	infoDB, _ := m.pool.GetDB(ctx, "information_schema")
	if infoDB != nil {
		_ = infoDB.QueryRowContext(ctx, pkQuery, p.Database, p.Table).Scan(&primaryKey)
	}

	// 2. Count Total Rows
	var totalRows int64
	var countQuery string
	var countArgs []interface{}

	if p.SearchWord != "" {
		// Discover columns to build safe LIKE search
		structRes, _ := m.GetLiveTableStructure(ctx, p.Database, p.Table)
		var likeClauses []string
		for _, col := range structRes.Columns {
			likeClauses = append(likeClauses, fmt.Sprintf("CAST(%s AS CHAR) LIKE ?", SafeQuoteIdentifier(col.Field)))
			countArgs = append(countArgs, "%"+p.SearchWord+"%")
		}
		if len(likeClauses) > 0 {
			countQuery = fmt.Sprintf("SELECT COUNT(*) FROM %s WHERE %s", quotedTable, strings.Join(likeClauses, " OR "))
		} else {
			countQuery = fmt.Sprintf("SELECT COUNT(*) FROM %s", quotedTable)
		}
	} else {
		countQuery = fmt.Sprintf("SELECT COUNT(*) FROM %s", quotedTable)
	}

	_ = db.QueryRowContext(ctx, countQuery, countArgs...).Scan(&totalRows)

	// 3. Build Select Query with Sorting and Pagination
	var selectQuery string
	var selectArgs []interface{}

	orderClause := ""
	if p.SortColumn != "" {
		sortDir := "ASC"
		if strings.EqualFold(p.SortOrder, "DESC") {
			sortDir = "DESC"
		}
		orderClause = fmt.Sprintf("ORDER BY %s %s", SafeQuoteIdentifier(p.SortColumn), sortDir)
	} else if primaryKey != "" {
		orderClause = fmt.Sprintf("ORDER BY %s ASC", SafeQuoteIdentifier(primaryKey))
	}

	if p.SearchWord != "" {
		structRes, _ := m.GetLiveTableStructure(ctx, p.Database, p.Table)
		var likeClauses []string
		for _, col := range structRes.Columns {
			likeClauses = append(likeClauses, fmt.Sprintf("CAST(%s AS CHAR) LIKE ?", SafeQuoteIdentifier(col.Field)))
			selectArgs = append(selectArgs, "%"+p.SearchWord+"%")
		}
		if len(likeClauses) > 0 {
			selectQuery = fmt.Sprintf("SELECT * FROM %s WHERE %s %s LIMIT ? OFFSET ?", quotedTable, strings.Join(likeClauses, " OR "), orderClause)
		} else {
			selectQuery = fmt.Sprintf("SELECT * FROM %s %s LIMIT ? OFFSET ?", quotedTable, orderClause)
		}
	} else {
		selectQuery = fmt.Sprintf("SELECT * FROM %s %s LIMIT ? OFFSET ?", quotedTable, orderClause)
	}
	selectArgs = append(selectArgs, p.Limit, offset)

	rows, err := db.QueryContext(ctx, selectQuery, selectArgs...)
	if err != nil {
		return nil, fmt.Errorf("query execution failed: %w", err)
	}
	defer rows.Close()

	colTypes, err := rows.ColumnTypes()
	if err != nil {
		return nil, fmt.Errorf("failed to read columns: %w", err)
	}

	var columns []string
	for _, ct := range colTypes {
		columns = append(columns, ct.Name())
	}

	var resultRows []map[string]interface{}
	for rows.Next() {
		scanArgs := make([]interface{}, len(columns))
		for i := range scanArgs {
			scanArgs[i] = new(interface{})
		}

		if err := rows.Scan(scanArgs...); err != nil {
			continue
		}

		rowMap := make(map[string]interface{})
		for i, col := range columns {
			val := *(scanArgs[i].(*interface{}))
			if val == nil {
				rowMap[col] = nil
			} else {
				switch v := val.(type) {
				case []byte:
					rowMap[col] = string(v)
				default:
					rowMap[col] = v
				}
			}
		}
		resultRows = append(resultRows, rowMap)
	}

	totalPages := 1
	if totalRows > 0 {
		totalPages = int((totalRows + int64(p.Limit) - 1) / int64(p.Limit))
	}

	return &BrowseRowsResult{
		Columns:       columns,
		Rows:          resultRows,
		TotalRows:     totalRows,
		Page:          p.Page,
		Limit:         p.Limit,
		TotalPages:    totalPages,
		ExecutionTime: fmt.Sprintf("%.4f sec", time.Since(start).Seconds()),
		PrimaryKey:    primaryKey,
	}, nil
}

// InsertTableRow inserts a new row into the specified table using parameterized inputs.
func (m *Manager) InsertTableRow(ctx context.Context, dbName, tableName string, values map[string]interface{}) (int64, error) {
	db, err := m.pool.GetDB(ctx, dbName)
	if err != nil {
		return 0, err
	}

	if len(values) == 0 {
		return 0, fmt.Errorf("no values provided for insertion")
	}

	var cols []string
	var placeholders []string
	var args []interface{}

	for k, v := range values {
		cols = append(cols, SafeQuoteIdentifier(k))
		placeholders = append(placeholders, "?")
		args = append(args, v)
	}

	query := fmt.Sprintf(
		"INSERT INTO %s (%s) VALUES (%s)",
		SafeQuoteIdentifier(tableName),
		strings.Join(cols, ", "),
		strings.Join(placeholders, ", "),
	)

	res, err := db.ExecContext(ctx, query, args...)
	if err != nil {
		return 0, err
	}

	return res.LastInsertId()
}

// UpdateTableRow updates an existing row identified by its primary key.
func (m *Manager) UpdateTableRow(ctx context.Context, dbName, tableName, pkColumn string, pkValue interface{}, values map[string]interface{}) (int64, error) {
	db, err := m.pool.GetDB(ctx, dbName)
	if err != nil {
		return 0, err
	}

	if len(values) == 0 {
		return 0, fmt.Errorf("no values provided for update")
	}

	var setClauses []string
	var args []interface{}

	for k, v := range values {
		setClauses = append(setClauses, fmt.Sprintf("%s = ?", SafeQuoteIdentifier(k)))
		args = append(args, v)
	}

	args = append(args, pkValue)
	query := fmt.Sprintf(
		"UPDATE %s SET %s WHERE %s = ?",
		SafeQuoteIdentifier(tableName),
		strings.Join(setClauses, ", "),
		SafeQuoteIdentifier(pkColumn),
	)

	res, err := db.ExecContext(ctx, query, args...)
	if err != nil {
		return 0, err
	}

	return res.RowsAffected()
}

// DeleteTableRows deletes rows matching given primary key values.
func (m *Manager) DeleteTableRows(ctx context.Context, dbName, tableName, pkColumn string, pkValues []interface{}) (int64, error) {
	if len(pkValues) == 0 {
		return 0, nil
	}

	db, err := m.pool.GetDB(ctx, dbName)
	if err != nil {
		return 0, err
	}

	placeholders := make([]string, len(pkValues))
	for i := range pkValues {
		placeholders[i] = "?"
	}

	query := fmt.Sprintf(
		"DELETE FROM %s WHERE %s IN (%s)",
		SafeQuoteIdentifier(tableName),
		SafeQuoteIdentifier(pkColumn),
		strings.Join(placeholders, ", "),
	)

	res, err := db.ExecContext(ctx, query, pkValues...)
	if err != nil {
		return 0, err
	}

	return res.RowsAffected()
}
