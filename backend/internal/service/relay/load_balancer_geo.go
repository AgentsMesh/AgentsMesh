package relay

import (
	"github.com/anthropics/agentsmesh/backend/internal/service/geo"
)

type GeoSelectOptions struct {
	OrgSlug         string
	Latitude        float64
	Longitude       float64
	HasUserLocation bool
}

const (
	maxNearbyGap = 2000

	minNearbyThresholdKm = 500

	earthCircumferenceKm = 40075
)

func (m *Manager) SelectRelayForPodGeo(opts GeoSelectOptions) *RelayInfo {
	if !opts.HasUserLocation {
		return m.SelectRelayWithAffinity(opts.OrgSlug)
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	if len(m.relays) == 0 {
		return nil
	}

	strictCandidates, lenientCandidates := m.collectCandidatesByTierLocked(opts)

	candidates := strictCandidates
	if len(candidates) == 0 {
		candidates = lenientCandidates
	}

	if len(candidates) == 0 {
		m.logger.Warn("No relay candidates for geo selection",
			"org_slug", opts.OrgSlug,
			"user_lat", opts.Latitude,
			"user_lng", opts.Longitude,
			"total_relays", len(m.relays))
		return nil
	}

	minDist := candidates[0].distance
	for _, c := range candidates[1:] {
		if c.distance < minDist {
			minDist = c.distance
		}
	}

	threshold := minDist * 1.5
	if threshold < minNearbyThresholdKm {
		threshold = minNearbyThresholdKm
	}
	if cap := minDist + maxNearbyGap; threshold > cap {
		threshold = cap
	}

	nearbyIDs := make([]string, 0, len(candidates))
	for _, c := range candidates {
		if c.distance <= threshold {
			nearbyIDs = append(nearbyIDs, c.id)
		}
	}

	selected := m.selectFromCandidatesLocked(opts.OrgSlug, nearbyIDs)
	if selected != nil {
		distKm := float64(-1)
		if selected.HasGeoCoords() {
			distKm = geo.HaversineDistance(
				opts.Latitude, opts.Longitude,
				selected.Latitude, selected.Longitude)
		}
		m.logger.Debug("Selected relay with geo affinity",
			"relay_id", selected.ID,
			"org_slug", opts.OrgSlug,
			"user_lat", opts.Latitude,
			"user_lng", opts.Longitude,
			"relay_lat", selected.Latitude,
			"relay_lng", selected.Longitude,
			"relay_distance_km", distKm,
			"nearby_count", len(nearbyIDs),
			"total_available", len(candidates))
		return selected
	}

	return nil
}

type relayDist struct {
	id       string
	distance float64
}

// collectCandidatesByTierLocked walks m.relays once collecting strict (within CPU/mem thresholds)
// and lenient (reachable, not at hard cap) tiers with their Haversine distances. Caller holds m.mu.RLock.
func (m *Manager) collectCandidatesByTierLocked(opts GeoSelectOptions) (strict, lenient []relayDist) {
	strict = make([]relayDist, 0, len(m.relays))
	lenient = make([]relayDist, 0, len(m.relays))

	for id, r := range m.relays {
		dist := float64(earthCircumferenceKm)
		if r.HasGeoCoords() {
			dist = geo.HaversineDistance(opts.Latitude, opts.Longitude, r.Latitude, r.Longitude)
		}
		rd := relayDist{id: id, distance: dist}

		if isRelayAvailable(r) {
			strict = append(strict, rd)
		} else if isRelayReachable(r) {
			lenient = append(lenient, rd)
		}
	}
	return strict, lenient
}
