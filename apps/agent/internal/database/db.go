package database

import (
	"database/sql"
	"fmt"
	"strings"

	_ "github.com/lib/pq"
)

type DBManager struct{}

func NewDBManager() *DBManager {
	return &DBManager{}
}

func (m *DBManager) CreateMySQLDatabase(rootDSN, dbName, charset, collation string) error {
	if charset == "" {
		charset = "utf8mb4"
	}
	if collation == "" {
		collation = "utf8mb4_unicode_ci"
	}

	dbName = sanitizeIdentifier(dbName)
	query := fmt.Sprintf("CREATE DATABASE IF NOT EXISTS `%s` CHARACTER SET %s COLLATE %s", dbName, charset, collation)

	db, err := sql.Open("mysql", rootDSN)
	if err != nil {
		return err
	}
	defer db.Close()

	_, err = db.Exec(query)
	return err
}

func (m *DBManager) DeleteMySQLDatabase(rootDSN, dbName string) error {
	dbName = sanitizeIdentifier(dbName)
	query := fmt.Sprintf("DROP DATABASE IF EXISTS `%s`", dbName)

	db, err := sql.Open("mysql", rootDSN)
	if err != nil {
		return err
	}
	defer db.Close()

	_, err = db.Exec(query)
	return err
}

func (m *DBManager) CreatePostgresDatabase(rootDSN, dbName string) error {
	dbName = sanitizeIdentifier(dbName)
	query := fmt.Sprintf("CREATE DATABASE %s", dbName)

	db, err := sql.Open("postgres", rootDSN)
	if err != nil {
		return err
	}
	defer db.Close()

	_, err = db.Exec(query)
	if err != nil && strings.Contains(err.Error(), "already exists") {
		return nil
	}
	return err
}

func (m *DBManager) DeletePostgresDatabase(rootDSN, dbName string) error {
	dbName = sanitizeIdentifier(dbName)
	query := fmt.Sprintf("DROP DATABASE IF EXISTS %s", dbName)

	db, err := sql.Open("postgres", rootDSN)
	if err != nil {
		return err
	}
	defer db.Close()

	_, err = db.Exec(query)
	return err
}

func sanitizeIdentifier(name string) string {
	name = strings.ReplaceAll(name, ";", "")
	name = strings.ReplaceAll(name, "--", "")
	name = strings.ReplaceAll(name, "'", "")
	name = strings.ReplaceAll(name, "\"", "")
	name = strings.ReplaceAll(name, "`", "")
	return strings.TrimSpace(name)
}
