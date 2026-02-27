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

// NewMcpRegistrySyncer creates a syncer that pulls from the MCP Registry.
func NewMcpRegistrySyncer(client *McpRegistryClient, repo extension.Repository) *McpRegistrySyncer {
	return &McpRegistrySyncer{
		client: client,
		repo:   repo,
	}
}

// Sync performs a full sync from the MCP Registry:
//  1. Fetch all latest+active servers from the registry API.
//  2. Convert each entry to McpMarketItem and upsert.
//  3. Deactivate local registry items that no longer exist upstream.
func (s *McpRegistrySyncer) Sync(ctx context.Context) error {
	entries, err := s.client.FetchAll(ctx)
	if err != nil {
		return fmt.Errorf("fetch registry: %w", err)
	}

	slog.Info("MCP Registry sync: fetched entries", "count", len(entries))

	now := time.Now()
	var synced []string
	var created, updated, skipped int

	for _, entry := range entries {
		if ctx.Err() != nil {
			return ctx.Err()
		}

		item, err := s.convertToMarketItem(entry, now)
		if err != nil {
			slog.Warn("MCP Registry sync: skipping entry",
				"name", entry.Server.Name, "error", err)
			skipped++
			continue
		}

		if err := s.repo.UpsertMcpMarketItem(ctx, item); err != nil {
			slog.Warn("MCP Registry sync: upsert failed",
				"slug", item.Slug, "error", err)
			skipped++
			continue
		}

		// Track whether this was a create or update for logging
		if item.ID == 0 {
			created++
		} else {
			updated++
		}
		synced = append(synced, item.RegistryName)
	}

	// Deactivate registry items no longer present in the upstream registry
	deactivated, err := s.repo.DeactivateMcpMarketItemsNotIn(ctx, extension.McpSourceRegistry, synced)
	if err != nil {
		slog.Warn("MCP Registry sync: deactivation failed", "error", err)
	}

	slog.Info("MCP Registry sync completed",
		"total", len(entries),
		"synced", len(synced),
		"skipped", skipped,
		"deactivated", deactivated,
	)
	return nil
}

// convertToMarketItem transforms a registry entry into a McpMarketItem.
func (s *McpRegistrySyncer) convertToMarketItem(entry RegistryServerEntry, now time.Time) (*extension.McpMarketItem, error) {
	srv := entry.Server

	if srv.Name == "" {
		return nil, fmt.Errorf("server has no name")
	}

	// Determine display name: prefer title, fallback to last part of name
	displayName := srv.Title
	if displayName == "" {
		parts := strings.Split(srv.Name, "/")
		displayName = parts[len(parts)-1]
		if displayName == "" {
			displayName = srv.Name
		}
	}

	// Generate slug from registry name: replace non-alphanumeric with dashes
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

	// Determine transport, command, args from packages (prefer stdio/npm)
	s.applyPackageConfig(item, srv.Packages)

	// If no package config was applied, try remotes
	if item.TransportType == "" {
		s.applyRemoteConfig(item, srv.Remotes)
	}

	// If still no transport type, skip this entry
	if item.TransportType == "" {
		return nil, fmt.Errorf("no usable package or remote for %s", srv.Name)
	}

	return item, nil
}

// applyPackageConfig extracts command/args/env from the first usable package.
// Priority: npm > pypi > oci.
func (s *McpRegistrySyncer) applyPackageConfig(item *extension.McpMarketItem, packages []RegistryPackage) {
	// Sort by preference: npm first, then pypi, then oci
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
		// Unknown registry type, skip command setup
		return
	}

	item.Category = best.RegistryType

	// Convert environment variables
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

// applyRemoteConfig extracts HTTP URL and headers from the first usable remote.
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

	// Convert headers to schema
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

// pkgPriority returns a sort priority (lower = preferred).
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

// registryNameToSlug converts a registry name like "io.github.user/server-name"
// to a URL-safe slug like "io-github-user--server-name".
func registryNameToSlug(name string) string {
	// Replace / with -- to preserve structure readability
	slug := strings.ReplaceAll(name, "/", "--")
	// Replace any remaining non-slug characters
	slug = strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || r == '_' || r == '.' {
			return r
		}
		return '-'
	}, slug)
	// Lowercase
	slug = strings.ToLower(slug)
	return slug
}
