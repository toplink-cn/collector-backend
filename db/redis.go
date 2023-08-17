package db

import (
	"collector-backend/util"
	"context"
	"runtime"

	"github.com/go-redis/redis/v8"
)

const PoolCapPreCoreNum = 10

func GetRedisConnection() *redis.Client {

	numCPU := runtime.NumCPU()
	poolCap := PoolCapPreCoreNum * numCPU

	client := redis.NewClient(&redis.Options{
		Network:  "unix",
		Addr:     "/app/run/redis.sock",
		DB:       0,
		PoolSize: int(poolCap),
	})

	// 使用 Ping() 方法检查是否成功连接到 Redis
	_, err := client.Ping(context.Background()).Result()
	if err != nil {
		util.FailOnError(err, "连接Redis失败")
	}

	return client
}
