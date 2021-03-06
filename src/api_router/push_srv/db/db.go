package db

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"
	_ "github.com/go-sql-driver/mysql"
	l4g "github.com/alecthomas/log4go"
	"api_router/base/config"
)

var (
	Url      = ""//"root@tcp(127.0.0.1:3306)/wallet"
	database string
	usertable = "user_property"
	db       *sql.DB

	q = map[string]string{}

	accountQ = map[string]string{
		"readUserCallbackUrl": "SELECT callback_url from %s.%s where user_key = ? limit ? offset ?",
	}

	st = map[string]*sql.Stmt{}
)

func Init(configPath string) {
	var d *sql.DB
	var err error

	err = config.LoadJsonNode(configPath, "db", &Url)
	if err != nil {
		l4g.Crashf("", err)
	}

	parts := strings.Split(Url, "/")
	if len(parts) != 2 {
		l4g.Crashf("Invalid database url")
	}

	if len(parts[1]) == 0 {
		l4g.Crashf("Invalid database name")
	}

	//url := parts[0]
	database = parts[1]

	//if d, err = sql.Open("mysql", url+"/"); err != nil {
	//	l4g.Crashf(err)
	//}
	//if _, err := d.Exec("CREATE DATABASE IF NOT EXISTS " + database); err != nil {
	//	l4g.Crashf(err)
	//}
	//d.Close()
	if d, err = sql.Open("mysql", Url); err != nil {
		l4g.Crashf("", err)
	}
	// http://www.01happy.com/golang-go-sql-drive-mysql-connection-pooling/
	d.SetMaxOpenConns(2000)
	d.SetMaxIdleConns(1000)
	d.Ping()
	//if _, err = d.Exec(accountdb.UsersSchema); err != nil {
	//	l4g.Crash(err)
	//}

	db = d

	for query, statement := range accountQ {
		prepared, err := db.Prepare(fmt.Sprintf(statement, database, usertable))
		if err != nil {
			l4g.Crashf("", err)
		}
		st[query] = prepared
	}
}

func ReadUserCallbackUrl(userKey string) (string, error) {
	var r *sql.Rows
	var err error

	r, err = st["readUserCallbackUrl"].Query(userKey, 1, 0)
	if err != nil {
		return "", err
	}
	defer r.Close()

	if !r.Next() {
		return "", errors.New("row no next")
	}

	var url string
	if err := r.Scan(&url); err != nil {
		if err == sql.ErrNoRows {
			return "", errors.New("no rows")
		}
		return "", err
	}
	if r.Err() != nil {
		return "", err
	}

	return url, nil
}