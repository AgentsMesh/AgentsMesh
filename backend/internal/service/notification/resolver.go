package notification

import "context"

// RecipientResolver resolves a parameter string into a list of user IDs
type RecipientResolver interface {
	Resolve(ctx context.Context, param string) ([]int64, error)
}

// PodInfoProvider abstracts pod info lookup to avoid importing runner service
type PodInfoProvider interface {
	GetPodOrganizationAndCreator(ctx context.Context, podKey string) (orgID int64, creatorID int64, err error)
}

// PodCreatorResolver resolves pod key to the creator's user ID
type PodCreatorResolver struct {
	podInfo PodInfoProvider
}

// NewPodCreatorResolver creates a new PodCreatorResolver
func NewPodCreatorResolver(podInfo PodInfoProvider) *PodCreatorResolver {
	return &PodCreatorResolver{podInfo: podInfo}
}

// Resolve returns the pod creator's user ID for the given pod key
func (r *PodCreatorResolver) Resolve(ctx context.Context, param string) ([]int64, error) {
	_, creatorID, err := r.podInfo.GetPodOrganizationAndCreator(ctx, param)
	if err != nil {
		return nil, err
	}
	return []int64{creatorID}, nil
}
