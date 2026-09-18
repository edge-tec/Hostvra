package database

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"sync"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

// PoolManager manages pooled *sql.DB connections to the local or remote MySQL/MariaDB server.
type PoolManager struct {
	mu           sync.RWMutex
	pools        map[string]*sql.DB
	rootPassword string
	host         string
	port         int
	socketPath   string
}

// NewPoolManager creates a new connection pool manager.
func NewPoolManager(rootPassword string) *PoolManager {
	// Detect default socket path
	socket := "/var/run/mysqld/mysqld.sock"
	if _, err := os.Stat(socket); err != nil {
		if _, err2 := os.Stat("/run/mysqld/mysqld.sock"); err2 == nil {
			socket = "/run/mysqld/mysqld.sock"
		} else if _, err3 := os.Stat("/tmp/mysql.sock"); err3 == nil {
			socket = "/tmp/mysql.sock"
		}
	}

	return &PoolManager{
		pools:        make(map[string]*sql.DB),
		rootPassword: rootPassword,
		host:         "127.0.0.1",
		port:         3306,
		socketPath:   socket,
	}
}

// SetRootPassword updates the stored root password and clears cached pools.
func (p *PoolManager) SetRootPassword(pass string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.rootPassword = pass
	for _, db := range p.pools {
		_ = db.Close()
	}
	p.pools = make(map[string]*sql.DB)
}

// buildDSN generates connection strings prioritizing Unix socket over TCP.
func (p *PoolManager) buildDSN(dbName string) []string {
	var dsns []string

	authPart := "root"
	if p.rootPassword != "" {
		authPart = fmt.Sprintf("root:%s", p.rootPassword)
	}

	params := "parseTime=true&loc=Local&timeout=3s&readTimeout=30s&writeTimeout=30s&multiStatements=true&interpolateParams=true"

	// 1. Unix socket if file exists
	if _, err := os.Stat(p.socketPath); err == nil {
		dsn := fmt.Sprintf("%s@unix(%s)/%s?%s", authPart, p.socketPath, dbName, params)
		dsns = append(dsns, dsn)
	}

	// 2. Localhost TCP
	dsnTCP := fmt.Sprintf("%s@tcp(%s:%d)/%s?%s", authPart, p.host, p.port, dbName, params)
	dsns = append(dsns, dsnTCP)

	// 3. Fallback without password if password was non-empty and failed
	if p.rootPassword != "" {
		if _, err := os.Stat(p.socketPath); err == nil {
			dsns = append(dsns, fmt.Sprintf("root@unix(%s)/%s?%s", p.socketPath, dbName, params))
		}
		dsns = append(dsns, fmt.Sprintf("root@tcp(%s:%d)/%s?%s", p.host, p.port, dbName, params))
	}

	return dsns
}

// GetDB returns a validated *sql.DB for the given database.
func (p *PoolManager) GetDB(ctx context.Context, dbName string) (*sql.DB, error) {
	p.mu.RLock()
	db, exists := p.pools[dbName]
	p.mu.RUnlock()

	if exists {
		if err := db.PingContext(ctx); err == nil {
			return db, nil
		}
		// Pool ping failed, re-open
		p.mu.Lock()
		_ = db.Close()
		delete(p.pools, dbName)
		p.mu.Unlock()
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	// Double check
	if db, exists := p.pools[dbName]; exists {
		if err := db.PingContext(ctx); err == nil {
			return db, nil
		}
	}

	var lastErr error
	for _, dsn := range p.buildDSN(dbName) {
		conn, err := sql.Open("mysql", dsn)
		if err != nil {
			lastErr = err
			continue
		}

		conn.SetMaxOpenConns(25)
		conn.SetMaxIdleConns(5)
		conn.SetConnMaxLifetime(5 * time.Minute)
		conn.SetConnMaxIdleTime(2 * time.Minute)

		pingCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
		err = conn.PingContext(pingCtx)
		cancel()

		if err == nil {
			p.pools[dbName] = conn
			return conn, nil
		}
		_ = conn.Close()
		lastErr = err
	}

	if lastErr == nil {
		lastErr = fmt.Errorf("unable to establish connection to database %s", dbName)
	}
	return nil, lastErr
}

// Close closes all cached database pools.
func (p *PoolManager) Close() {
	p.mu.Lock()
	defer p.mu.Unlock()
	for _, db := range p.pools {
		_ = db.Close()
	}
	p.pools = make(map[string]*sql.DB)
}
