// Package database creates database connections used by the application.
package database

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/MAK890/task-manager-api/internal/config"
	"github.com/go-sql-driver/mysql"
)

// OpenMySQL prepares the MySQL client and confirms that the server is reachable.
func OpenMySQL(ctx context.Context, cfg config.MySQL) (*sql.DB, error) {
	// MySQL driver di config hun environment-derived application config ton aundi hai.
	driverConfig := mysql.Config{
		User:      cfg.User,
		Passwd:    cfg.Password,
		Net:       "tcp",
		Addr:      cfg.Address,
		DBName:    cfg.Database,
		ParseTime: true,
	}

	db, err := sql.Open("mysql", driverConfig.FormatDSN())
	if err != nil {
		return nil, fmt.Errorf("prepare MySQL connection: %w", err)
	}

	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("connect to MySQL: %w", err)
	}

	return db, nil
}
