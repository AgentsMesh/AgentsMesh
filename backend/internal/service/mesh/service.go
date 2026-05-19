package mesh

import (
	"context"
	"errors"
	"log/slog"

	"github.com/anthropics/agentsmesh/backend/internal/domain/agentpod"
	"github.com/anthropics/agentsmesh/backend/internal/domain/channel"
	"github.com/anthropics/agentsmesh/backend/internal/domain/mesh"
	bindingService "github.com/anthropics/agentsmesh/backend/internal/service/binding"
	channelService "github.com/anthropics/agentsmesh/backend/internal/service/channel"
	podService "github.com/anthropics/agentsmesh/backend/internal/service/agentpod"
)

var (
	ErrTicketNotFound = errors.New("ticket not found")
	ErrRunnerNotFound = errors.New("runner not found")
)

type Service struct {
	repo           mesh.MeshRepository
	podService     *podService.PodService
	channelService *channelService.Service
	bindingService *bindingService.Service
}

func NewService(
	repo mesh.MeshRepository,
	ps *podService.PodService,
	cs *channelService.Service,
	bs *bindingService.Service,
) *Service {
	return &Service{
		repo:           repo,
		podService:     ps,
		channelService: cs,
		bindingService: bs,
	}
}

func (s *Service) GetTopology(ctx context.Context, orgID, userID int64) (*mesh.MeshTopology, error) {
	pods, _, err := s.podService.ListPods(ctx, orgID, agentpod.PodListQuery{Limit: 100})
	if err != nil {
		return nil, err
	}

	nodes := make([]mesh.MeshNode, 0)
	podKeys := make([]string, 0)

	for _, pod := range pods {
		if pod.IsActive() {
			node := s.podToNode(pod)
			nodes = append(nodes, node)
			podKeys = append(podKeys, pod.PodKey)
		}
	}

	edges := make([]mesh.MeshEdge, 0)
	seenBindings := make(map[int64]bool)
	for _, key := range podKeys {
		activeStatus := channel.BindingStatusActive
		bindings, err := s.bindingService.GetBindingsForPod(ctx, key, &activeStatus)
		if err != nil {
			continue
		}
		for _, b := range bindings {
			if seenBindings[b.ID] {
				continue
			}
			seenBindings[b.ID] = true

			if b.IsActive() {
				edges = append(edges, mesh.MeshEdge{
					ID:            b.ID,
					Source:        b.InitiatorPod,
					Target:        b.TargetPod,
					GrantedScopes: []string(b.GrantedScopes),
					PendingScopes: []string(b.PendingScopes),
					Status:        b.Status,
				})
			}
		}
	}

	channels, _, err := s.channelService.ListChannels(ctx, orgID, userID, &channel.ChannelListFilter{
		IncludeArchived: false,
		Limit:           50,
		Offset:          0,
	})
	if err != nil {
		return nil, err
	}

	channelInfos := make([]mesh.ChannelInfo, 0, len(channels))
	for _, ch := range channels {
		channelPods := s.getChannelPods(ctx, ch.ID)

		messageCount := s.getChannelMessageCount(ctx, ch.ID)

		channelInfos = append(channelInfos, mesh.ChannelInfo{
			ID:           ch.ID,
			Name:         ch.Name,
			Description:  ch.Description,
			PodKeys:      channelPods,
			MessageCount: messageCount,
			IsArchived:   ch.IsArchived,
		})
	}

	runners, err := s.repo.ListEnabledRunners(ctx, orgID)
	if err != nil {
		return nil, err
	}

	runnerInfos := make([]mesh.RunnerInfo, 0, len(runners))
	for _, r := range runners {
		runnerInfos = append(runnerInfos, mesh.RunnerInfo{
			ID:                r.ID,
			NodeID:            r.NodeID,
			Status:            r.Status,
			MaxConcurrentPods: r.MaxConcurrentPods,
			CurrentPods:       r.CurrentPods,
		})
	}

	return &mesh.MeshTopology{
		Nodes:    nodes,
		Edges:    edges,
		Channels: channelInfos,
		Runners:  runnerInfos,
	}, nil
}

func (s *Service) podToNode(pod *agentpod.Pod) mesh.MeshNode {
	node := mesh.MeshNode{
		PodKey:       pod.PodKey,
		Status:       pod.Status,
		AgentStatus:  pod.AgentStatus,
		Model:        pod.Model,
		Title:        pod.Title,
		Alias:        pod.Alias,
		TicketID:     pod.TicketID,
		RepositoryID: pod.RepositoryID,
		CreatedByID:  pod.CreatedByID,
		RunnerID:     pod.RunnerID,
		StartedAt:    pod.StartedAt,
	}

	if pod.Runner != nil {
		node.RunnerNodeID = pod.Runner.NodeID
		node.RunnerStatus = pod.Runner.Status
	}

	if pod.Ticket != nil {
		node.TicketSlug = &pod.Ticket.Slug
		node.TicketTitle = &pod.Ticket.Title
	}

	return node
}

func (s *Service) getChannelPods(ctx context.Context, channelID int64) []string {
	keys, err := s.repo.GetChannelPodKeys(ctx, channelID)
	if err != nil {
		return nil
	}
	return keys
}

func (s *Service) getChannelMessageCount(ctx context.Context, channelID int64) int {
	count, err := s.repo.CountChannelMessages(ctx, channelID)
	if err != nil {
		return 0
	}
	return int(count)
}

// CreatePodForTicket is the legacy ticket-pod entry — predates AgentFile SSOT,
// Claude-only by historical convention. New code uses PodOrchestrator.
func (s *Service) CreatePodForTicket(ctx context.Context, req *mesh.CreatePodForTicketRequest) (*agentpod.Pod, error) {
	model := req.Model
	if model == "" {
		model = mesh.LegacyTicketPodModel
	}
	permissionMode := req.PermissionMode
	if permissionMode == "" {
		permissionMode = mesh.LegacyTicketPodPermissionMode
	}
	return s.podService.CreatePodForTicket(ctx, &podService.CreatePodRequest{
		OrganizationID: req.OrganizationID,
		RunnerID:       req.RunnerID,
		AgentSlug:      mesh.LegacyTicketPodAgentSlug,
		TicketID:       &req.TicketID,
		CreatedByID:    req.CreatedByID,
		Prompt:         req.Prompt,
		Model:          model,
		PermissionMode: permissionMode,
	})
}

func (s *Service) GetPodsForTicket(ctx context.Context, ticketID int64) ([]mesh.MeshNode, error) {
	pods, err := s.podService.GetPodsByTicket(ctx, ticketID)
	if err != nil {
		return nil, err
	}

	nodes := make([]mesh.MeshNode, len(pods))
	for i, pod := range pods {
		nodes[i] = s.podToNode(pod)
	}
	return nodes, nil
}

func (s *Service) GetActivePodsForTicket(ctx context.Context, ticketID int64) ([]mesh.MeshNode, error) {
	pods, err := s.podService.GetPodsByTicket(ctx, ticketID)
	if err != nil {
		return nil, err
	}

	nodes := make([]mesh.MeshNode, 0)
	for _, pod := range pods {
		if pod.IsActive() {
			nodes = append(nodes, s.podToNode(pod))
		}
	}
	return nodes, nil
}

func (s *Service) BatchGetTicketPods(ctx context.Context, ticketIDs []int64) (*mesh.BatchTicketPodsResponse, error) {
	pods, err := s.repo.ListPodsByTicketIDs(ctx, ticketIDs)
	if err != nil {
		return nil, err
	}

	result := make(map[int64][]mesh.MeshNode)
	for _, pod := range pods {
		if pod.TicketID != nil {
			ticketID := *pod.TicketID
			if _, exists := result[ticketID]; !exists {
				result[ticketID] = make([]mesh.MeshNode, 0)
			}
			result[ticketID] = append(result[ticketID], s.podToNode(pod))
		}
	}

	for _, id := range ticketIDs {
		if _, exists := result[id]; !exists {
			result[id] = make([]mesh.MeshNode, 0)
		}
	}

	return &mesh.BatchTicketPodsResponse{
		TicketPods: result,
	}, nil
}

func (s *Service) JoinChannel(ctx context.Context, channelID int64, podKey string) error {
	cp := &mesh.ChannelPod{
		ChannelID: channelID,
		PodKey:    podKey,
	}
	if err := s.repo.CreateChannelPod(ctx, cp); err != nil {
		slog.ErrorContext(ctx, "failed to join channel", "channel_id", channelID, "pod_key", podKey, "error", err)
		return err
	}
	slog.InfoContext(ctx, "pod joined channel", "channel_id", channelID, "pod_key", podKey)
	return nil
}

func (s *Service) LeaveChannel(ctx context.Context, channelID int64, podKey string) error {
	if err := s.repo.DeleteChannelPod(ctx, channelID, podKey); err != nil {
		slog.ErrorContext(ctx, "failed to leave channel", "channel_id", channelID, "pod_key", podKey, "error", err)
		return err
	}
	slog.InfoContext(ctx, "pod left channel", "channel_id", channelID, "pod_key", podKey)
	return nil
}

func (s *Service) RecordChannelAccess(ctx context.Context, channelID int64, podKey *string, userID *int64) error {
	access := &mesh.ChannelAccess{
		ChannelID: channelID,
		PodKey:    podKey,
		UserID:    userID,
	}
	return s.repo.CreateChannelAccess(ctx, access)
}
