package grant

import "context"

type Repository interface {
	Create(ctx context.Context, g *ResourceGrant) error
	Delete(ctx context.Context, resourceType, resourceID string, grantID int64) error
	ListByResource(ctx context.Context, resourceType, resourceID string) ([]*ResourceGrant, error)
	GetGrantedUserIDs(ctx context.Context, resourceType, resourceID string) ([]int64, error)
	GetGrantedResourceIDs(ctx context.Context, resourceType string, userID int64, orgID int64) ([]string, error)
	DeleteByResource(ctx context.Context, resourceType, resourceID string) error
}
