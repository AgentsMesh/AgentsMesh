package extension

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/anthropics/agentsmesh/backend/internal/domain/extension"
)

// McpRegistrySyncer synchronizes the official MCP Registry data into the local
// mcp_market_items table.
type McpRegistrySyncer struct {
	client *McpRegistryClient
	repo   extension.Repository
}

func NewMcpRegistrySyncer(client *McpRegistryClient, repo extension.Repository) *McpRegistrySyncer {
	return &McpRegistrySyncer{
		client: client,
		repo:   repo,
	}
}

func (s *McpRegistrySyncer) Sync(ctx context.Context) error {
	entries, err := s.client.FetchAll(ctx)
	if err != nil {
		return fmt.Errorf("fetch registry: %w", err)
	}

	slog.InfoContext(ctx, "MCP Registry sync: fetched entries", "count", len(entries))

	now := time.Now()
	var items []*extension.McpMarketItem
	var synced []string
	var skipped int

	for _, entry := range entries {
		if ctx.Err() != nil {
			return ctx.Err()
		}

		item, err := s.convertToMarketItem(entry, now)
		if err != nil {
			slog.WarnContext(ctx, "MCP Registry sync: skipping entry",
				"name", entry.Server.Name, "error", err)
			skipped++
			continue
		}

		items = append(items, item)
		synced = append(synced, item.RegistryName)
	}

	if len(items) > 0 {
		if err := s.repo.BatchUpsertMcpMarketItems(ctx, items); err != nil {
			return fmt.Errorf("batch upsert: %w", err)
		}
	}

	deactivated, err := s.repo.DeactivateMcpMarketItemsNotIn(ctx, extension.McpSourceRegistry, synced)
	if err != nil {
		slog.WarnContext(ctx, "MCP Registry sync: deactivation failed", "error", err)
	}

	slog.InfoContext(ctx, "MCP Registry sync completed",
		"total", len(entries),
		"synced", len(synced),
		"skipped", skipped,
		"deactivated", deactivated,
	)
	return nil
}

func (s *McpRegistrySyncer) convertToMarketItem(entry RegistryServerEntry, now time.Time) (*extension.McpMarketItem, error) {
	srv := entry.Server

	if srv.Name == "" {
		return nil, fmt.Errorf("server has no name")
	}

	displayName := srv.Title
	if displayName == "" {
		parts := strings.Split(srv.Name, "/")
		displayName = parts[len(parts)-1]
		if displayName == "" {
			displayName = srv.Name
		}
	}

	slug := registryNameToSlug(srv.Name)

	item := &extension.McpMarketItem{
		Slug:         slug,
		Name:         displayName,
		Description:  srv.Description,
		IsActive:     true,
		Source:       extension.McpSourceRegistry,
		RegistryName: srv.Name,
		Version:      srv.Version,
		RegistryMeta: entry.Meta,
		LastSyncedAt: &now,
	}

	if srv.Repository != nil {
		item.RepositoryURL = srv.Repository.URL
	}

	s.applyPackageConfig(item, srv.Packages)

	if item.TransportType == "" {
		s.applyRemoteConfig(item, srv.Remotes)
	}

	if item.TransportType == "" {
		return nil, fmt.Errorf("no usable package or remote for %s", srv.Name)
	}

	return item, nil
}

func (s *McpRegistrySyncer) applyPackageConfig(item *extension.McpMarketItem, packages []RegistryPackage) {
	var best *RegistryPackage
	for i := range packages {
		pkg := &packages[i]
		if best == nil {
			best = pkg
			continue
		}
		if pkgPriority(pkg.RegistryType) < pkgPriority(best.RegistryType) {
			best = pkg
		}
	}

	if best == nil {
		return
	}

	item.TransportType = extension.TransportTypeStdio
	if best.Transport.Type != "" {
		item.TransportType = best.Transport.Type
	}

	switch best.RegistryType {
	case "npm":
		item.Command = "npx"
		args := []string{"-y", best.Identifier}
		argsJSON, _ := json.Marshal(args)
		item.DefaultArgs = argsJSON
	case "pypi":
		item.Command = "uvx"
		args := []string{best.Identifier}
		argsJSON, _ := json.Marshal(args)
		item.DefaultArgs = argsJSON
	case "oci":
		item.Command = "docker"
		args := []string{"run", "-i", "--rm", best.Identifier}
		argsJSON, _ := json.Marshal(args)
		item.DefaultArgs = argsJSON
	default:
		return
	}

	item.Category = best.RegistryType

	if len(best.EnvironmentVariables) > 0 {
		envSchema := make([]extension.EnvVarSchemaEntry, 0, len(best.EnvironmentVariables))
		for _, ev := range best.EnvironmentVariables {
			entry := extension.EnvVarSchemaEntry{
				Name:      ev.Name,
				Label:     ev.Description,
				Required:  ev.IsRequired,
				Sensitive: ev.IsSecret,
			}
			if ev.Default != "" {
				entry.Placeholder = ev.Default
			}
			envSchema = append(envSchema, entry)
		}
		schemaJSON, _ := json.Marshal(envSchema)
		item.EnvVarSchema = schemaJSON
	}
}

func (s *McpRegistrySyncer) applyRemoteConfig(item *extension.McpMarketItem, remotes []RegistryRemote) {
	if len(remotes) == 0 {
		return
	}

	remote := remotes[0]
	switch remote.Type {
	case "sse":
		item.TransportType = extension.TransportTypeSSE
	case "streamable-http", "http":
		item.TransportType = extension.TransportTypeHTTP
	default:
		item.TransportType = remote.Type
	}

	item.DefaultHttpURL = remote.URL

	if len(remote.Headers) > 0 {
		headers := make([]map[string]interface{}, 0, len(remote.Headers))
		for _, h := range remote.Headers {
			header := map[string]interface{}{
				"name":       h.Name,
				"required":   h.IsRequired,
				"sensitive":  h.IsSecret,
			}
			if h.Description != "" {
				header["description"] = h.Description
			}
			if h.Value != "" {
				header["value"] = h.Value
			}
			headers = append(headers, header)
		}
		headersJSON, _ := json.Marshal(headers)
		item.DefaultHttpHeaders = headersJSON
	}
}

func pkgPriority(registryType string) int {
	switch registryType {
	case "npm":
		return 0
	case "pypi":
		return 1
	case "oci":
		return 2
	default:
		return 9
	}
}

func registryNameToSlug(name string) string {
	slug := strings.ReplaceAll(name, "/", "--")
	slug = strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || r == '_' || r == '.' {
			return r
		}
		return '-'
	}, slug)
	slug = strings.ToLower(slug)
	return slug
}
