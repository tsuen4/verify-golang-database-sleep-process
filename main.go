/*
queryを使用する場合はrowsをCloseしよう！！
*/
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

var db *sqlx.DB

var (
	createTableQuery    = "CREATE TABLE IF NOT EXISTS `tests`(id int NOT NULL AUTO_INCREMENT, PRIMARY KEY(id))"
	insertQuery         = "INSERT INTO `tests` VALUES()"
	select1Query        = "SELECT 1"
	selectFromTestQuery = "SELECT * FROM `tests`"
)

type Test struct {
	Id int `db:"id"`
}

func main() {
	log.SetFlags(log.Ldate | log.Ltime | log.Lmicroseconds)
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "err: %s", err)
		os.Exit(1)
	}
}

func run() error {
	var err error
	db, err = NewDB()
	if err != nil {
		return fmt.Errorf("NewDB: %w", err)
	}

	http.HandleFunc("/add-data", logStats(addDataAction))
	http.HandleFunc("/select", logStats(selectAction))
	http.HandleFunc("/select-ctx", logStats(selectCtxWithTimeoutAction))
	http.HandleFunc("/select-conn", logStats(selectConnAction))
	http.HandleFunc("/query", logStats(queryAction))
	http.HandleFunc("/query-ctx", logStats(queryCtxWithTimeoutAction))
	http.HandleFunc("/query-conn", logStats(queryConnAction))

	log.Println("running server", fmt.Sprintf(":%s", httpPort))
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

	db, err := sqlx.Connect(dbDriverName, cfg.FormatDSN())
	if err != nil {
		return nil, fmt.Errorf("sql.Open: %w", err)
	}

	// db.SetMaxOpenConns(30)
	// db.SetMaxIdleConns(30)
	// db.SetConnMaxLifetime(time.Second * 5)
	// db.SetConnMaxIdleTime(time.Second * 5)

	if _, err := db.Exec(createTableQuery); err != nil {
		return nil, fmt.Errorf("db.Exec: %w", err)
	}

	return db, nil
}

func errorLog(message error) {
	fmt.Fprint(os.Stderr, message.Error()+"\n")
}

func logStats(h http.HandlerFunc) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer log.Printf("%s: %+v\n", r.URL.Path, db.Stats())
		h.ServeHTTP(w, r)
	})
}

func addDataAction(w http.ResponseWriter, r *http.Request) {
	if _, err := db.Exec(insertQuery); err != nil {
		errorLog(fmt.Errorf("db.Exec: %s", err))
	}
}

func selectAction(w http.ResponseWriter, r *http.Request) {
	// Select は内部で rows.Close() が実行される
	test := []Test{}
	if err := db.Select(&test, selectFromTestQuery); err != nil {
		errorLog(fmt.Errorf("db.Select: %s", err))
	}
	fmt.Println(test)
}

func selectCtxWithTimeoutAction(w http.ResponseWriter, r *http.Request) {
	// Select は内部で rows.Close() が実行される
	test := []Test{}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()
	if err := db.SelectContext(ctx, &test, selectFromTestQuery); err != nil {
		errorLog(fmt.Errorf("db.Select: %s", err))
	}
	fmt.Println(test)
}

func selectConnAction(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()
	conn, err := db.Connx(ctx)
	if err != nil {
		errorLog(fmt.Errorf("db.Connx: %s", err))
	}
	defer func() {
		if err := conn.Close(); err != nil {
			errorLog(fmt.Errorf("conn.Close: %s", err))
		}
	}()

	test := []Test{}
	if err := conn.SelectContext(ctx, &test, selectFromTestQuery); err != nil {
		errorLog(fmt.Errorf("conn.SelectContext: %s", err))
	}
	fmt.Println(test)
}

func queryAction(w http.ResponseWriter, r *http.Request) {
	// *sql.Rows は Close() しないと process が溜まる
	rows, err := db.Queryx(select1Query)
	if err != nil {
		errorLog(fmt.Errorf("db.Query: %s", err))
	}
	if err := rows.Close(); err != nil {
		errorLog(fmt.Errorf("rows.Close: %s", err))
	}
}

func queryCtxWithTimeoutAction(w http.ResponseWriter, r *http.Request) {
	// rows.Close() を行わなくても context.WithTimeout によってタイムアウト、cancel() したものは process から削除される
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()
	if _, err := db.QueryxContext(ctx, select1Query); err != nil {
		errorLog(fmt.Errorf("db.Query: %s", err))
	}
}

func queryConnAction(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()
	conn, err := db.Connx(ctx)
	if err != nil {
		errorLog(fmt.Errorf("db.Connx: %s", err))
	}
	defer func() {
		if err := conn.Close(); err != nil {
			errorLog(fmt.Errorf("conn.Close: %s", err))
		}
	}()

	rows, err := conn.QueryxContext(ctx, select1Query)
	if err != nil {
		errorLog(fmt.Errorf("conn.QueryxContext: %s", err))
	}
	if err := rows.Close(); err != nil {
		errorLog(fmt.Errorf("rows.Close: %s", err))
	}
}
