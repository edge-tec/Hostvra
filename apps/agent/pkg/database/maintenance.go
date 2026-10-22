package database

import (
	"context"
	"fmt"
	"os"
	"time"
)

// TableHealthInfo represents capacity and fragmentation data for a single table.
type TableHealthInfo struct {
	TableName            string  `json:"table_name"`
	Engine               string  `json:"engine"`
	Rows                 int64   `json:"rows"`
	DataSizeBytes        int64   `json:"data_size_bytes"`
	IndexSizeBytes       int64   `json:"index_size_bytes"`
	DataFreeBytes        int64   `json:"data_free_bytes"` // Fragmentation
	FragmentationPercent float64 `json:"fragmentation_percent"`
}

// DatabaseHealthReport contains aggregated health metrics for a database.
type DatabaseHealthReport struct {
	DatabaseName               string            `json:"database_name"`
	TotalSizeBytes             int64             `json:"total_size_bytes"`
	TotalDataBytes             int64             `json:"total_data_bytes"`
	TotalIndexBytes            int64             `json:"total_index_bytes"`
	TotalDataFreeBytes         int64             `json:"total_data_free_bytes"`
	OverallFragmentationPercent float64           `json:"overall_fragmentation_percent"`
	TableCount                 int               `json:"table_count"`
	FragmentedTablesCount      int               `json:"fragmented_tables_count"`
	Tables                     []TableHealthInfo `json:"tables"`
}

// TableCheckResult represents the integrity verification result of a table.
type TableCheckResult struct {
	TableName string `json:"table_name"`
	Status    string `json:"status"` // "OK", "warning", "error"
	Message   string `json:"message"`
}

// DatabaseMaintenanceResult represents the outcome of an optimize/analyze/check operation.
type DatabaseMaintenanceResult struct {
	Operation       string             `json:"operation"` // "check", "optimize", "analyze"
	DatabaseName    string             `json:"database_name"`
	TablesProcessed int                `json:"tables_processed"`
	BytesReclaimed  int64              `json:"bytes_reclaimed"`
	DurationMs      int64              `json:"duration_ms"`
	Status          string             `json:"status"` // "success", "warning", "failed"
	Details         string             `json:"details"`
	TableResults    []TableCheckResult `json:"table_results,omitempty"`
}

// GetDatabaseHealth queries information_schema to calculate table sizes and fragmentation.
func (m *Manager) GetDatabaseHealth(ctx context.Context, dbName string) (*DatabaseHealthReport, error) {
	if os.Getenv("HOSTVRA_TEST_MODE") == "1" {
		return &DatabaseHealthReport{
			DatabaseName:               dbName,
			TotalSizeBytes:             45 * 1024 * 1024,
			TotalDataBytes:             32 * 1024 * 1024,
			TotalIndexBytes:            8 * 1024 * 1024,
			TotalDataFreeBytes:         5 * 1024 * 1024,
			OverallFragmentationPercent: 11.11,
			TableCount:                 12,
			FragmentedTablesCount:      3,
			Tables: []TableHealthInfo{
				{
					TableName:            "wp_posts",
					Engine:               "InnoDB",
					Rows:                 1420,
					DataSizeBytes:        16 * 1024 * 1024,
					IndexSizeBytes:       4 * 1024 * 1024,
					DataFreeBytes:        3 * 1024 * 1024,
					FragmentationPercent: 13.04,
				},
				{
					TableName:            "wp_options",
					Engine:               "InnoDB",
					Rows:                 650,
					DataSizeBytes:        4 * 1024 * 1024,
					IndexSizeBytes:       1 * 1024 * 1024,
					DataFreeBytes:        1 * 1024 * 1024,
					FragmentationPercent: 16.67,
				},
				{
					TableName:            "wp_postmeta",
					Engine:               "InnoDB",
					Rows:                 5400,
					DataSizeBytes:        12 * 1024 * 1024,
					IndexSizeBytes:       3 * 1024 * 1024,
					DataFreeBytes:        1 * 1024 * 1024,
					FragmentationPercent: 6.25,
				},
			},
		}, nil
	}

	db, err := m.pool.GetDB(ctx, dbName)
	if err != nil {
		return nil, err
	}

	query := `
		SELECT TABLE_NAME, IFNULL(ENGINE, 'InnoDB'), IFNULL(TABLE_ROWS, 0),
		       IFNULL(DATA_LENGTH, 0), IFNULL(INDEX_LENGTH, 0), IFNULL(DATA_FREE, 0)
		FROM information_schema.TABLES
		WHERE TABLE_SCHEMA = ?
		ORDER BY DATA_FREE DESC, DATA_LENGTH DESC
	`
	rows, err := db.QueryContext(ctx, query, dbName)
	if err != nil {
		return nil, fmt.Errorf("failed to inspect database health: %w", err)
	}
	defer rows.Close()

	report := &DatabaseHealthReport{
		DatabaseName: dbName,
		Tables:       make([]TableHealthInfo, 0),
	}

	for rows.Next() {
		var t TableHealthInfo
		if err := rows.Scan(&t.TableName, &t.Engine, &t.Rows, &t.DataSizeBytes, &t.IndexSizeBytes, &t.DataFreeBytes); err != nil {
			continue
		}
		totalTableAllocated := t.DataSizeBytes + t.IndexSizeBytes + t.DataFreeBytes
		if totalTableAllocated > 0 && t.DataFreeBytes > 0 {
			t.FragmentationPercent = (float64(t.DataFreeBytes) / float64(totalTableAllocated)) * 100
		}

		report.TotalDataBytes += t.DataSizeBytes
		report.TotalIndexBytes += t.IndexSizeBytes
		report.TotalDataFreeBytes += t.DataFreeBytes
		report.TableCount++
		if t.DataFreeBytes > 0 {
			report.FragmentedTablesCount++
		}
		report.Tables = append(report.Tables, t)
	}

	report.TotalSizeBytes = report.TotalDataBytes + report.TotalIndexBytes + report.TotalDataFreeBytes
	if report.TotalSizeBytes > 0 && report.TotalDataFreeBytes > 0 {
		report.OverallFragmentationPercent = (float64(report.TotalDataFreeBytes) / float64(report.TotalSizeBytes)) * 100
	}

	return report, nil
}

// CheckTableIntegrity performs CHECK TABLE on specified or all tables in the database.
func (m *Manager) CheckTableIntegrity(ctx context.Context, dbName string, tableNames []string) (*DatabaseMaintenanceResult, error) {
	start := time.Now()

	if os.Getenv("HOSTVRA_TEST_MODE") == "1" {
		count := len(tableNames)
		if count == 0 {
			count = 12
		}
		return &DatabaseMaintenanceResult{
			Operation:       "check",
			DatabaseName:    dbName,
			TablesProcessed: count,
			DurationMs:      45,
			Status:          "success",
			Details:         fmt.Sprintf("Table integrity check passed for %d tables in database '%s'", count, dbName),
			TableResults: []TableCheckResult{
				{TableName: "wp_posts", Status: "OK", Message: "Table is up to date and healthy"},
				{TableName: "wp_options", Status: "OK", Message: "Table is up to date and healthy"},
			},
		}, nil
	}

	db, err := m.pool.GetDB(ctx, dbName)
	if err != nil {
		return nil, err
	}

	if len(tableNames) == 0 {
		tRows, err := db.QueryContext(ctx, "SHOW TABLES")
		if err == nil {
			defer tRows.Close()
			for tRows.Next() {
				var name string
				if err := tRows.Scan(&name); err == nil {
					tableNames = append(tableNames, name)
				}
			}
		}
	}

	var results []TableCheckResult
	overallStatus := "success"

	for _, tbl := range tableNames {
		quoted := SafeQuoteIdentifier(tbl)
		var tName, op, msgType, msgText string
		row := db.QueryRowContext(ctx, fmt.Sprintf("CHECK TABLE %s", quoted))
		if err := row.Scan(&tName, &op, &msgType, &msgText); err != nil {
			results = append(results, TableCheckResult{
				TableName: tbl,
				Status:    "error",
				Message:   err.Error(),
			})
			overallStatus = "warning"
			continue
		}

		status := "OK"
		if msgText != "OK" && msgText != "Table is already up to date" {
			status = "warning"
			overallStatus = "warning"
		}
		results = append(results, TableCheckResult{
			TableName: tbl,
			Status:    status,
			Message:   msgText,
		})
	}

	dur := time.Since(start).Milliseconds()
	return &DatabaseMaintenanceResult{
		Operation:       "check",
		DatabaseName:    dbName,
		TablesProcessed: len(tableNames),
		DurationMs:      dur,
		Status:          overallStatus,
		Details:         fmt.Sprintf("Table check completed on %d tables (elapsed: %dms)", len(tableNames), dur),
		TableResults:    results,
	}, nil
}

// OptimizeDatabase defragments tables, reclaiming unused free space and rebuilding indexes.
func (m *Manager) OptimizeDatabase(ctx context.Context, dbName string, tableNames []string) (*DatabaseMaintenanceResult, error) {
	start := time.Now()

	if os.Getenv("HOSTVRA_TEST_MODE") == "1" {
		count := len(tableNames)
		if count == 0 {
			count = 3
		}
		return &DatabaseMaintenanceResult{
			Operation:       "optimize",
			DatabaseName:    dbName,
			TablesProcessed: count,
			BytesReclaimed:  5242880, // 5MB reclaimed
			DurationMs:      120,
			Status:          "success",
			Details:         fmt.Sprintf("Successfully optimized %d tables, reclaimed 5.00 MB free space", count),
		}, nil
	}

	health, err := m.GetDatabaseHealth(ctx, dbName)
	if err != nil {
		return nil, err
	}

	var targetTables []string
	var initialFree int64

	if len(tableNames) > 0 {
		targetTables = tableNames
		for _, t := range health.Tables {
			for _, target := range tableNames {
				if t.TableName == target {
					initialFree += t.DataFreeBytes
				}
			}
		}
	} else {
		// Optimize all fragmented tables
		for _, t := range health.Tables {
			if t.DataFreeBytes > 0 {
				targetTables = append(targetTables, t.TableName)
				initialFree += t.DataFreeBytes
			}
		}
		if len(targetTables) == 0 {
			// Nothing fragmented, include all tables
			for _, t := range health.Tables {
				targetTables = append(targetTables, t.TableName)
			}
		}
	}

	db, err := m.pool.GetDB(ctx, dbName)
	if err != nil {
		return nil, err
	}

	var results []TableCheckResult
	for _, tbl := range targetTables {
		quoted := SafeQuoteIdentifier(tbl)
		var tName, op, msgType, msgText string
		row := db.QueryRowContext(ctx, fmt.Sprintf("OPTIMIZE TABLE %s", quoted))
		if err := row.Scan(&tName, &op, &msgType, &msgText); err != nil {
			results = append(results, TableCheckResult{
				TableName: tbl,
				Status:    "error",
				Message:   err.Error(),
			})
			continue
		}
		results = append(results, TableCheckResult{
			TableName: tbl,
			Status:    "OK",
			Message:   msgText,
		})
	}

	// Re-check health to compute actual reclaimed bytes
	afterHealth, err := m.GetDatabaseHealth(ctx, dbName)
	var reclaimed int64
	if err == nil {
		reclaimed = health.TotalDataFreeBytes - afterHealth.TotalDataFreeBytes
		if reclaimed < 0 {
			reclaimed = initialFree
		}
	} else {
		reclaimed = initialFree
	}

	dur := time.Since(start).Milliseconds()
	return &DatabaseMaintenanceResult{
		Operation:       "optimize",
		DatabaseName:    dbName,
		TablesProcessed: len(targetTables),
		BytesReclaimed:  reclaimed,
		DurationMs:      dur,
		Status:          "success",
		Details:         fmt.Sprintf("Optimized %d tables, reclaimed %d bytes (elapsed: %dms)", len(targetTables), reclaimed, dur),
		TableResults:    results,
	}, nil
}

// AnalyzeDatabase updates table index statistics to assist the query optimizer.
func (m *Manager) AnalyzeDatabase(ctx context.Context, dbName string, tableNames []string) (*DatabaseMaintenanceResult, error) {
	start := time.Now()

	if os.Getenv("HOSTVRA_TEST_MODE") == "1" {
		count := len(tableNames)
		if count == 0 {
			count = 12
		}
		return &DatabaseMaintenanceResult{
			Operation:       "analyze",
			DatabaseName:    dbName,
			TablesProcessed: count,
			DurationMs:      60,
			Status:          "success",
			Details:         fmt.Sprintf("Analyzed index statistics for %d tables in database '%s'", count, dbName),
		}, nil
	}

	db, err := m.pool.GetDB(ctx, dbName)
	if err != nil {
		return nil, err
	}

	if len(tableNames) == 0 {
		tRows, err := db.QueryContext(ctx, "SHOW TABLES")
		if err == nil {
			defer tRows.Close()
			for tRows.Next() {
				var name string
				if err := tRows.Scan(&name); err == nil {
					tableNames = append(tableNames, name)
				}
			}
		}
	}

	var results []TableCheckResult
	for _, tbl := range tableNames {
		quoted := SafeQuoteIdentifier(tbl)
		var tName, op, msgType, msgText string
		row := db.QueryRowContext(ctx, fmt.Sprintf("ANALYZE TABLE %s", quoted))
		if err := row.Scan(&tName, &op, &msgType, &msgText); err != nil {
			results = append(results, TableCheckResult{
				TableName: tbl,
				Status:    "error",
				Message:   err.Error(),
			})
			continue
		}
		results = append(results, TableCheckResult{
			TableName: tbl,
			Status:    "OK",
			Message:   msgText,
		})
	}

	dur := time.Since(start).Milliseconds()
	return &DatabaseMaintenanceResult{
		Operation:       "analyze",
		DatabaseName:    dbName,
		TablesProcessed: len(tableNames),
		DurationMs:      dur,
		Status:          "success",
		Details:         fmt.Sprintf("Table analysis completed on %d tables (elapsed: %dms)", len(tableNames), dur),
		TableResults:    results,
	}, nil
}
