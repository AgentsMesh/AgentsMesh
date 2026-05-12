// Package connect contains Connect-RPC handlers for AgentsMesh's data plane.
//
// Service handlers live in subpackages (e.g. `connect/extension/`,
// `connect/pod/`), each mounting onto the shared http.ServeMux via the
// pattern set up in backend/cmd/server/connect_init.go.
//
// This file is a package placeholder; the package will gain actual
// handlers as services are migrated from REST to Connect-RPC. See
// .claude/plans/proto-migration-adr.md for the migration plan.
package connect
