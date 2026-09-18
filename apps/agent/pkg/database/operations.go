package database

import (
	"context"
	"fmt"
	"strings"
)

// AddColumn executes ALTER TABLE ADD COLUMN.
func (m *Manager) AddColumn(ctx context.Context, dbName, tableName, colName, colType, collation, isNull, defaultVal, extra, afterCol string) error {
	db, err := m.pool.GetDB(ctx, dbName)
	if err != nil {
		return err
	}

	nullClause := "NOT NULL"
	if strings.EqualFold(isNull, "YES") || strings.EqualFold(isNull, "NULL") {
		nullClause = "NULL"
	}

	colDef := fmt.Sprintf("%s %s", SafeQuoteIdentifier(colName), colType)
	if collation != "" {
		colDef += fmt.Sprintf(" COLLATE %s", collation)
	}
	colDef += " " + nullClause

	if defaultVal != "" && defaultVal != "NULL" {
		colDef += fmt.Sprintf(" DEFAULT '%s'", strings.ReplaceAll(defaultVal, "'", "''"))
	} else if nullClause == "NULL" {
		colDef += " DEFAULT NULL"
	}

	if extra != "" {
		colDef += " " + extra
	}

	afterClause := ""
	if afterCol != "" {
		afterClause = fmt.Sprintf(" AFTER %s", SafeQuoteIdentifier(afterCol))
	}

	query := fmt.Sprintf("ALTER TABLE %s ADD COLUMN %s%s", SafeQuoteIdentifier(tableName), colDef, afterClause)
	_, err = db.ExecContext(ctx, query)
	return err
}

// ModifyColumn executes ALTER TABLE CHANGE COLUMN.
func (m *Manager) ModifyColumn(ctx context.Context, dbName, tableName, oldColName, newColName, colType, collation, isNull, defaultVal, extra string) error {
	db, err := m.pool.GetDB(ctx, dbName)
	if err != nil {
		return err
	}

	if newColName == "" {
		newColName = oldColName
	}

	nullClause := "NOT NULL"
	if strings.EqualFold(isNull, "YES") || strings.EqualFold(isNull, "NULL") {
		nullClause = "NULL"
	}

	colDef := fmt.Sprintf("%s %s %s", SafeQuoteIdentifier(oldColName), SafeQuoteIdentifier(newColName), colType)
	if collation != "" {
		colDef += fmt.Sprintf(" COLLATE %s", collation)
	}
	colDef += " " + nullClause

	if defaultVal != "" && defaultVal != "NULL" {
		colDef += fmt.Sprintf(" DEFAULT '%s'", strings.ReplaceAll(defaultVal, "'", "''"))
	} else if nullClause == "NULL" {
		colDef += " DEFAULT NULL"
	}

	if extra != "" {
		colDef += " " + extra
	}

	query := fmt.Sprintf("ALTER TABLE %s CHANGE COLUMN %s", SafeQuoteIdentifier(tableName), colDef)
	_, err = db.ExecContext(ctx, query)
	return err
}

// DropColumn executes ALTER TABLE DROP COLUMN.
func (m *Manager) DropColumn(ctx context.Context, dbName, tableName, colName string) error {
	db, err := m.pool.GetDB(ctx, dbName)
	if err != nil {
		return err
	}

	query := fmt.Sprintf("ALTER TABLE %s DROP COLUMN %s", SafeQuoteIdentifier(tableName), SafeQuoteIdentifier(colName))
	_, err = db.ExecContext(ctx, query)
	return err
}

// AddIndex executes ALTER TABLE ADD [UNIQUE|INDEX|FULLTEXT|SPATIAL].
func (m *Manager) AddIndex(ctx context.Context, dbName, tableName, indexName, indexType string, columns []string) error {
	db, err := m.pool.GetDB(ctx, dbName)
	if err != nil {
		return err
	}

	if len(columns) == 0 {
		return fmt.Errorf("at least one column must be specified for index")
	}

	var quotedCols []string
	for _, c := range columns {
		quotedCols = append(quotedCols, SafeQuoteIdentifier(c))
	}

	idxTypeUpper := strings.ToUpper(indexType)
	var query string
	if idxTypeUpper == "PRIMARY" {
		query = fmt.Sprintf("ALTER TABLE %s ADD PRIMARY KEY (%s)", SafeQuoteIdentifier(tableName), strings.Join(quotedCols, ", "))
	} else if idxTypeUpper == "UNIQUE" {
		query = fmt.Sprintf("ALTER TABLE %s ADD UNIQUE KEY %s (%s)", SafeQuoteIdentifier(tableName), SafeQuoteIdentifier(indexName), strings.Join(quotedCols, ", "))
	} else if idxTypeUpper == "FULLTEXT" {
		query = fmt.Sprintf("ALTER TABLE %s ADD FULLTEXT KEY %s (%s)", SafeQuoteIdentifier(tableName), SafeQuoteIdentifier(indexName), strings.Join(quotedCols, ", "))
	} else if idxTypeUpper == "SPATIAL" {
		query = fmt.Sprintf("ALTER TABLE %s ADD SPATIAL KEY %s (%s)", SafeQuoteIdentifier(tableName), SafeQuoteIdentifier(indexName), strings.Join(quotedCols, ", "))
	} else {
		query = fmt.Sprintf("ALTER TABLE %s ADD INDEX %s (%s)", SafeQuoteIdentifier(tableName), SafeQuoteIdentifier(indexName), strings.Join(quotedCols, ", "))
	}

	_, err = db.ExecContext(ctx, query)
	return err
}

// DropIndex executes ALTER TABLE DROP INDEX or DROP PRIMARY KEY.
func (m *Manager) DropIndex(ctx context.Context, dbName, tableName, indexName string) error {
	db, err := m.pool.GetDB(ctx, dbName)
	if err != nil {
		return err
	}

	var query string
	if strings.EqualFold(indexName, "PRIMARY") {
		query = fmt.Sprintf("ALTER TABLE %s DROP PRIMARY KEY", SafeQuoteIdentifier(tableName))
	} else {
		query = fmt.Sprintf("ALTER TABLE %s DROP INDEX %s", SafeQuoteIdentifier(tableName), SafeQuoteIdentifier(indexName))
	}

	_, err = db.ExecContext(ctx, query)
	return err
}

// RenameTable executes RENAME TABLE.
func (m *Manager) RenameTable(ctx context.Context, dbName, oldName, newName string) error {
	db, err := m.pool.GetDB(ctx, dbName)
	if err != nil {
		return err
	}

	query := fmt.Sprintf("RENAME TABLE %s TO %s", SafeQuoteIdentifier(oldName), SafeQuoteIdentifier(newName))
	_, err = db.ExecContext(ctx, query)
	return err
}

// CopyTable creates a copy of a table structure and optionally copies all rows.
func (m *Manager) CopyTable(ctx context.Context, dbName, srcTable, dstTable string, withData bool) error {
	db, err := m.pool.GetDB(ctx, dbName)
	if err != nil {
		return err
	}

	// 1. Create structure like source
	createQ := fmt.Sprintf("CREATE TABLE %s LIKE %s", SafeQuoteIdentifier(dstTable), SafeQuoteIdentifier(srcTable))
	if _, err := db.ExecContext(ctx, createQ); err != nil {
		return err
	}

	// 2. Copy rows if requested
	if withData {
		insertQ := fmt.Sprintf("INSERT INTO %s SELECT * FROM %s", SafeQuoteIdentifier(dstTable), SafeQuoteIdentifier(srcTable))
		_, err = db.ExecContext(ctx, insertQ)
		return err
	}

	return nil
}

// RunTableMaintenance executes OPTIMIZE, CHECK, ANALYZE, or REPAIR.
func (m *Manager) RunTableMaintenance(ctx context.Context, dbName, tableName, action string) (string, error) {
	db, err := m.pool.GetDB(ctx, dbName)
	if err != nil {
		return "", err
	}

	actUpper := strings.ToUpper(action)
	switch actUpper {
	case "OPTIMIZE", "CHECK", "ANALYZE", "REPAIR":
	default:
		actUpper = "OPTIMIZE"
	}

	query := fmt.Sprintf("%s TABLE %s", actUpper, SafeQuoteIdentifier(tableName))
	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		return "", err
	}
	defer rows.Close()

	var lines []string
	for rows.Next() {
		var tbl, op, msgType, msgText string
		if err := rows.Scan(&tbl, &op, &msgType, &msgText); err == nil {
			lines = append(lines, fmt.Sprintf("%s %s [%s]: %s", tbl, op, msgType, msgText))
		}
	}

	if len(lines) == 0 {
		return fmt.Sprintf("Operation %s completed.", actUpper), nil
	}
	return strings.Join(lines, "\n"), nil
}

// TruncateTable executes TRUNCATE TABLE.
func (m *Manager) TruncateTable(ctx context.Context, dbName, tableName string) error {
	db, err := m.pool.GetDB(ctx, dbName)
	if err != nil {
		return err
	}

	query := fmt.Sprintf("TRUNCATE TABLE %s", SafeQuoteIdentifier(tableName))
	_, err = db.ExecContext(ctx, query)
	return err
}

// DropTable executes DROP TABLE IF EXISTS.
func (m *Manager) DropTable(ctx context.Context, dbName, tableName string) error {
	db, err := m.pool.GetDB(ctx, dbName)
	if err != nil {
		return err
	}

	query := fmt.Sprintf("DROP TABLE IF EXISTS %s", SafeQuoteIdentifier(tableName))
	_, err = db.ExecContext(ctx, query)
	return err
}

// AlterDatabaseCollation executes ALTER DATABASE COLLATE.
func (m *Manager) AlterDatabaseCollation(ctx context.Context, dbName, collation string) error {
	db, err := m.pool.GetDB(ctx, dbName)
	if err != nil {
		return err
	}

	charset := "utf8mb4"
	if strings.HasPrefix(collation, "latin1") {
		charset = "latin1"
	} else if strings.HasPrefix(collation, "utf8_") {
		charset = "utf8"
	}

	query := fmt.Sprintf("ALTER DATABASE %s CHARACTER SET %s COLLATE %s", SafeQuoteIdentifier(dbName), charset, collation)
	_, err = db.ExecContext(ctx, query)
	return err
}
