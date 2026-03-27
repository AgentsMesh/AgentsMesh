package extension

import (
	"testing"
)

// ---------------------------------------------------------------------------
// InstalledSkill.GetEffectiveSha / GetEffectiveStorageKey / GetEffectivePackageSize
// ---------------------------------------------------------------------------

func TestInstalledSkill_GetEffectiveSha(t *testing.T) {
	pinnedVersion := 5

	tests := []struct {
		name  string
		skill InstalledSkill
		want  string
	}{
		{
			name: "market_tracking_latest",
			skill: InstalledSkill{
				InstallSource: InstallSourceMarket,
				PinnedVersion: nil,
				ContentSha:    "own-sha",
				MarketItem:    &SkillMarketItem{ContentSha: "market-sha"},
			},
			want: "market-sha",
		},
		{
			name: "market_pinned_version",
			skill: InstalledSkill{
				InstallSource: InstallSourceMarket,
				PinnedVersion: &pinnedVersion,
				ContentSha:    "own-sha",
				MarketItem:    &SkillMarketItem{ContentSha: "market-sha"},
			},
			want: "own-sha",
		},
		{
			name: "market_no_market_item",
			skill: InstalledSkill{
				InstallSource: InstallSourceMarket,
				PinnedVersion: nil,
				ContentSha:    "own-sha",
				MarketItem:    nil,
			},
			want: "own-sha",
		},
		{
			name: "github_source",
			skill: InstalledSkill{
				InstallSource: InstallSourceGitHub,
				ContentSha:    "github-sha",
				MarketItem:    &SkillMarketItem{ContentSha: "market-sha"},
			},
			want: "github-sha",
		},
		{
			name: "upload_source",
			skill: InstalledSkill{
				InstallSource: InstallSourceUpload,
				ContentSha:    "upload-sha",
			},
			want: "upload-sha",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.skill.GetEffectiveSha()
			if got != tt.want {
				t.Errorf("GetEffectiveSha() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestInstalledSkill_GetEffectiveStorageKey(t *testing.T) {
	pinnedVersion := 5

	tests := []struct {
		name  string
		skill InstalledSkill
		want  string
	}{
		{
			name: "market_tracking_latest",
			skill: InstalledSkill{
				InstallSource: InstallSourceMarket,
				PinnedVersion: nil,
				StorageKey:    "own-key",
				MarketItem:    &SkillMarketItem{StorageKey: "market-key"},
			},
			want: "market-key",
		},
		{
			name: "market_pinned_version",
			skill: InstalledSkill{
				InstallSource: InstallSourceMarket,
				PinnedVersion: &pinnedVersion,
				StorageKey:    "own-key",
				MarketItem:    &SkillMarketItem{StorageKey: "market-key"},
			},
			want: "own-key",
		},
		{
			name: "market_no_market_item",
			skill: InstalledSkill{
				InstallSource: InstallSourceMarket,
				PinnedVersion: nil,
				StorageKey:    "own-key",
				MarketItem:    nil,
			},
			want: "own-key",
		},
		{
			name: "github_source",
			skill: InstalledSkill{
				InstallSource: InstallSourceGitHub,
				StorageKey:    "github-key",
				MarketItem:    &SkillMarketItem{StorageKey: "market-key"},
			},
			want: "github-key",
		},
		{
			name: "upload_source",
			skill: InstalledSkill{
				InstallSource: InstallSourceUpload,
				StorageKey:    "upload-key",
			},
			want: "upload-key",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.skill.GetEffectiveStorageKey()
			if got != tt.want {
				t.Errorf("GetEffectiveStorageKey() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestInstalledSkill_GetEffectivePackageSize(t *testing.T) {
	pinnedVersion := 5

	tests := []struct {
		name  string
		skill InstalledSkill
		want  int64
	}{
		{
			name: "market_tracking_latest",
			skill: InstalledSkill{
				InstallSource: InstallSourceMarket,
				PinnedVersion: nil,
				PackageSize:   100,
				MarketItem:    &SkillMarketItem{PackageSize: 999},
			},
			want: 999,
		},
		{
			name: "market_pinned_version",
			skill: InstalledSkill{
				InstallSource: InstallSourceMarket,
				PinnedVersion: &pinnedVersion,
				PackageSize:   100,
				MarketItem:    &SkillMarketItem{PackageSize: 999},
			},
			want: 100,
		},
		{
			name: "market_no_market_item",
			skill: InstalledSkill{
				InstallSource: InstallSourceMarket,
				PinnedVersion: nil,
				PackageSize:   100,
				MarketItem:    nil,
			},
			want: 100,
		},
		{
			name: "github_source",
			skill: InstalledSkill{
				InstallSource: InstallSourceGitHub,
				PackageSize:   200,
				MarketItem:    &SkillMarketItem{PackageSize: 999},
			},
			want: 200,
		},
		{
			name: "upload_source",
			skill: InstalledSkill{
				InstallSource: InstallSourceUpload,
				PackageSize:   300,
			},
			want: 300,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.skill.GetEffectivePackageSize()
			if got != tt.want {
				t.Errorf("GetEffectivePackageSize() = %d, want %d", got, tt.want)
			}
		})
	}
}
