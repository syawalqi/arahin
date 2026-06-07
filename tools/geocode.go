package tools

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

type GeocodeResult struct {
	Name        string  `json:"name"`
	Lat         float64 `json:"lat"`
	Lng         float64 `json:"lng"`
	DisplayName string  `json:"display_name,omitempty"`
	Error       string  `json:"error,omitempty"`
}

type geocodeCandidate struct {
	Name        string `json:"name"`
	Lat         string `json:"lat"`
	Lon         string `json:"lon"`
	DisplayName string `json:"display_name"`
}

type Geocoder struct {
	client  *http.Client
	cache   map[string]*GeocodeResult
	mu      sync.Mutex
	lastReq time.Time
}

func NewGeocoder() *Geocoder {
	return &Geocoder{
		client: &http.Client{Timeout: 15 * time.Second},
		cache:  make(map[string]*GeocodeResult),
	}
}

func (g *Geocoder) Geocode(placeName string) *GeocodeResult {
	key := strings.ToLower(strings.TrimSpace(placeName))
	if key == "" {
		return &GeocodeResult{Error: "empty place name"}
	}

	g.mu.Lock()
	if cached, ok := g.cache[key]; ok {
		g.mu.Unlock()
		return cached
	}
	g.mu.Unlock()

	// Try the full name first
	result := g.query(placeName)
	if result == nil {
		// Retry with city/region suffix stripped
		short := strings.SplitN(placeName, ",", 2)[0]
		if trimmed := strings.TrimSpace(short); trimmed != placeName {
			result = g.query(trimmed)
		}
	}

	if result == nil {
		result = &GeocodeResult{Error: fmt.Sprintf("geocode failed: %s", placeName)}
	}

	g.mu.Lock()
	g.cache[key] = result
	g.mu.Unlock()
	return result
}

func (g *Geocoder) query(placeName string) *GeocodeResult {
	// Rate limit: 1 req/s
	g.mu.Lock()
	elapsed := time.Since(g.lastReq)
	if elapsed < time.Second {
		time.Sleep(time.Second - elapsed)
	}
	g.lastReq = time.Now()
	g.mu.Unlock()

	params := url.Values{}
	params.Set("q", placeName)
	params.Set("format", "json")
	params.Set("limit", "5")

	req, _ := http.NewRequest("GET", "https://nominatim.openstreetmap.org/search?"+params.Encode(), nil)
	req.Header.Set("User-Agent", "ARAHIN/1.0")
	req.Header.Set("Accept-Language", "id,en")

	resp, err := g.client.Do(req)
	if err != nil {
		return nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil
	}

	var candidates []geocodeCandidate
	if err := json.NewDecoder(resp.Body).Decode(&candidates); err != nil {
		return nil
	}
	if len(candidates) == 0 {
		return nil
	}

	best := pickBest(candidates, placeName)
	lat, _ := strToFloat(best.Lat)
	lng, _ := strToFloat(best.Lon)

	return &GeocodeResult{
		Name:        best.Name,
		Lat:         lat,
		Lng:         lng,
		DisplayName: best.DisplayName,
	}
}

func pickBest(candidates []geocodeCandidate, query string) geocodeCandidate {
	queryLower := strings.ToLower(query)
	queryWords := make([]string, 0)
	for _, w := range strings.Fields(queryLower) {
		if len(w) > 2 {
			queryWords = append(queryWords, w)
		}
	}

	indonesianCities := map[string]bool{
		"yogyakarta": true, "jakarta": true, "bandung": true, "surabaya": true,
		"semarang": true, "magelang": true, "solo": true, "medan": true,
		"bogor": true, "depok": true, "tangerang": true, "bekasi": true,
		"malang": true, "surakarta": true, "cirebon": true, "pekalongan": true,
	}

	best := candidates[0]
	bestScore := -1

	for _, c := range candidates {
		score := 0
		name := strings.ToLower(c.Name)
		display := strings.ToLower(c.DisplayName)

		for _, qw := range queryWords {
			if strings.Contains(name, qw) {
				score += 2
			} else if strings.Contains(display, qw) {
				score += 1
			}
			if strings.Contains(display, qw) && indonesianCities[qw] {
				score += 4
			}
		}

		if score > bestScore {
			best = c
			bestScore = score
		}
	}

	return best
}

func strToFloat(s string) (float64, error) {
	var f float64
	err := json.Unmarshal([]byte(s), &f)
	return f, err
}
