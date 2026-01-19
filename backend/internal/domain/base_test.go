package domain

import (
	"testing"
	"time"
)

func TestBaseModelStruct(t *testing.T) {
	now := time.Now()
	base := BaseModel{
		ID:        1,
		CreatedAt: now,
		UpdatedAt: now,
	}

	if base.ID != 1 {
		t.Errorf("expected ID 1, got %d", base.ID)
	}
	if base.CreatedAt != now {
		t.Error("expected CreatedAt to match")
	}
	if base.UpdatedAt != now {
		t.Error("expected UpdatedAt to match")
	}
}

func TestTenantModelStruct(t *testing.T) {
	now := time.Now()
	tenant := TenantModel{
		BaseModel: BaseModel{
			ID:        1,
			CreatedAt: now,
			UpdatedAt: now,
		},
		OrganizationID: 100,
	}

	if tenant.ID != 1 {
		t.Errorf("expected ID 1, got %d", tenant.ID)
	}
	if tenant.OrganizationID != 100 {
		t.Errorf("expected OrganizationID 100, got %d", tenant.OrganizationID)
	}
}

// --- Benchmark Tests ---

func BenchmarkBaseModelCreation(b *testing.B) {
	now := time.Now()
	for i := 0; i < b.N; i++ {
		_ = BaseModel{
			ID:        int64(i),
			CreatedAt: now,
			UpdatedAt: now,
		}
	}
}

func BenchmarkTenantModelCreation(b *testing.B) {
	now := time.Now()
	for i := 0; i < b.N; i++ {
		_ = TenantModel{
			BaseModel: BaseModel{
				ID:        int64(i),
				CreatedAt: now,
				UpdatedAt: now,
			},
			OrganizationID: 100,
		}
	}
}
