package runner

import (
	"context"
	"sync"
)

const (
	routerShards = 64
)

type PodInfoGetter interface {
	GetPodOrganizationAndCreator(ctx context.Context, podKey string) (orgID, creatorID int64, err error)
	UpdatePodTitle(ctx context.Context, podKey, title string) error
}

type routerShard struct {
	podRunnerMap map[string]int64 // pod -> runner mapping
	mu           sync.RWMutex
}

func newRouterShard() *routerShard {
	return &routerShard{
		podRunnerMap: make(map[string]int64),
	}
}
