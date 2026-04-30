package store

import (
	"context"
	"fmt"
	"testing"

	"github.com/redis/go-redis/v9"
)

func TestRedis(t *testing.T) {
	url, _ := redisOptionsFromEnv()
	if url != nil {

		//return o, nil
		t.Log("start")
		rdb := redis.NewClient(url)
		ctx := context.Background()
		err := rdb.Ping(ctx).Err()
		if err != nil {
			t.Fatalf("rdb.Ping() error = %v", err)
		}
		fmt.Println("Ping success")
		rdb.Set(ctx, "test", "test", 0).Err()
		val, err := rdb.Get(ctx, "test").Result()
		if err != nil {
			t.Fatalf("rdb.Get() error = %v", err)
		}
		fmt.Println(val)
	}

}
