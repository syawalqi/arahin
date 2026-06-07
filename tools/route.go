package tools

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type RouteSegment struct {
	DistanceKm  float64     `json:"distance_km"`
	DurationMin float64     `json:"duration_min"`
	Geometry    interface{} `json:"geometry"`
	Steps       []Step      `json:"steps"`
}

type Step struct {
	Instruction string `json:"instruction"`
	DistanceM   float64 `json:"distance_m"`
}

type DistanceMatrix struct {
	Durations [][]float64 `json:"durations"`
	Distances [][]float64 `json:"distances"`
}

type routeClient struct {
	client *http.Client
}

func newRouteClient() *routeClient {
	return &routeClient{
		client: &http.Client{Timeout: 15 * time.Second},
	}
}

var defaultRouteClient = newRouteClient()

// GetRoute returns driving route and geometry between two waypoints.
func GetRoute(origin, destination GeocodeResult) (*RouteSegment, error) {
	if origin.Error != "" {
		return nil, fmt.Errorf("origin: %s", origin.Error)
	}
	if destination.Error != "" {
		return nil, fmt.Errorf("destination: %s", destination.Error)
	}

	src := fmt.Sprintf("%f,%f", origin.Lng, origin.Lat)
	dst := fmt.Sprintf("%f,%f", destination.Lng, destination.Lat)

	url := fmt.Sprintf("https://router.project-osrm.org/route/v1/driving/%s;%s?overview=full&geometries=geojson&steps=true&alternatives=false",
		src, dst)

	resp, err := defaultRouteClient.client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("osrm request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("osrm status %d", resp.StatusCode)
	}

	var data struct {
		Code   string `json:"code"`
		Routes []struct {
			Distance float64     `json:"distance"`
			Duration float64     `json:"duration"`
			Geometry interface{} `json:"geometry"`
			Legs     []struct {
				Steps []struct {
					Name     string  `json:"name"`
					Distance float64 `json:"distance"`
				} `json:"steps"`
			} `json:"legs"`
		} `json:"routes"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, fmt.Errorf("osrm decode: %w", err)
	}
	if data.Code != "Ok" || len(data.Routes) == 0 {
		return nil, fmt.Errorf("osrm: %s", data.Code)
	}

	route := data.Routes[0]
	steps := make([]Step, 0)
	for _, leg := range route.Legs {
		for _, s := range leg.Steps {
			if s.Name != "" {
				steps = append(steps, Step{Instruction: s.Name, DistanceM: s.Distance})
			}
		}
	}

	return &RouteSegment{
		DistanceKm:  route.Distance / 1000,
		DurationMin: route.Duration / 60,
		Geometry:    route.Geometry,
		Steps:       steps,
	}, nil
}

// GetDistanceMatrix returns the distance/duration matrix for all waypoint pairs.
func GetDistanceMatrix(waypoints []GeocodeResult) (*DistanceMatrix, error) {
	if len(waypoints) < 2 {
		return nil, fmt.Errorf("need at least 2 waypoints, got %d", len(waypoints))
	}

	// Build coordinate string: lng,lat;lng,lat;...
	parts := make([]string, len(waypoints))
	for i, wp := range waypoints {
		if wp.Error != "" {
			return nil, fmt.Errorf("waypoint %d: %s", i, wp.Error)
		}
		parts[i] = fmt.Sprintf("%f,%f", wp.Lng, wp.Lat)
	}

	coords := ""
	for i, p := range parts {
		if i > 0 {
			coords += ";"
		}
		coords += p
	}

	url := fmt.Sprintf("https://router.project-osrm.org/table/v1/driving/%s?annotations=distance,duration", coords)

	// Longer timeout for multiple waypoints
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("osrm table request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("osrm table status %d", resp.StatusCode)
	}

	var data struct {
		Code       string          `json:"code"`
		Durations  [][]float64     `json:"durations"`
		Distances  [][]float64     `json:"distances"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, fmt.Errorf("osrm table decode: %w", err)
	}
	if data.Code != "Ok" {
		return nil, fmt.Errorf("osrm table: %s", data.Code)
	}

	return &DistanceMatrix{
		Durations: data.Durations,
		Distances: data.Distances,
	}, nil
}
