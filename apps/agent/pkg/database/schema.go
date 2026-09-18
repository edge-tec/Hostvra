package database

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"
)

// DatabaseTreeNode represents a database with all its child entities for the sidebar tree.
type DatabaseTreeNode struct {
	Name       string        `json:"name"`
	Tables     []TableNode   `json:"tables"`
	Views      []string      `json:"views"`
	Procedures []string      `json:"procedures"`
	Functions  []string      `json:"functions"`
	Events     []string      `json:"events"`
	Triggers   []TriggerNode `json:"triggers"`
}

type TableNode struct {
	Name string `json:"name"`
	Rows int64  `json:"rows"`
}

type TriggerNode struct {
	Name  string `json:"name"`
	Table string `json:"table"`
}

// TableDetail represents comprehensive table metadata.
type TableDetail struct {
	Name        string     `json:"name"`
	Type        string     `json:"type"` // BASE TABLE or VIEW
	Engine      string     `json:"engine"`
	Collation   string     `json:"collation"`
	Rows        int64      `json:"rows"`
	DataSizeKb  int64      `json:"data_size_kb"`
	IndexSizeKb int64      `json:"index_size_kb"`
	TotalSizeKb int64      `json:"total_size_kb"`
	Comment     string     `json:"comment"`
	CreateTime  *time.Time `json:"create_time,omitempty"`
	UpdateTime  *time.Time `json:"update_time,omitempty"`
}

// FullColumnDetail represents full column attributes for structure view.
type FullColumnDetail struct {
	Field      string  `json:"field"`
	Type       string  `json:"type"`
	Collation  *string `json:"collation"`
	Null       string  `json:"null"`
	Key        string  `json:"key"`
	Default    *string `json:"default"`
	Extra      string  `json:"extra"`
	Privileges string  `json:"privileges"`
	Comment    string  `json:"comment"`
}

// IndexDetail represents table index properties.
type IndexDetail struct {
	KeyName    string `json:"key_name"`
	NonUnique  int    `json:"non_unique"`
	ColumnName string `json:"column_name"`
	IndexType  string `json:"index_type"`
	SeqInIndex int    `json:"seq_in_index"`
	Comment    string `json:"comment"`
}

// ForeignKeyDetail represents referential constraints.
type ForeignKeyDetail struct {
	ConstraintName string `json:"constraint_name"`
	ColumnName     string `json:"column_name"`
	RefTable       string `json:"ref_table"`
	RefColumn      string `json:"ref_column"`
	OnUpdate       string `json:"on_update"`
	OnDelete       string `json:"on_delete"`
}

// TableStructureResult holds columns, indexes, and foreign keys.
type TableStructureResult struct {
	Table       string              `json:"table"`
	Columns     []FullColumnDetail  `json:"columns"`
	Indexes     []IndexDetail       `json:"indexes"`
	ForeignKeys []ForeignKeyDetail  `json:"foreign_keys"`
}

// RoutineDetail represents procedures and functions.
type RoutineDetail struct {
	Name       string `json:"name"`
	Type       string `json:"type"` // PROCEDURE or FUNCTION
	DataType   string `json:"data_type"`
	Parameters string `json:"parameters"`
	Definition string `json:"definition"`
	Security   string `json:"security"`
}

// EventDetail represents scheduled events.
type EventDetail struct {
	Name       string `json:"name"`
	Status     string `json:"status"` // ENABLED, DISABLED, SLAVESIDE_DISABLED
	Type       string `json:"type"`   // ONE TIME or RECURRING
	Interval   string `json:"interval"`
	Starts     string `json:"starts"`
	Definition string `json:"definition"`
}

// TriggerDetail represents database triggers.
type TriggerDetail struct {
	Name       string `json:"name"`
	Table      string `json:"table"`
	Timing     string `json:"timing"` // BEFORE or AFTER
	Event      string `json:"event"`  // INSERT, UPDATE, DELETE
	Statement  string `json:"statement"`
	Definer    string `json:"definer"`
}

// ViewDetail represents database views.
type ViewDetail struct {
	Name         string `json:"name"`
	Definition   string `json:"definition"`
	CheckOption  string `json:"check_option"`
	IsUpdatable  string `json:"is_updatable"`
	SecurityType string `json:"security_type"`
}

// SafeQuoteIdentifier quotes a database, table, or column identifier safely with backticks.
func SafeQuoteIdentifier(name string) string {
	clean := strings.ReplaceAll(name, "`", "``")
	return "`" + clean + "`"
}

// GetDatabaseTree retrieves the live hierarchy for all accessible databases.
func (m *Manager) GetDatabaseTree(ctx context.Context) ([]DatabaseTreeNode, error) {
	db, err := m.pool.GetDB(ctx, "information_schema")
	if err != nil {
		// Fallback via CLI if socket/direct connection had an issue
		return m.getDatabaseTreeCLI(ctx)
	}

	rows, err := db.QueryContext(ctx, "SELECT SCHEMA_NAME FROM information_schema.SCHEMATA WHERE SCHEMA_NAME NOT IN ('information_schema', 'performance_schema', 'sys') ORDER BY SCHEMA_NAME ASC")
	if err != nil {
		return m.getDatabaseTreeCLI(ctx)
	}
	defer rows.Close()

	var nodes []DatabaseTreeNode
	for rows.Next() {
		var schema string
		if err := rows.Scan(&schema); err == nil {
			node := DatabaseTreeNode{
				Name:       schema,
				Tables:     []TableNode{},
				Views:      []string{},
				Procedures: []string{},
				Functions:  []string{},
				Events:     []string{},
				Triggers:   []TriggerNode{},
			}

			// 1. Tables & Views
			tblRows, err := db.QueryContext(ctx, "SELECT TABLE_NAME, TABLE_TYPE, IFNULL(TABLE_ROWS, 0) FROM information_schema.TABLES WHERE TABLE_SCHEMA = ? ORDER BY TABLE_NAME ASC", schema)
			if err == nil {
				for tblRows.Next() {
					var tName, tType string
					var tRowCount int64
					if err := tblRows.Scan(&tName, &tType, &tRowCount); err == nil {
						if strings.EqualFold(tType, "VIEW") {
							node.Views = append(node.Views, tName)
						} else {
							node.Tables = append(node.Tables, TableNode{Name: tName, Rows: tRowCount})
						}
					}
				}
				tblRows.Close()
			}

			// 2. Routines (Procedures & Functions)
			rRows, err := db.QueryContext(ctx, "SELECT ROUTINE_NAME, ROUTINE_TYPE FROM information_schema.ROUTINES WHERE ROUTINE_SCHEMA = ? ORDER BY ROUTINE_NAME ASC", schema)
			if err == nil {
				for rRows.Next() {
					var rName, rType string
					if err := rRows.Scan(&rName, &rType); err == nil {
						if strings.EqualFold(rType, "FUNCTION") {
							node.Functions = append(node.Functions, rName)
						} else {
							node.Procedures = append(node.Procedures, rName)
						}
					}
				}
				rRows.Close()
			}

			// 3. Events
			evRows, err := db.QueryContext(ctx, "SELECT EVENT_NAME FROM information_schema.EVENTS WHERE EVENT_SCHEMA = ? ORDER BY EVENT_NAME ASC", schema)
			if err == nil {
				for evRows.Next() {
					var evName string
					if err := evRows.Scan(&evName); err == nil {
						node.Events = append(node.Events, evName)
					}
				}
				evRows.Close()
			}

			// 4. Triggers
			trgRows, err := db.QueryContext(ctx, "SELECT TRIGGER_NAME, EVENT_OBJECT_TABLE FROM information_schema.TRIGGERS WHERE TRIGGER_SCHEMA = ? ORDER BY TRIGGER_NAME ASC", schema)
			if err == nil {
				for trgRows.Next() {
					var trgName, trgTable string
					if err := trgRows.Scan(&trgName, &trgTable); err == nil {
						node.Triggers = append(node.Triggers, TriggerNode{Name: trgName, Table: trgTable})
					}
				}
				trgRows.Close()
			}

			nodes = append(nodes, node)
		}
	}

	return nodes, nil
}

// GetLiveTablesDetails returns complete metrics for all tables in the specified database.
func (m *Manager) GetLiveTablesDetails(ctx context.Context, dbName string) ([]TableDetail, error) {
	db, err := m.pool.GetDB(ctx, "information_schema")
	if err != nil {
		return m.getLiveTablesDetailsCLI(ctx, dbName)
	}

	query := `
		SELECT 
			TABLE_NAME, 
			IFNULL(TABLE_TYPE, 'BASE TABLE'), 
			IFNULL(ENGINE, 'InnoDB'), 
			IFNULL(TABLE_COLLATION, 'utf8mb4_unicode_ci'), 
			IFNULL(TABLE_ROWS, 0), 
			IFNULL(DATA_LENGTH, 0), 
			IFNULL(INDEX_LENGTH, 0), 
			IFNULL(TABLE_COMMENT, ''),
			CREATE_TIME,
			UPDATE_TIME
		FROM information_schema.TABLES 
		WHERE TABLE_SCHEMA = ? 
		ORDER BY TABLE_NAME ASC
	`
	rows, err := db.QueryContext(ctx, query, dbName)
	if err != nil {
		return m.getLiveTablesDetailsCLI(ctx, dbName)
	}
	defer rows.Close()

	var list []TableDetail
	for rows.Next() {
		var d TableDetail
		var dataLen, idxLen int64
		var createTime, updateTime sql.NullTime

		err := rows.Scan(
			&d.Name,
			&d.Type,
			&d.Engine,
			&d.Collation,
			&d.Rows,
			&dataLen,
			&idxLen,
			&d.Comment,
			&createTime,
			&updateTime,
		)
		if err == nil {
			d.DataSizeKb = dataLen / 1024
			d.IndexSizeKb = idxLen / 1024
			d.TotalSizeKb = (dataLen + idxLen) / 1024
			if d.TotalSizeKb == 0 && d.Rows > 0 {
				d.TotalSizeKb = 16
			}
			if createTime.Valid {
				d.CreateTime = &createTime.Time
			}
			if updateTime.Valid {
				d.UpdateTime = &updateTime.Time
			}
			list = append(list, d)
		}
	}

	return list, nil
}

// GetLiveTableStructure inspects columns, indexes, and constraints for a single table.
func (m *Manager) GetLiveTableStructure(ctx context.Context, dbName, tableName string) (*TableStructureResult, error) {
	db, err := m.pool.GetDB(ctx, dbName)
	if err != nil {
		return nil, err
	}

	res := &TableStructureResult{
		Table:       tableName,
		Columns:     []FullColumnDetail{},
		Indexes:     []IndexDetail{},
		ForeignKeys: []ForeignKeyDetail{},
	}

	// 1. Columns via SHOW FULL COLUMNS
	colRows, err := db.QueryContext(ctx, fmt.Sprintf("SHOW FULL COLUMNS FROM %s", SafeQuoteIdentifier(tableName)))
	if err != nil {
		return nil, err
	}
	defer colRows.Close()

	for colRows.Next() {
		var c FullColumnDetail
		var collation, def, priv, comm sql.NullString
		err := colRows.Scan(
			&c.Field,
			&c.Type,
			&collation,
			&c.Null,
			&c.Key,
			&def,
			&c.Extra,
			&priv,
			&comm,
		)
		if err == nil {
			if collation.Valid {
				c.Collation = &collation.String
			}
			if def.Valid {
				c.Default = &def.String
			}
			if priv.Valid {
				c.Privileges = priv.String
			}
			if comm.Valid {
				c.Comment = comm.String
			}
			res.Columns = append(res.Columns, c)
		}
	}

	// 2. Indexes via SHOW INDEX FROM
	idxRows, err := db.QueryContext(ctx, fmt.Sprintf("SHOW INDEX FROM %s", SafeQuoteIdentifier(tableName)))
	if err == nil {
		defer idxRows.Close()
		for idxRows.Next() {
			var table, keyName, colName, collation, indexType, visible, expression sql.NullString
			var nonUnique, seqInIndex int
			var cardinality, subPart, packed sql.NullInt64
			var nullable, comment, indexComment sql.NullString

			err := idxRows.Scan(
				&table,
				&nonUnique,
				&keyName,
				&seqInIndex,
				&colName,
				&collation,
				&cardinality,
				&subPart,
				&packed,
				&nullable,
				&indexType,
				&comment,
				&indexComment,
				&visible,
				&expression,
			)
			if err == nil {
				res.Indexes = append(res.Indexes, IndexDetail{
					KeyName:    keyName.String,
					NonUnique:  nonUnique,
					ColumnName: colName.String,
					IndexType:  indexType.String,
					SeqInIndex: seqInIndex,
					Comment:    comment.String,
				})
			}
		}
	}

	// 3. Foreign Keys from information_schema
	infoDB, err := m.pool.GetDB(ctx, "information_schema")
	if err == nil {
		fkQuery := `
			SELECT 
				k.CONSTRAINT_NAME, 
				k.COLUMN_NAME, 
				k.REFERENCED_TABLE_NAME, 
				k.REFERENCED_COLUMN_NAME,
				IFNULL(r.UPDATE_RULE, 'RESTRICT'),
				IFNULL(r.DELETE_RULE, 'RESTRICT')
			FROM information_schema.KEY_COLUMN_USAGE k
			JOIN information_schema.REFERENTIAL_CONSTRAINTS r 
				ON k.CONSTRAINT_NAME = r.CONSTRAINT_NAME AND k.CONSTRAINT_SCHEMA = r.CONSTRAINT_SCHEMA
			WHERE k.TABLE_SCHEMA = ? AND k.TABLE_NAME = ? AND k.REFERENCED_TABLE_NAME IS NOT NULL
		`
		fkRows, err := infoDB.QueryContext(ctx, fkQuery, dbName, tableName)
		if err == nil {
			defer fkRows.Close()
			for fkRows.Next() {
				var fk ForeignKeyDetail
				if err := fkRows.Scan(&fk.ConstraintName, &fk.ColumnName, &fk.RefTable, &fk.RefColumn, &fk.OnUpdate, &fk.OnDelete); err == nil {
					res.ForeignKeys = append(res.ForeignKeys, fk)
				}
			}
		}
	}

	return res, nil
}

func (m *Manager) getDatabaseTreeCLI(ctx context.Context) ([]DatabaseTreeNode, error) {
	dbs, err := m.GetServerDatabases(ctx)
	if err != nil {
		return nil, err
	}
	var nodes []DatabaseTreeNode
	for _, db := range dbs {
		node := DatabaseTreeNode{
			Name:       db,
			Tables:     []TableNode{},
			Views:      []string{},
			Procedures: []string{},
			Functions:  []string{},
			Events:     []string{},
			Triggers:   []TriggerNode{},
		}
		tables, err := m.GetDatabaseTables(ctx, db)
		if err == nil {
			for _, t := range tables {
				node.Tables = append(node.Tables, TableNode{
					Name: t.Name,
					Rows: t.Rows,
				})
			}
		}
		nodes = append(nodes, node)
	}
	return nodes, nil
}

func (m *Manager) getLiveTablesDetailsCLI(ctx context.Context, dbName string) ([]TableDetail, error) {
	tables, err := m.GetDatabaseTables(ctx, dbName)
	if err != nil {
		return nil, err
	}
	var details []TableDetail
	for _, t := range tables {
		details = append(details, TableDetail{
			Name:        t.Name,
			Type:        "BASE TABLE",
			Engine:      t.Engine,
			Collation:   t.Collation,
			Rows:        t.Rows,
			DataSizeKb:  t.SizeKb,
			IndexSizeKb: 0,
			TotalSizeKb: t.SizeKb,
			Comment:     t.Comment,
		})
	}
	return details, nil
}

// GetViews retrieves all views and their definitions in a database.
func (m *Manager) GetViews(ctx context.Context, dbName string) ([]ViewDetail, error) {
	db, err := m.pool.GetDB(ctx, "information_schema")
	if err != nil {
		return []ViewDetail{}, nil
	}
	rows, err := db.QueryContext(ctx, "SELECT TABLE_NAME, IFNULL(VIEW_DEFINITION, ''), IFNULL(CHECK_OPTION, 'NONE'), IFNULL(IS_UPDATABLE, 'NO'), IFNULL(SECURITY_TYPE, 'DEFINER') FROM information_schema.VIEWS WHERE TABLE_SCHEMA = ? ORDER BY TABLE_NAME ASC", dbName)
	if err != nil {
		return []ViewDetail{}, nil
	}
	defer rows.Close()

	var views []ViewDetail
	for rows.Next() {
		var v ViewDetail
		if err := rows.Scan(&v.Name, &v.Definition, &v.CheckOption, &v.IsUpdatable, &v.SecurityType); err == nil {
			views = append(views, v)
		}
	}
	return views, nil
}

// GetRoutines retrieves all stored procedures and functions in a database.
func (m *Manager) GetRoutines(ctx context.Context, dbName string) ([]RoutineDetail, error) {
	db, err := m.pool.GetDB(ctx, "information_schema")
	if err != nil {
		return []RoutineDetail{}, nil
	}
	rows, err := db.QueryContext(ctx, "SELECT ROUTINE_NAME, ROUTINE_TYPE, IFNULL(DATA_TYPE, ''), IFNULL(ROUTINE_DEFINITION, ''), IFNULL(SECURITY_TYPE, 'DEFINER') FROM information_schema.ROUTINES WHERE ROUTINE_SCHEMA = ? ORDER BY ROUTINE_NAME ASC", dbName)
	if err != nil {
		return []RoutineDetail{}, nil
	}
	defer rows.Close()

	var routines []RoutineDetail
	for rows.Next() {
		var r RoutineDetail
		if err := rows.Scan(&r.Name, &r.Type, &r.DataType, &r.Definition, &r.Security); err == nil {
			routines = append(routines, r)
		}
	}
	return routines, nil
}

// GetEvents retrieves all scheduled events in a database.
func (m *Manager) GetEvents(ctx context.Context, dbName string) ([]EventDetail, error) {
	db, err := m.pool.GetDB(ctx, "information_schema")
	if err != nil {
		return []EventDetail{}, nil
	}
	rows, err := db.QueryContext(ctx, "SELECT EVENT_NAME, IFNULL(STATUS, 'ENABLED'), IFNULL(EVENT_TYPE, 'RECURRING'), IFNULL(INTERVAL_VALUE, ''), IFNULL(STARTS, ''), IFNULL(EVENT_DEFINITION, '') FROM information_schema.EVENTS WHERE EVENT_SCHEMA = ? ORDER BY EVENT_NAME ASC", dbName)
	if err != nil {
		return []EventDetail{}, nil
	}
	defer rows.Close()

	var events []EventDetail
	for rows.Next() {
		var e EventDetail
		var starts sql.NullString
		if err := rows.Scan(&e.Name, &e.Status, &e.Type, &e.Interval, &starts, &e.Definition); err == nil {
			if starts.Valid {
				e.Starts = starts.String
			}
			events = append(events, e)
		}
	}
	return events, nil
}

// GetTriggers retrieves all triggers in a database.
func (m *Manager) GetTriggers(ctx context.Context, dbName string) ([]TriggerDetail, error) {
	db, err := m.pool.GetDB(ctx, "information_schema")
	if err != nil {
		return []TriggerDetail{}, nil
	}
	rows, err := db.QueryContext(ctx, "SELECT TRIGGER_NAME, EVENT_OBJECT_TABLE, ACTION_TIMING, EVENT_MANIPULATION, ACTION_STATEMENT, IFNULL(DEFINER, '') FROM information_schema.TRIGGERS WHERE TRIGGER_SCHEMA = ? ORDER BY TRIGGER_NAME ASC", dbName)
	if err != nil {
		return []TriggerDetail{}, nil
	}
	defer rows.Close()

	var triggers []TriggerDetail
	for rows.Next() {
		var t TriggerDetail
		if err := rows.Scan(&t.Name, &t.Table, &t.Timing, &t.Event, &t.Statement, &t.Definer); err == nil {
			triggers = append(triggers, t)
		}
	}
	return triggers, nil
}

