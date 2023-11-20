package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"
)

const dbDriverName = "mysql"

var httpPort = "8080"

var (
	dbName     = "test"
	dbUser     = "root"
	dbPassword = "test"
	dbHost     = "db:3306"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "err: %s", err)
		os.Exit(1)
	}
}

func run() error {
	db, err := NewDB()
	if err != nil {
		return fmt.Errorf("NewDB: %w", err)
	}

	/**
	before:
	*/
	http.HandleFunc("/query", func(w http.ResponseWriter, r *http.Request) {
		_, err = db.Query("SELECT 1")

		if err != nil {
			fmt.Fprintf(os.Stderr, "db.Query: %s\n", err)
		}

		log.Printf("/query: %+v\n", db.Stats())
	})

	/**
	after:
	*/
	http.HandleFunc("/query-ctx", func(w http.ResponseWriter, r *http.Request) {
		/**
		context.WithTimeout をかませることで、タイムアウト、cancel() したものは inUse から削除される
			= MySQL の processlist に Sleep 状態で残らなくなる
		*/
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
		defer cancel()
		_, err = db.QueryContext(ctx, "SELECT 1")

		if err != nil {
			fmt.Fprintf(os.Stderr, "db.Query: %s\n", err)
		}

		log.Printf("/query-ctx: %+v\n", db.Stats())
	})

	fmt.Println("running server", fmt.Sprintf(":%s", httpPort))
	if err := http.ListenAndServe(fmt.Sprintf(":%s", httpPort), nil); err != nil {
		return fmt.Errorf("http.ListenAndServe: %w", err)
	}

	return nil
}

func NewDB() (*sqlx.DB, error) {
	cfg := mysql.NewConfig()
	cfg.DBName = dbName
	cfg.User = dbUser
	cfg.Passwd = dbPassword
	cfg.Addr = dbHost
	cfg.Net = "tcp"

	db, err := sqlx.Open(dbDriverName, cfg.FormatDSN())
	if err != nil {
		return nil, fmt.Errorf("sql.Open: %w", err)
	}

	// db.SetMaxOpenConns(30)
	// db.SetMaxIdleConns(30)
	// db.SetConnMaxLifetime(time.Second * 5)
	// db.SetConnMaxIdleTime(time.Second * 5)

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("db.Ping: %w", err)
	}

	return db, nil
}
