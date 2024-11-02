package maps

import (
	"alertgo/types"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

const (
	cMapsBaseURL = "https://maps.googleapis.com/maps/api/staticmap"
	cGeocodeURL  = "https://maps.googleapis.com/maps/api/geocode/json"
)

func GenerateMapURL(locations []string, apiKey string) string {
	if len(locations) == 0 {
		return ""
	}

	markers := make([]string, 0, len(locations))
	var minLat, maxLat, minLng, maxLng float64
	first := true

	// First pass: collect coordinates and calculate bounds
	for _, loc := range locations {
		coords, err := geocodeLocation(loc, apiKey)
		if err != nil {
			log.Printf("Warning: Failed to geocode location %s: %v", loc, err)
			continue
		}

		parts := strings.Split(coords, ",")
		if len(parts) != 2 {
			continue
		}

		lat, _ := strconv.ParseFloat(parts[0], 64)
		lng, _ := strconv.ParseFloat(parts[1], 64)

		if first {
			minLat, maxLat = lat, lat
			minLng, maxLng = lng, lng
			first = false
		} else {
			minLat = math.Min(minLat, lat)
			maxLat = math.Max(maxLat, lat)
			minLng = math.Min(minLng, lng)
			maxLng = math.Max(maxLng, lng)
		}

		markers = append(markers, "icon:https://maps.google.com/mapfiles/kml/shapes/placemark_square_highlight.png|scale:1|"+coords)
	}

	if len(markers) == 0 {
		return ""
	}

	// Calculate center
	centerLat := (minLat + maxLat) / 2
	centerLng := (minLng + maxLng) / 2
	center := fmt.Sprintf("%.6f,%.6f", centerLat, centerLng)

	// Calculate appropriate zoom level based on bounds
	latSpread := maxLat - minLat
	lngSpread := maxLng - minLng
	spread := math.Max(latSpread, lngSpread)

	// Adjust zoom based on spread (values tuned for Israel's geography)
	zoom := "13" // default zoom
	if spread > 0.5 {
		zoom = "9"
	} else if spread > 0.2 {
		zoom = "10"
	} else if spread > 0.1 {
		zoom = "11"
	} else if spread > 0.05 {
		zoom = "12"
	}

	params := url.Values{
		"center":   {center},
		"zoom":     {zoom},
		"size":     {"800x600"}, // Larger size for better visibility
		"markers":  markers,
		"key":      {apiKey},
		"language": {"he"},
		"format":   {"png"},
		"scale":    {"2"}, // Retina display support
	}

	finalURL := cMapsBaseURL + "?" + params.Encode()
	return finalURL
}

func geocodeLocation(location, apiKey string) (string, error) {
	params := url.Values{
		"address":  {location + ", Israel"}, // Add country for better results
		"key":      {apiKey},
		"language": {"he"},
	}

	resp, err := http.Get(cGeocodeURL + "?" + params.Encode())
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var result types.GeocodingResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}

	if len(result.Results) == 0 {
		return "", fmt.Errorf("no results found for location: %s", location)
	}

	coords := fmt.Sprintf("%.6f,%.6f",
		result.Results[0].Geometry.Location.Lat,
		result.Results[0].Geometry.Location.Lng)

	return coords, nil
}
