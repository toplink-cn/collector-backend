package db

import (
	"collector-backend/util"
	"database/sql"
	"fmt"

	_ "github.com/go-sql-driver/mysql"
)

func GetMysqlConnection() *sql.DB {
	// 数据库连接参数
	dbUser := "dcim"
	dbPass := "ULUxbq8Clw6K3sJ8Ax4KJ4jMo7gwJLPI"
	dbUnixSocket := "/Users/pange/run/mysqld.sock"
	dbName := "dcim"

	// 构建连接字符串
	connStr := fmt.Sprintf("%s:%s@unix(%s)/%s", dbUser, dbPass, dbUnixSocket, dbName)

	// 打开数据库连接
	db, err := sql.Open("mysql", connStr)
	util.FailOnError(err, "连接数据库失败")

	return db
}
