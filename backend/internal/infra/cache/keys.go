package cache

import "fmt"

var (
	ErrNotFound = fmt.Errorf("key not found")
)

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

func PodKey(podKey string) string {
	return PrefixPod + podKey
}

func UserKey(userID int64) string {
	return fmt.Sprintf("%s%d", PrefixUser, userID)
}

func OrgKey(orgID int64) string {
	return fmt.Sprintf("%s%d", PrefixOrg, orgID)
}

func RunnerKey(runnerID int64) string {
	return fmt.Sprintf("%s%d", PrefixRunner, runnerID)
}

func ChannelKey(channelID int64) string {
	return fmt.Sprintf("%s%d", PrefixChannel, channelID)
}

func RateLimitKey(identifier string) string {
	return PrefixRateLimit + identifier
}

func LockKey(resource string) string {
	return PrefixLock + resource
}

func PubSubChannel(channelType string, id int64) string {
	return fmt.Sprintf("%s%s:%d", PrefixPubSub, channelType, id)
}
