package relay

import (
	"github.com/anthropics/agentsmesh/backend/internal/service/geo"
)

// GeoSelectOptions holds options for geo-aware relay selection.
type GeoSelectOptions struct {
	OrgSlug         string
	Latitude        float64
	Longitude       float64
	HasUserLocation bool // true if user's geo coordinates are available
}

// Geo-aware relay selection constants.
const (
	// maxNearbyGap is the maximum distance gap (km) allowed between the nearest relay
	// and other relays to consider them "nearby". Prevents relays on opposite sides
	// of the globe from being grouped together when all relays are far away.
	maxNearbyGap = 2000

	// minNearbyThresholdKm is the minimum nearby threshold (km).
	// Ensures relays in the same region (~500km) are always grouped together.
	minNearbyThresholdKm = 500

	// earthCircumferenceKm is the approximate circumference of the Earth in km.
	// Used as a sentinel distance for relays without geo data (treated as farthest).
	earthCircumferenceKm = 40075
)

// SelectRelayForPodGeo selects a relay using geo-proximity + org-affinity.
//
// Algorithm:
//  1. If HasUserLocation is false, fall back to pure org-affinity.
//  2. Compute distance from user to each available relay.
//  3. Group relays within "nearby" threshold.
//  4. Within the nearby group, apply org-affinity for stable selection.
//
// Nearby threshold = max(minDist * 1.5, minNearbyThresholdKm), capped at minDist + maxNearbyGap.
// This ensures:
//   - Close relays cluster naturally (minNearbyThresholdKm tolerance for same-region)
//   - Far relays don't get mixed with each other (maxNearbyGap cap)
//
// When all relays fail strict availability checks (CPU/mem > strictCPUThreshold/strictMemThreshold),
// falls back to lenient checks (healthy + not at hard capacity) to avoid returning nil.
func (m *Manager) SelectRelayForPodGeo(opts GeoSelectOptions) *RelayInfo {
	if !opts.HasUserLocation {
		return m.SelectRelayWithAffinity(opts.OrgSlug)
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	if len(m.relays) == 0 {
		return nil
	}

	// Single-pass: collect strict and lenient candidates with distances simultaneously
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

	// Find minimum distance
	minDist := candidates[0].distance
	for _, c := range candidates[1:] {
		if c.distance < minDist {
			minDist = c.distance
		}
	}

	// Nearby threshold: min distance * 1.5, at least minNearbyThresholdKm, capped at minDist + maxNearbyGap
	threshold := minDist * 1.5
	if threshold < minNearbyThresholdKm {
		threshold = minNearbyThresholdKm
	}
	if cap := minDist + maxNearbyGap; threshold > cap {
		threshold = cap
	}

	// Filter to nearby relays
	nearbyIDs := make([]string, 0, len(candidates))
	for _, c := range candidates {
		if c.distance <= threshold {
			nearbyIDs = append(nearbyIDs, c.id)
		}
	}

	// Apply org-affinity within the nearby group
	selected := m.selectFromCandidatesLocked(opts.OrgSlug, nearbyIDs)
	if selected != nil {
		distKm := float64(-1) // -1 indicates relay has no geo data
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

// relayDist pairs a relay ID with its distance from the user.
type relayDist struct {
	id       string
	distance float64 // km from user
}

// collectCandidatesByTierLocked performs a single pass over relays, collecting
// strict (CPU/mem within thresholds) and lenient (healthy + not at hard capacity) candidates
// with their Haversine distances. Caller must hold m.mu.RLock.
func (m *Manager) collectCandidatesByTierLocked(opts GeoSelectOptions) (strict, lenient []relayDist) {
	strict = make([]relayDist, 0, len(m.relays))
	lenient = make([]relayDist, 0, len(m.relays))

	for id, r := range m.relays {
		// Relays without geo data get max distance (treated as farthest)
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
