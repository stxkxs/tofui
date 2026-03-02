package logstream

import (
	"context"
	"log/slog"
	"sync"

	"github.com/redis/go-redis/v9"
)

// RedisStreamer uses Redis pub/sub for cross-process log fan-out.
type RedisStreamer struct {
	rdb *redis.Client

	mu      sync.RWMutex
	streams map[string]*redisStream
}

type redisStream struct {
	subs   []chan []byte
	cancel context.CancelFunc
}

func NewRedisStreamer(redisURL string) (*RedisStreamer, error) {
	opts, err := redis.ParseURL(redisURL)
	if err != nil {
		return nil, err
	}

	rdb := redis.NewClient(opts)
	if err := rdb.Ping(context.Background()).Err(); err != nil {
		return nil, err
	}

	return &RedisStreamer{
		rdb:     rdb,
		streams: make(map[string]*redisStream),
	}, nil
}

func channelName(runID string) string {
	return "run:logs:" + runID
}

func (s *RedisStreamer) Publish(runID string, data []byte) {
	// Publish to Redis so other processes can receive
	err := s.rdb.Publish(context.Background(), channelName(runID), data).Err()
	if err != nil {
		slog.Warn("redis publish failed", "run_id", runID, "error", err)
	}
}

func (s *RedisStreamer) Subscribe(runID string) <-chan []byte {
	s.mu.Lock()
	defer s.mu.Unlock()

	ch := make(chan []byte, 256)

	rs, exists := s.streams[runID]
	if !exists {
		ctx, cancel := context.WithCancel(context.Background())
		rs = &redisStream{cancel: cancel}
		s.streams[runID] = rs

		// Start Redis subscription goroutine
		go s.receiveFromRedis(ctx, runID)
	}

	rs.subs = append(rs.subs, ch)
	return ch
}

func (s *RedisStreamer) receiveFromRedis(ctx context.Context, runID string) {
	pubsub := s.rdb.Subscribe(ctx, channelName(runID))
	defer pubsub.Close()

	ch := pubsub.Channel()
	for {
		select {
		case <-ctx.Done():
			return
		case msg, ok := <-ch:
			if !ok {
				return
			}
			s.fanOut(runID, []byte(msg.Payload))
		}
	}
}

func (s *RedisStreamer) fanOut(runID string, data []byte) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	rs, exists := s.streams[runID]
	if !exists {
		return
	}

	for _, ch := range rs.subs {
		select {
		case ch <- data:
		default:
			// Drop message if subscriber is slow
		}
	}
}

func (s *RedisStreamer) Unsubscribe(runID string, ch <-chan []byte) {
	s.mu.Lock()
	defer s.mu.Unlock()

	rs, exists := s.streams[runID]
	if !exists {
		return
	}

	for i, sub := range rs.subs {
		if sub == ch {
			rs.subs = append(rs.subs[:i], rs.subs[i+1:]...)
			close(sub)
			break
		}
	}

	// If no more subscribers, clean up Redis subscription
	if len(rs.subs) == 0 {
		rs.cancel()
		delete(s.streams, runID)
	}
}

func (s *RedisStreamer) Close(runID string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	rs, exists := s.streams[runID]
	if !exists {
		return
	}

	rs.cancel()
	for _, ch := range rs.subs {
		close(ch)
	}
	delete(s.streams, runID)
}
