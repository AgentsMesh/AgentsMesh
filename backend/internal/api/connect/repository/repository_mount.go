package repositoryconnect

import (
	"errors"
	"net/http"

	"connectrpc.com/connect"

	billingservice "github.com/anthropics/agentsmesh/backend/internal/service/billing"
	repositoryservice "github.com/anthropics/agentsmesh/backend/internal/service/repository"
)

// mapServiceError mirrors REST's handler-by-handler `apierr` mapping
// (repositories_*.go). Translates repository-domain sentinels to Connect
// codes per conventions §10.
func mapServiceError(err error) error {
	if err == nil {
		return nil
	}
	switch {
	case errors.Is(err, repositoryservice.ErrRepositoryNotFound),
		errors.Is(err, repositoryservice.ErrWebhookNotFound):
		return connect.NewError(connect.CodeNotFound, err)
	case errors.Is(err, repositoryservice.ErrNoPermission):
		return connect.NewError(connect.CodePermissionDenied, err)
	case errors.Is(err, repositoryservice.ErrRepositoryExists):
		return connect.NewError(connect.CodeAlreadyExists, err)
	case errors.Is(err, repositoryservice.ErrRepositoryHasLoopRefs):
		return connect.NewError(connect.CodeFailedPrecondition, err)
	default:
		return connect.NewError(connect.CodeInternal, err)
	}
}

// mapBillingError keeps the REST billing-quota response semantics (402-style
// codes mapped to Connect codes per conventions §10). REST returned 402; the
// closest Connect code semantically is ResourceExhausted (quota) or
// FailedPrecondition (frozen subscription). TS surfaces the paywall based
// on the code.
func mapBillingError(err error) error {
	switch {
	case errors.Is(err, billingservice.ErrQuotaExceeded):
		return connect.NewError(connect.CodeResourceExhausted, err)
	case errors.Is(err, billingservice.ErrSubscriptionFrozen):
		return connect.NewError(connect.CodeFailedPrecondition, err)
	default:
		return connect.NewError(connect.CodeInternal, err)
	}
}

// Mount registers all RepositoryService procedures on mux behind the
// auth interceptor supplied via opts (see cmd/server/connect_init.go).
func Mount(mux *http.ServeMux, srv *Server, opts ...connect.HandlerOption) {
	mux.Handle(ListRepositoriesProcedure, connect.NewUnaryHandler(
		ListRepositoriesProcedure, srv.ListRepositories, opts...,
	))
	mux.Handle(GetRepositoryProcedure, connect.NewUnaryHandler(
		GetRepositoryProcedure, srv.GetRepository, opts...,
	))
	mux.Handle(CreateRepositoryProcedure, connect.NewUnaryHandler(
		CreateRepositoryProcedure, srv.CreateRepository, opts...,
	))
	mux.Handle(UpdateRepositoryProcedure, connect.NewUnaryHandler(
		UpdateRepositoryProcedure, srv.UpdateRepository, opts...,
	))
	mux.Handle(DeleteRepositoryProcedure, connect.NewUnaryHandler(
		DeleteRepositoryProcedure, srv.DeleteRepository, opts...,
	))
	mux.Handle(ListRepositoryBranchesProcedure, connect.NewUnaryHandler(
		ListRepositoryBranchesProcedure, srv.ListRepositoryBranches, opts...,
	))
	mux.Handle(SyncRepositoryBranchesProcedure, connect.NewUnaryHandler(
		SyncRepositoryBranchesProcedure, srv.SyncRepositoryBranches, opts...,
	))
	mux.Handle(ListRepositoryMergeRequestsProcedure, connect.NewUnaryHandler(
		ListRepositoryMergeRequestsProcedure, srv.ListRepositoryMergeRequests, opts...,
	))
	mux.Handle(RegisterRepositoryWebhookProcedure, connect.NewUnaryHandler(
		RegisterRepositoryWebhookProcedure, srv.RegisterRepositoryWebhook, opts...,
	))
	mux.Handle(DeleteRepositoryWebhookProcedure, connect.NewUnaryHandler(
		DeleteRepositoryWebhookProcedure, srv.DeleteRepositoryWebhook, opts...,
	))
	mux.Handle(GetRepositoryWebhookStatusProcedure, connect.NewUnaryHandler(
		GetRepositoryWebhookStatusProcedure, srv.GetRepositoryWebhookStatus, opts...,
	))
	mux.Handle(GetRepositoryWebhookSecretProcedure, connect.NewUnaryHandler(
		GetRepositoryWebhookSecretProcedure, srv.GetRepositoryWebhookSecret, opts...,
	))
	mux.Handle(MarkRepositoryWebhookConfiguredProcedure, connect.NewUnaryHandler(
		MarkRepositoryWebhookConfiguredProcedure, srv.MarkRepositoryWebhookConfigured, opts...,
	))
}
