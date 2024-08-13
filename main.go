package main

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"net"
	"os"
	"sync"
	"time"

	dragonPb "grpc-dragonfly-usage/proto-collection/pb/dragonService"

	"github.com/redis/go-redis/v9"
	"google.golang.org/grpc"
)

var (
	totalProcessingTime time.Duration
)

const (
	defaultDBAddr     = "localhost:6379"
	defaultServerPort = "50051"
)

type server struct {
	dragonPb.UnimplementedDragonServiceServer
	rdb *redis.Client
}

func dragonflyTestRoutine(ctx context.Context, s *server, wg *sync.WaitGroup) {
	defer wg.Done()
	routineStart := time.Now()

	key := fmt.Sprintf("key-%d", rand.Int())
	value := fmt.Sprintf("value-%d", rand.Int())

	ttl := time.Duration(0) * time.Second
	err := s.rdb.Set(ctx, key, value, ttl).Err()
	if err != nil {
		fmt.Printf("Error setting key in Redis: %v\n", err)
		return
	}

	// resultBeforeSleep, _ := s.rdb.Get(ctx, key).Result()
	// if resultBeforeSleep != value {
	// 	fmt.Printf("❌ Error mismatched value from DB: %v, ttl: %v\n", err, ttl)
	// }

	sleepDuration := ttl
	time.Sleep(sleepDuration)

	// resultAfterSleep, err2 := s.rdb.Get(ctx, key).Result()
	// if err2 == redis.Nil && resultAfterSleep == "" {
	// 	//do nothing
	// } else if err2 != nil {
	// 	fmt.Printf("❌ Error getting key from DB: %v\n", err)
	// } else {
	// 	fmt.Printf("🐱 Key %s did not expire. ttl: %v, Value: %s\n", key, ttl, resultAfterSleep)
	// }

	routineEnd := time.Now()
	extraTime := routineEnd.Sub(routineStart) - sleepDuration
	totalProcessingTime += extraTime
	if extraTime > 0 {
		fmt.Printf("Routine took extra time: %v\n", extraTime)
	}
}

func (s *server) LoadTestCall(ctx context.Context, param *dragonPb.LoadTestRequest) (*dragonPb.LoadTestResponse, error) {
	startTime := time.Now()
	var wg sync.WaitGroup

	fmt.Println("A new test has started: ", startTime)
	for i := int64(0); i < param.GetNoOfRoutines(); i++ {
		wg.Add(1)
		go dragonflyTestRoutine(ctx, s, &wg)
	}

	wg.Wait()
	totalTime := time.Since(startTime)

	avgProcessingTime := int64(totalProcessingTime.Nanoseconds()) / param.GetNoOfRoutines()

	fmt.Printf("🎬 Load test completed. Avg processing time taken: %v\n", time.Duration(avgProcessingTime))

	fmt.Printf("🎬 Total time taken: %v", totalTime)
	return &dragonPb.LoadTestResponse{
		Message: fmt.Sprintf("Load test completed. Total time taken: %v", totalTime),
	}, nil
}

func main() {
	dbAddr := os.Getenv("DB_ADDR")
	serverPortAddr := os.Getenv("SERVER_PORT_ADDR")

	if dbAddr == "" {
		dbAddr = defaultDBAddr
	}
	if serverPortAddr == "" {
		serverPortAddr = defaultServerPort
	}

	rdb := redis.NewClient(&redis.Options{
		Addr: dbAddr,
	})

	lis, err := net.Listen("tcp", fmt.Sprint(":", serverPortAddr))
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	s := grpc.NewServer()
	dragonPb.RegisterDragonServiceServer(s, &server{rdb: rdb})

	log.Printf("gRPC server listening on %s", lis.Addr())
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}

// func (s *server) SetValue(ctx context.Context, req *dragonPb.SetValueRequest) (*dragonPb.SetValueResponse, error) {
// 	err := s.rdb.Set(ctx, req.GetKey(), req.GetValue(), 0).Err()
// 	if err != nil {
// 		return nil, err
// 	}
// 	return &dragonPb.SetValueResponse{Message: "Success"}, nil
// }

// func (s *server) GetValue(ctx context.Context, req *dragonPb.GetValueRequest) (*dragonPb.GetValueResponse, error) {
// 	val, err := s.rdb.Get(ctx, req.GetKey()).Result()
// 	if err != nil {
// 		return nil, err
// 	}
// 	return &dragonPb.GetValueResponse{Value: val}, nil
// }
