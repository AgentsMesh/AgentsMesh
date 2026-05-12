// Package apikeyconnect hosts Connect-RPC handlers for the apikey
// domain. Mirrors backend/internal/api/rest/v1/apikeys.go but exposes
// the data plane via Connect (binary protobuf wire, see conventions.md
// §2.5). REST stays mounted in parallel; the migration runs dual-track
// until all 26 services have flipped.
//
// PR #345 lineage: CreateApiKey returns the multi-field
// {api_key, raw_key} envelope intentionally (conventions §9 exception)
// so the secret survives the wire — exactly the failure mode the proto
// migration set out to eliminate.
package apikeyconnect

import (
	"net/http"

	"connectrpc.com/connect"

	"github.com/anthropics/agentsmesh/backend/internal/middleware"
	apikeyservice "github.com/anthropics/agentsmesh/backend/internal/service/apikey"
)

// ServiceName mirrors proto.apikey.v1.ApiKeyService exactly — Connect
// derives the URL from `<package>.<Service>` (conventions §1, §12).
const ServiceName = "proto.apikey.v1.ApiKeyService"

const (
	ListApiKeysProcedure  = "/" + ServiceName + "/ListApiKeys"
	GetApiKeyProcedure    = "/" + ServiceName + "/GetApiKey"
	CreateApiKeyProcedure = "/" + ServiceName + "/CreateApiKey"
	UpdateApiKeyProcedure = "/" + ServiceName + "/UpdateApiKey"
	RevokeApiKeyProcedure = "/" + ServiceName + "/RevokeApiKey"
	DeleteApiKeyProcedure = "/" + ServiceName + "/DeleteApiKey"
)

// Server implements the ApiKeyService contract. Mirrors REST's
// APIKeyHandler dependencies (apikeys.go:15).
type Server struct {
	apiKeySvc apikeyservice.Interface
	orgSvc    middleware.OrganizationService
}

func NewServer(apiKeySvc apikeyservice.Interface, orgSvc middleware.OrganizationService) *Server {
	return &Server{apiKeySvc: apiKeySvc, orgSvc: orgSvc}
}

// Mount registers all ApiKeyService procedures on mux behind the auth
// interceptor supplied via opts (cmd/server/connect_init.go).
func Mount(mux *http.ServeMux, srv *Server, opts ...connect.HandlerOption) {
	mux.Handle(ListApiKeysProcedure, connect.NewUnaryHandler(
		ListApiKeysProcedure, srv.ListApiKeys, opts...,
	))
	mux.Handle(GetApiKeyProcedure, connect.NewUnaryHandler(
		GetApiKeyProcedure, srv.GetApiKey, opts...,
	))
	mux.Handle(CreateApiKeyProcedure, connect.NewUnaryHandler(
		CreateApiKeyProcedure, srv.CreateApiKey, opts...,
	))
	mux.Handle(UpdateApiKeyProcedure, connect.NewUnaryHandler(
		UpdateApiKeyProcedure, srv.UpdateApiKey, opts...,
	))
	mux.Handle(RevokeApiKeyProcedure, connect.NewUnaryHandler(
		RevokeApiKeyProcedure, srv.RevokeApiKey, opts...,
	))
	mux.Handle(DeleteApiKeyProcedure, connect.NewUnaryHandler(
		DeleteApiKeyProcedure, srv.DeleteApiKey, opts...,
	))
}
