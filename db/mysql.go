package db

import (
	"collector-backend/util"
	"database/sql"
	"fmt"
	"log"
	"os"

	_ "github.com/go-sql-driver/mysql"
	"github.com/joho/godotenv"
)

func GetMysqlConnection() *sql.DB {

	err := godotenv.Load()
	if err != nil {
		log.Fatal("无法加载 .env 文件")
	}

	// 访问环境变量
	dbUser := os.Getenv("MYSQL_USER")
	dbPass := os.Getenv("MYSQL_PASSWORD")
	dbUnixSocket := os.Getenv("MYSQL_UNIXSOCKET")
	dbName := os.Getenv("MYSQL_DATABASE")
	// 构建连接字符串
	connStr := fmt.Sprintf("%s:%s@unix(%s)/%s", dbUser, dbPass, dbUnixSocket, dbName)

	// 打开数据库连接
	db, err := sql.Open("mysql", connStr)
	util.FailOnError(err, "连接数据库失败")

	return db
}
