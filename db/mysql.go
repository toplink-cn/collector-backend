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
		log.Println("无法加载 .env 文件")
	}

	dbPass := os.Getenv("MYSQL_PASSWORD")
	// 构建连接字符串
	connStr := fmt.Sprintf("%s:%s@unix(%s)/%s", "dcim", dbPass, "/app/run/mysqld.sock", "dcim")

	// 打开数据库连接
	db, err := sql.Open("mysql", connStr)
	util.FailOnError(err, "连接数据库失败")

	return db
}
