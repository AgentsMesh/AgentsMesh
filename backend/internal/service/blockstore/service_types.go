package blockstoreservice

import (
	"context"
	"log/slog"
	"sync"
	"sync/atomic"
	"time"

	"github.com/anthropics/agentsmesh/backend/internal/domain/blockstore"
	blockstoreinfra "github.com/anthropics/agentsmesh/backend/internal/infra/blockstore"
	"github.com/anthropics/agentsmesh/backend/internal/infra/otel"
)

type Service struct {
	repo      blockstore.Repository
	publisher *blockstoreinfra.OpPublisher
	embedder  EmbeddingProvider
	logger    *slog.Logger

	// Embeddings drain non-blocking; queue-full drops are tolerated (auxiliary index).
	// embedClosed gates late enqueues so they don't leak into embedInflight after Close.
	embedQueue    chan embedJob
	embedWG       sync.WaitGroup
	embedInflight sync.WaitGroup
	embedCancel   context.CancelFunc
	embedClosed   atomic.Bool

	warnDefaultEmbedderOnce sync.Once
}

type embedJob struct {
	ops []*blockstore.BlockOp
}

func NewService(repo blockstore.Repository, logger *slog.Logger) *Service {
	if logger == nil {
		logger = slog.Default()
	}
	s := &Service{
		repo:       repo,
		embedder:   NewHashEmbedder(256),
		logger:     logger,
		embedQueue: make(chan embedJob, 256),
	}
	s.startEmbedWorker()
	return s
}

func (s *Service) startEmbedWorker() {
	ctx, cancel := context.WithCancel(context.Background())
	s.embedCancel = cancel
	s.embedWG.Add(1)
	go func() {
		defer s.embedWG.Done()
		for {
			select {
			case <-ctx.Done():
				return
			case job, ok := <-s.embedQueue:
				if !ok {
					return
				}
				if s.embedder != nil {
					otel.BlockstoreEmbedQueue.Add(ctx, -1)
					embedStart := time.Now()
					s.refreshEmbeddings(ctx, job.ops)
					otel.BlockstoreEmbedDuration.Record(ctx,
						float64(time.Since(embedStart).Milliseconds()))
				}
				s.embedInflight.Done()
			}
		}
	}()
}

func (s *Service) Close() {
	if !s.embedClosed.CompareAndSwap(false, true) {
		return
	}
	if s.embedCancel != nil {
		s.embedCancel()
	}
	s.embedWG.Wait()
}

func (s *Service) enqueueEmbeddings(ops []*blockstore.BlockOp) {
	if s.embedder == nil || len(ops) == 0 || s.embedClosed.Load() {
		return
	}
	s.embedInflight.Add(1)
	select {
	case s.embedQueue <- embedJob{ops: ops}:
		otel.BlockstoreEmbedQueue.Add(context.Background(), 1)
	default:
		s.embedInflight.Done()
		s.logger.Warn("blockstore.embedding.queue_full",
			"dropped_ops", len(ops))
	}
}

func (s *Service) FlushEmbeddings() {
	s.embedInflight.Wait()
}

func (s *Service) SetPublisher(p *blockstoreinfra.OpPublisher) {
	s.publisher = p
}

func (s *Service) SetEmbedder(e EmbeddingProvider) {
	s.embedder = e
}

func (s *Service) WarnIfDefaultEmbedder() {
	s.warnDefaultEmbedderOnce.Do(func() {
		if s.embedder == nil {
			return
		}
		if s.embedder.Model() == "hash-bow-v1" {
			s.logger.Warn("blockstore.embedder.default_in_use",
				"model", s.embedder.Model(),
				"note", "HashEmbedder is a dev fallback. Call service.SetEmbedder(openaiEmbedder) before serving traffic in production.")
		}
	})
}
