package infrastructure

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"fmt"
	"log"
	"time"

	"github.com/go-sql-driver/mysql"

	"bbc-mcp/internal/config"
)

const driverName = "mysql-logging"

func init() {
	mysqlDriver := &mysql.MySQLDriver{}
	sql.Register(driverName, &loggingDriver{inner: mysqlDriver})
}

type loggingDriver struct {
	inner driver.Driver
}

func (d *loggingDriver) Open(dsn string) (driver.Conn, error) {
	conn, err := d.inner.Open(dsn)
	if err != nil {
		return nil, err
	}
	return &loggingConn{Conn: conn}, nil
}

type loggingConn struct {
	driver.Conn
}

func (c *loggingConn) Prepare(query string) (driver.Stmt, error) {
	log.Printf("[mysql] PREPARE: %s", query)
	stmt, err := c.Conn.Prepare(query)
	if err != nil {
		return nil, err
	}
	return &loggingStmt{Stmt: stmt, query: query}, nil
}

// ExecerContext is used by database/sql for ExecContext
func (c *loggingConn) ExecContext(ctx context.Context, query string, args []driver.NamedValue) (driver.Result, error) {
	log.Printf("[mysql] EXEC: %s | args=%v", query, namedValues(args))
	if execer, ok := c.Conn.(driver.ExecerContext); ok {
		return execer.ExecContext(ctx, query, args)
	}
	return nil, driver.ErrSkip
}

// QueryerContext is used by database/sql for QueryContext
func (c *loggingConn) QueryContext(ctx context.Context, query string, args []driver.NamedValue) (driver.Rows, error) {
	log.Printf("[mysql] QUERY: %s | args=%v", query, namedValues(args))
	if queryer, ok := c.Conn.(driver.QueryerContext); ok {
		return queryer.QueryContext(ctx, query, args)
	}
	return nil, driver.ErrSkip
}

// ConnPrepareContext is used by database/sql for prepare with context
func (c *loggingConn) PrepareContext(ctx context.Context, query string) (driver.Stmt, error) {
	log.Printf("[mysql] PREPARE: %s", query)
	if prepCtx, ok := c.Conn.(driver.ConnPrepareContext); ok {
		stmt, err := prepCtx.PrepareContext(ctx, query)
		if err != nil {
			return nil, err
		}
		return &loggingStmt{Stmt: stmt, query: query}, nil
	}
	return nil, driver.ErrSkip
}

// Pass through optional interfaces
func (c *loggingConn) BeginTx(ctx context.Context, opts driver.TxOptions) (driver.Tx, error) {
	if beginner, ok := c.Conn.(driver.ConnBeginTx); ok {
		return beginner.BeginTx(ctx, opts)
	}
	return nil, driver.ErrSkip
}

func (c *loggingConn) Close() error {
	return c.Conn.Close()
}

func (c *loggingConn) Ping(ctx context.Context) error {
	if pinger, ok := c.Conn.(driver.Pinger); ok {
		return pinger.Ping(ctx)
	}
	return nil
}

func (c *loggingConn) ResetSession(ctx context.Context) error {
	if resetter, ok := c.Conn.(driver.SessionResetter); ok {
		return resetter.ResetSession(ctx)
	}
	return nil
}

func (c *loggingConn) CheckNamedValue(nv *driver.NamedValue) error {
	if checker, ok := c.Conn.(driver.NamedValueChecker); ok {
		return checker.CheckNamedValue(nv)
	}
	return driver.ErrSkip
}

type loggingStmt struct {
	driver.Stmt
	query string
}

func (s *loggingStmt) ExecContext(ctx context.Context, args []driver.NamedValue) (driver.Result, error) {
	log.Printf("[mysql] STMT-EXEC: %s | args=%v", s.query, namedValues(args))
	if execCtx, ok := s.Stmt.(driver.StmtExecContext); ok {
		return execCtx.ExecContext(ctx, args)
	}
	return nil, driver.ErrSkip
}

func (s *loggingStmt) QueryContext(ctx context.Context, args []driver.NamedValue) (driver.Rows, error) {
	log.Printf("[mysql] STMT-QUERY: %s | args=%v", s.query, namedValues(args))
	if queryCtx, ok := s.Stmt.(driver.StmtQueryContext); ok {
		return queryCtx.QueryContext(ctx, args)
	}
	return nil, driver.ErrSkip
}

func namedValues(args []driver.NamedValue) []any {
	vals := make([]any, len(args))
	for i, a := range args {
		vals[i] = a.Value
	}
	return vals
}

func NewMySQLDataSource(cfg *config.DatabaseConfig) (*sql.DB, error) {
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8&parseTime=true&loc=Local",
		cfg.User, cfg.Password, cfg.Host, cfg.Port, cfg.Name)

	db, err := sql.Open(driverName, dsn)
	if err != nil {
		log.Printf("mysql: 打开连接失败: %v", err)
		return nil, fmt.Errorf("mysql open: %w", err)
	}

	db.SetMaxOpenConns(cfg.MaxOpenConns)
	db.SetMaxIdleConns(cfg.MaxIdleConns)
	db.SetConnMaxLifetime(time.Hour)

	if err = db.Ping(); err != nil {
		log.Printf("mysql: Ping 失败: %v", err)
		return nil, fmt.Errorf("mysql ping: %w", err)
	}

	return db, nil
}
