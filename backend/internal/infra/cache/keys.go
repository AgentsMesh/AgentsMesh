package cache

import "fmt"

// Errors
var (
	ErrNotFound = fmt.Errorf("key not found")
)

// Key prefixes for different data types
const (
	PrefixPod       = "pod:"
	PrefixUser      = "user:"
	PrefixOrg       = "org:"
	PrefixRunner    = "runner:"
	PrefixChannel   = "channel:"
	PrefixRateLimit = "ratelimit:"
	PrefixLock      = "lock:"
	PrefixPubSub    = "pubsub:"
)

// PodKey returns a cache key for a pod
func PodKey(podKey string) string {
	return PrefixPod + podKey
}

// UserKey returns a cache key for a user
func UserKey(userID int64) string {
	return fmt.Sprintf("%s%d", PrefixUser, userID)
}

// OrgKey returns a cache key for an organization
func OrgKey(orgID int64) string {
	return fmt.Sprintf("%s%d", PrefixOrg, orgID)
}

// RunnerKey returns a cache key for a runner
func RunnerKey(runnerID int64) string {
	return fmt.Sprintf("%s%d", PrefixRunner, runnerID)
}

// ChannelKey returns a cache key for a channel
func ChannelKey(channelID int64) string {
	return fmt.Sprintf("%s%d", PrefixChannel, channelID)
}

// RateLimitKey returns a cache key for rate limiting
func RateLimitKey(identifier string) string {
	return PrefixRateLimit + identifier
}

// LockKey returns a cache key for distributed locking
func LockKey(resource string) string {
	return PrefixLock + resource
}

// PubSubChannel returns a pub/sub channel name
func PubSubChannel(channelType string, id int64) string {
	return fmt.Sprintf("%s%s:%d", PrefixPubSub, channelType, id)
}
