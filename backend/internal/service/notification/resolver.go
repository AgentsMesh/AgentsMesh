package notification

import "context"

type RecipientResolver interface {
	Resolve(ctx context.Context, param string) ([]int64, error)
}

type PodInfoProvider interface {
	GetPodOrganizationAndCreator(ctx context.Context, podKey string) (orgID int64, creatorID int64, err error)
}

type PodCreatorResolver struct {
	podInfo PodInfoProvider
}

func NewPodCreatorResolver(podInfo PodInfoProvider) *PodCreatorResolver {
	return &PodCreatorResolver{podInfo: podInfo}
}

func (r *PodCreatorResolver) Resolve(ctx context.Context, param string) ([]int64, error) {
	_, creatorID, err := r.podInfo.GetPodOrganizationAndCreator(ctx, param)
	if err != nil {
		return nil, err
	}
	return []int64{creatorID}, nil
}
