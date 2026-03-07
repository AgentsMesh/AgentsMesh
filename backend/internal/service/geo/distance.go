package geo

import "math"

const earthRadiusKm = 6371.0

// HaversineDistance calculates the great-circle distance in km between two
// points specified by latitude and longitude (in degrees).
func HaversineDistance(lat1, lng1, lat2, lng2 float64) float64 {
	dLat := degreesToRadians(lat2 - lat1)
	dLng := degreesToRadians(lng2 - lng1)

	lat1Rad := degreesToRadians(lat1)
	lat2Rad := degreesToRadians(lat2)

	a := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Cos(lat1Rad)*math.Cos(lat2Rad)*math.Sin(dLng/2)*math.Sin(dLng/2)
	// Clamp to [0, 1]: floating-point rounding near antipodal points can push a
	// slightly above 1.0, making sqrt(1-a) return NaN.
	if a > 1 {
		a = 1
	}
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))

	return earthRadiusKm * c
}

// degreesToRadians converts degrees to radians.
func degreesToRadians(deg float64) float64 {
	return deg * math.Pi / 180
}
