package geo

import (
	"math"
	"testing"
)

func TestHaversineDistance(t *testing.T) {
	tests := []struct {
		name           string
		lat1, lng1     float64
		lat2, lng2     float64
		expectedKm     float64
		toleranceKm    float64
	}{
		{
			name:        "same point",
			lat1:        40.7128, lng1: -74.0060,
			lat2:        40.7128, lng2: -74.0060,
			expectedKm:  0,
			toleranceKm: 0.01,
		},
		{
			name:        "New York to London",
			lat1:        40.7128, lng1: -74.0060,
			lat2:        51.5074, lng2: -0.1278,
			expectedKm:  5570,
			toleranceKm: 20,
		},
		{
			name:        "Tokyo to Sydney",
			lat1:        35.6762, lng1: 139.6503,
			lat2:        -33.8688, lng2: 151.2093,
			expectedKm:  7823,
			toleranceKm: 20,
		},
		{
			name:        "North Pole to South Pole",
			lat1:        90, lng1: 0,
			lat2:        -90, lng2: 0,
			expectedKm:  20015,
			toleranceKm: 20,
		},
		{
			name:        "Shanghai to Beijing",
			lat1:        31.2304, lng1: 121.4737,
			lat2:        39.9042, lng2: 116.4074,
			expectedKm:  1068,
			toleranceKm: 15,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := HaversineDistance(tt.lat1, tt.lng1, tt.lat2, tt.lng2)
			if math.Abs(got-tt.expectedKm) > tt.toleranceKm {
				t.Errorf("HaversineDistance(%v,%v → %v,%v) = %.1f km, want ~%.0f km (±%.0f)",
					tt.lat1, tt.lng1, tt.lat2, tt.lng2, got, tt.expectedKm, tt.toleranceKm)
			}
		})
	}
}

func TestHaversineDistanceSymmetric(t *testing.T) {
	d1 := HaversineDistance(40.7128, -74.0060, 51.5074, -0.1278)
	d2 := HaversineDistance(51.5074, -0.1278, 40.7128, -74.0060)
	if math.Abs(d1-d2) > 0.001 {
		t.Errorf("distance should be symmetric: %v != %v", d1, d2)
	}
}

func TestHaversineDistanceAntipodal(t *testing.T) {
	// Antipodal points should be ~20015 km (half Earth circumference)
	d := HaversineDistance(0, 0, 0, 180)
	if math.Abs(d-20015) > 20 {
		t.Errorf("antipodal distance: got %.1f km, want ~20015 km", d)
	}
}

func TestHaversineDistanceDatelineCrossing(t *testing.T) {
	// Points across the international date line should compute correctly.
	// Fiji (179°E) to Samoa (-171°W) — ~1200km, not ~39000km
	d := HaversineDistance(-17.7134, 178.065, -13.8333, -171.75)
	if d > 2000 {
		t.Errorf("dateline crossing distance too large: got %.1f km, want < 2000 km", d)
	}
}
