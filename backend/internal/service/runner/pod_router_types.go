package runner

import (
	"context"
	"sync"
)

const (
	// routerShards is the number of shards for pod router data partitioning
	// 64 shards reduce lock contention for 500K pods
	routerShards = 64
)

// PodInfoGetter interface for getting and updating pod information
type PodInfoGetter interface {
	GetPodOrganizationAndCreator(ctx context.Context, podKey string) (orgID, creatorID int64, err error)
	UpdatePodTitle(ctx context.Context, podKey, title string) error
}

// routerShard holds a subset of pod router data with its own lock
// Note: After Relay migration, VirtualTerminal and client management moved to Runner/Relay
type routerShard struct {
	podRunnerMap map[string]int64 // pod -> runner mapping
	mu           sync.RWMutex
}

// newRouterShard creates a new router shard with initialized maps
func newRouterShard() *routerShard {
	return &routerShard{
		podRunnerMap: make(map[string]int64),
	}
}
