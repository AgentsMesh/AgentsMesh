// Package tasks provides pipeline watching functionality.
//
// Deprecated: This file has been split into smaller files following SRP:
//   - pipeline_watcher_types.go: Constants, types, and constructor
//   - pipeline_watcher_watch.go: Watch, UpdateStatus, Unwatch operations
//   - pipeline_watcher_query.go: Query operations (GetPipeline, GetCompletedPipelines, etc.)
//   - pipeline_watcher_process.go: Processing operations (MarkProcessed, StoreArtifact)
package tasks
