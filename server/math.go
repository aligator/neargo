package server

import "math"

// HSin haversin(θ) function
func HSin(theta float64) float64 {
	return math.Pow(math.Sin(theta/2), 2)
}

// Distance calculated in km between the given lat/lon coordinates.
func Distance(aLat, aLon, bLat, bLon float64) float64 {
	// convert to radians
	// must cast radius as float to multiply later
	var aRLat, aRLon, bRLat, bRLon, r float64
	aRLat = aLat * math.Pi / 180
	aRLon = aLon * math.Pi / 180
	bRLat = bLat * math.Pi / 180
	bRLon = bLon * math.Pi / 180
	r = 6378100 // Earth radius in METERS

	// calculate
	h := HSin(bRLat-aRLat) + math.Cos(aRLat)*math.Cos(bRLat)*HSin(bRLon-aRLon)

	return 2 * r * math.Asin(math.Sqrt(h)) / 1000
}
