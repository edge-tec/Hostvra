package database

import (
	"database/sql"
	"errors"
	"fmt"
	"regexp"
	"strings"

	_ "github.com/lib/pq"
)

var (
	ErrInvalidIdentifier = errors.New("invalid database identifier: must be 1-63 characters, alphanumeric or underscore, starting with letter or underscore")
	validIdentRegex      = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]{0,62}$`)
	validCharsetRegex    = regexp.MustCompile(`^[a-zA-Z0-9_-]{1,32}$`)
	validCollationRegex  = regexp.MustCompile(`^[a-zA-Z0-9_]{1,64}$`)
)

type DBManager struct{}

func NewDBManager() *DBManager {
	return &DBManager{}
}

func validateIdentifier(name string) error {
	name = strings.TrimSpace(name)
	if !validIdentRegex.MatchString(name) {
		return fmt.Errorf("%w: '%s'", ErrInvalidIdentifier, name)
	}
	return nil
}

func (m *DBManager) CreateMySQLDatabase(rootDSN, dbName, charset, collation string) error {
	if err := validateIdentifier(dbName); err != nil {
		return err
	}

	if charset == "" {
		charset = "utf8mb4"
	}
	if collation == "" {
		collation = "utf8mb4_unicode_ci"
	}

	if !validCharsetRegex.MatchString(charset) || !validCollationRegex.MatchString(collation) {
		return errors.New("invalid charset or collation format")
	}

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
	if err := validateIdentifier(dbName); err != nil {
		return err
	}

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
	if err := validateIdentifier(dbName); err != nil {
		return err
	}

	// Double-quote identifier in PostgreSQL to prevent keyword conflicts
	query := fmt.Sprintf(`CREATE DATABASE "%s"`, dbName)

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
	if err := validateIdentifier(dbName); err != nil {
		return err
	}

	query := fmt.Sprintf(`DROP DATABASE IF EXISTS "%s"`, dbName)

	db, err := sql.Open("postgres", rootDSN)
	if err != nil {
		return err
	}
	defer db.Close()

	_, err = db.Exec(query)
	return err
}
