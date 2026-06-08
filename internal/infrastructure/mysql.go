package infrastructure

import (
	"database/sql"
	"fmt"
	"log"
	"time"

	_ "github.com/go-sql-driver/mysql"

	"bbc-mcp/internal/config"
)

func NewMySQLDataSource(cfg *config.DatabaseConfig) (*sql.DB, error) {
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8&parseTime=true&loc=Local",
		cfg.User, cfg.Password, cfg.Host, cfg.Port, cfg.Name)

	db, err := sql.Open("mysql", dsn)
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
