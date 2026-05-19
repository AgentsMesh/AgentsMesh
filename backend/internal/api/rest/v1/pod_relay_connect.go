package v1

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/anthropics/agentsmesh/backend/internal/domain/grant"
	"github.com/anthropics/agentsmesh/backend/internal/middleware"
	grantservice "github.com/anthropics/agentsmesh/backend/internal/service/grant"
	"github.com/anthropics/agentsmesh/backend/internal/service/geo"
	"github.com/anthropics/agentsmesh/backend/internal/service/relay"
	"github.com/anthropics/agentsmesh/backend/internal/service/runner"
	"github.com/anthropics/agentsmesh/backend/pkg/apierr"
	"github.com/anthropics/agentsmesh/backend/pkg/policy"
)

type PodConnectHandler struct {
	podService     PodServiceForHandler
	relayManager   *relay.Manager
	tokenGenerator *relay.TokenGenerator
	commandSender  runner.RunnerCommandSender
	stateReader    runner.RunnerStateReader
	geoResolver    geo.Resolver
	grantService   *grantservice.Service
}

func NewPodConnectHandler(
	podService PodServiceForHandler,
	relayManager *relay.Manager,
	tokenGenerator *relay.TokenGenerator,
	commandSender runner.RunnerCommandSender,
	stateReader runner.RunnerStateReader,
	geoResolver geo.Resolver,
	grantSvc ...*grantservice.Service,
) *PodConnectHandler {
	h := &PodConnectHandler{
		podService:     podService,
		relayManager:   relayManager,
		tokenGenerator: tokenGenerator,
		commandSender:  commandSender,
		stateReader:    stateReader,
		geoResolver:    geoResolver,
	}
	if len(grantSvc) > 0 {
		h.grantService = grantSvc[0]
	}
	return h
}

func (h *PodConnectHandler) podResourceWithGrants(ctx context.Context, podKey string, orgID, createdByID int64) policy.ResourceContext {
	rc := policy.PodResource(orgID, createdByID)
	if h.grantService == nil {
		return rc
	}
	if ids, err := h.grantService.GetGrantedUserIDs(ctx, grant.TypePod, podKey); err == nil && len(ids) > 0 {
		return rc.WithGrants(ids)
	}
	return rc
}

type PodConnectResponse struct {
	RelayURL string `json:"relay_url"`
	Token    string `json:"token"`
	PodKey   string `json:"pod_key"`

	// LocalRelayURL is the runner's local WS server URL (e.g. ws://127.0.0.1:38421)
	// when the runner advertises one and the caller can use it. Empty otherwise.
	// Clients SHOULD prefer LocalRelayURL when set, with cloud RelayURL as fallback.
	LocalRelayURL    string `json:"local_relay_url,omitempty"`
	LocalToken       string `json:"local_token,omitempty"`
	LocalRelayNodeID string `json:"local_relay_node_id,omitempty"` // Runner's node_id; renderer skips probe when this != its own host node_id.
}

func (h *PodConnectHandler) GetPodConnection(c *gin.Context) {
	podKey := c.Param("key")

	if h.relayManager == nil || !h.relayManager.HasHealthyRelays() {
		apierr.ServiceUnavailable(c, apierr.SERVICE_UNAVAILABLE, "Relay service is not available")
		return
	}

	pod, err := h.podService.GetPod(c.Request.Context(), podKey)
	if err != nil {
		apierr.ResourceNotFound(c, "Pod not found")
		return
	}

	tenant := middleware.GetTenant(c)
	if tenant == nil {
		apierr.Unauthorized(c, apierr.AUTH_REQUIRED, "Unauthorized")
		return
	}
	if !policy.PodPolicy.AllowRead(
		policy.NewSubject(tenant.OrganizationID, tenant.UserID, tenant.UserRole),
		h.podResourceWithGrants(c.Request.Context(), podKey, pod.OrganizationID, pod.CreatedByID),
	) {
		apierr.ForbiddenAccess(c)
		return
	}

	if !pod.IsActive() {
		apierr.BadRequest(c, apierr.VALIDATION_FAILED, "Pod is not active")
		return
	}

	userID := middleware.GetUserID(c)
	if userID == 0 {
		apierr.Unauthorized(c, apierr.AUTH_REQUIRED, "User not found")
		return
	}

	opts := relay.GeoSelectOptions{OrgSlug: tenant.OrganizationSlug}
	if h.geoResolver != nil {
		if loc := h.geoResolver.Resolve(c.ClientIP()); loc != nil {
			opts.Latitude = loc.Latitude
			opts.Longitude = loc.Longitude
			opts.HasUserLocation = true
		}
	}
	relayInfo := h.relayManager.SelectRelayForPodGeo(opts)
	if relayInfo == nil {
		apierr.ServiceUnavailable(c, apierr.SERVICE_UNAVAILABLE, "No healthy relay available")
		return
	}

	// Always notify runner to connect to relay
	// Runner handles idempotency - if already connected to same relay, it just updates the token
	var localRelayURL, localToken, localRelayNodeID string
	if h.commandSender != nil && pod.RunnerID > 0 {
		runnerToken, err := h.tokenGenerator.GenerateToken(
			podKey,
			pod.RunnerID,
			0, // userID=0 for runner token
			tenant.OrganizationID,
			time.Hour,
		)
		if err != nil {
			apierr.InternalError(c, "Failed to generate runner token")
			return
		}

		if h.stateReader != nil {
			localRelayURL = h.stateReader.GetRunnerLocalRelayURL(pod.RunnerID)
		}
		if localRelayURL != "" {
			localToken, err = h.tokenGenerator.GenerateToken(
				podKey,
				pod.RunnerID,
				userID,
				tenant.OrganizationID,
				time.Hour,
			)
			if err != nil {
				slog.WarnContext(c.Request.Context(), "failed to generate local token, falling back to cloud relay only",
					"pod_key", podKey, "runner_id", pod.RunnerID, "error", err)
				localRelayURL = ""
			} else if h.stateReader != nil {
				localRelayNodeID = h.stateReader.GetRunnerNodeID(pod.RunnerID)
			}
		}

		if err := h.commandSender.SendSubscribePod(
			c.Request.Context(),
			pod.RunnerID,
			podKey,
			relayInfo.URL, // Public URL via reverse proxy — all runners use this single URL
			runnerToken,
			localToken,
			true, // include snapshot
			1000, // snapshot history lines
		); err != nil {
			slog.WarnContext(c.Request.Context(), "Failed to send subscribe pod command to runner",
				"pod_key", podKey,
				"runner_id", pod.RunnerID,
				"error", err)
		}
	}

	runnerID := pod.RunnerID

	token, err := h.tokenGenerator.GenerateToken(
		podKey,
		runnerID,
		userID,
		tenant.OrganizationID,
		time.Hour,
	)
	if err != nil {
		apierr.InternalError(c, "Failed to generate token")
		return
	}

	c.JSON(http.StatusOK, PodConnectResponse{
		RelayURL:         relayInfo.URL,
		Token:            token,
		PodKey:           podKey,
		LocalRelayURL:    localRelayURL,
		LocalToken:       localToken,
		LocalRelayNodeID: localRelayNodeID,
	})
}

func RegisterPodConnectRoutes(
	router *gin.RouterGroup,
	podService PodServiceForHandler,
	relayManager *relay.Manager,
	tokenGenerator *relay.TokenGenerator,
	commandSender runner.RunnerCommandSender,
	stateReader runner.RunnerStateReader,
	geoResolver geo.Resolver,
	grantSvc *grantservice.Service,
) {
	handler := NewPodConnectHandler(podService, relayManager, tokenGenerator, commandSender, stateReader, geoResolver, grantSvc)
	router.GET("/pods/:key/relay/connect", handler.GetPodConnection)
}
