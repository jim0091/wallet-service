package db

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"
	"api_router/account_srv/user"
	_ "github.com/go-sql-driver/mysql"
	l4g "github.com/alecthomas/log4go"
	"api_router/base/config"
)

var (
	//Url      = "root:root@tcp(127.0.0.1:3306)/wallet"
	Url      = ""//"root@tcp(127.0.0.1:3306)/wallet"
	database string
	usertable = "user_property"
	db       *sql.DB

	q = map[string]string{}

	accountQ = map[string]string{
		"register": `INSERT into %s.%s (
				user_key, user_class, 
				public_key, source_ip, callback_url, level, is_frozen,
				create_time, update_time) 
				values (?, ?,
				?, ?, ?, ?, ?,
				?, ?)`,
		"delete": "DELETE from %s.%s where user_key = ?",

		"updateProfile":          "UPDATE %s.%s set public_key = ?, source_ip = ?, callback_url = ?, update_time = ? where user_key = ?",
		"readProfile":         	  "SELECT public_key, source_ip, callback_url from %s.%s where user_key = ?",

		"setFrozen":         	  "UPDATE %s.%s set is_frozen = ? where user_key = ?",

		"listUsers":              "SELECT id, user_key, user_class, level, is_frozen from %s.%s where id < ? order by id desc limit ?",
		"listUsers2":             "SELECT id, user_key, user_class, level, is_frozen from %s.%s order by id desc limit ?",
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
	//	l4g.Crashf(err.Error())
	//}
	// do not create db auto
	//if _, err := d.Exec("CREATE DATABASE IF NOT EXISTS " + database); err != nil {
	//	l4g.Crashf(err.Error())
	//}
	//d.Close()

	if d, err = sql.Open("mysql", Url); err != nil {
		l4g.Crashf(err.Error())
	}
	// http://www.01happy.com/golang-go-sql-drive-mysql-connection-pooling/
	d.SetMaxOpenConns(2000)
	d.SetMaxIdleConns(1000)
	d.Ping()
	// do not create table auto
	//if _, err = d.Exec(UsersSchema); err != nil {
	//	l4g.Crashf(err.Error())
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

func Register(userRegister *user.ReqUserRegister, userKey string) error {
	var datetime = time.Now().UTC()
	datetime.Format(time.RFC3339)
	_, err := st["register"].Exec(
		userKey, userRegister.UserClass,
		"", "", "", userRegister.Level, 0,
		datetime, datetime)
	return err
}

func Delete(userKey string) error {
	_, err := st["delete"].Exec(userKey)
	return err
}

func UpdateProfile(userUpdateProfile *user.ReqUserUpdateProfile) error {
	var datetime = time.Now().UTC()
	datetime.Format(time.RFC3339)
	_, err := st["updateProfile"].Exec(userUpdateProfile.PublicKey, userUpdateProfile.SourceIP, userUpdateProfile.CallbackUrl, datetime, userUpdateProfile.UserKey)
	return err
}

func ReadProfile(userKey string) (*user.AckUserReadProfile, error) {
	var r *sql.Rows
	var err error

	r, err = st["readProfile"].Query(userKey)
	if err != nil {
		return nil, err
	}
	defer r.Close()

	if !r.Next() {
		return nil, errors.New("row no next")
	}

	ackUserReadProfile := &user.AckUserReadProfile{}
	if err := r.Scan(&ackUserReadProfile.PublicKey, &ackUserReadProfile.SourceIP, &ackUserReadProfile.CallbackUrl); err != nil {
		if err == sql.ErrNoRows {
			return nil, errors.New("no rows")
		}
		return nil, err
	}
	if r.Err() != nil {
		return nil, err
	}

	ackUserReadProfile.UserKey = userKey
	return ackUserReadProfile, nil
}

func SetFrozen(userKey string, frozen rune) error {
	_, err := st["setFrozen"].Exec(frozen, userKey)
	return err
}

func ListUsers(id int, num int) (*user.AckUserList, error) {
	var r *sql.Rows
	var err error

	if id < 0 {
		r, err = st["listUsers2"].Query(num)
	}else{
		r, err = st["listUsers"].Query(id, num)
	}

	if err != nil {
		return nil, err
	}
	defer r.Close()

	ul := &user.AckUserList{}
	for r.Next()  {
		up := user.UserBasic{}
		if err := r.Scan(&up.Id, &up.UserKey, &up.UserClass, &up.Level, &up.IsFrozen); err != nil {
			if err == sql.ErrNoRows {
				continue
			}
			continue
		}

		ul.Users = append(ul.Users, up)
	}

	if r.Err() != nil {
		return nil, err
	}

	return ul, nil
}