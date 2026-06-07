package engine

import (
	"context"
	"fmt"
	"strings"

	"github.com/syawalqi/arahin/llm"
	"github.com/syawalqi/arahin/tools"
)

type PipelineEngine struct {
	llm      llm.Provider
	model    string
	geocoder *tools.Geocoder
	poiSrch  *tools.POISearcher
}

func NewPipelineEngine(provider llm.Provider, model string, gc *tools.Geocoder, ps *tools.POISearcher) *PipelineEngine {
	return &PipelineEngine{
		llm:      provider,
		model:    model,
		geocoder: gc,
		poiSrch:  ps,
	}
}

func (e *PipelineEngine) Name() Mode { return ModePipeline }

type pipelineTask struct {
	Type     string `json:"type"`
	Place    string `json:"place,omitempty"`
	Query    string `json:"query,omitempty"`
	Category string `json:"category,omitempty"`
	Anchor   string `json:"anchor,omitempty"`
}

func (e *PipelineEngine) Plan(ctx context.Context, prompt string) ([]Waypoint, error) {
	// Extract waypoints using Go parsing (same as bare mode).
	names := extractWaypointNames(prompt)
	if len(names) < 2 {
		llmNames, err := e.llmFallback(ctx, prompt)
		if err != nil {
			return nil, fmt.Errorf("pipeline: could not extract waypoints: %w", err)
		}
		names = llmNames
	}
	return e.resolveWaypoints(ctx, names)
}

func (e *PipelineEngine) llmFallback(ctx context.Context, prompt string) ([]string, error) {
	resp, err := e.llm.ChatCollect(ctx, llm.ChatRequest{
		Model:       e.model,
		Temperature: 0.1,
		MaxTokens:   1536,
		Messages: []llm.Message{
			{Role: llm.RoleSystem, Content: "Return a JSON array of 2-5 waypoint strings. Each includes city. No thinking."},
			{Role: llm.RoleUser, Content: prompt},
		},
	})
	if err != nil {
		return nil, err
	}
	names := tryParseStringArray(resp.Content)
	if names == nil {
		return nil, fmt.Errorf("LLM did not return valid JSON")
	}
	return names, nil
}

// resolveWaypoints takes parsed waypoint strings and resolves them to coordinates.
// It classifies each waypoint: named place → geocode, vague/query → poi_search, category → poi_search.
func (e *PipelineEngine) resolveWaypoints(ctx context.Context, names []string) ([]Waypoint, error) {
	waypoints := make([]Waypoint, 0, len(names))
	var lastAnchor *Waypoint

	for _, name := range names {
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}

		queryLower := strings.ToLower(name)

		// Check if it's a known category (masjid, restoran, etc)
		if _, isCategory := categoryMap[queryLower]; isCategory {
			if lastAnchor != nil {
				pois := e.poiSrch.Search(queryLower, lastAnchor.Lat, lastAnchor.Lng, 3)
				if len(pois) > 0 {
					wp := Waypoint{Name: pois[0].Name, Lat: pois[0].Lat, Lng: pois[0].Lng, DisplayName: pois[0].Category}
					waypoints = append(waypoints, wp)
					lastAnchor = &wp
					continue
				}
			}
			// fallback: use as named place
		}

		// Check if it's a vague/query term
		if isVague(name) {
			if lastAnchor != nil {
				pois := e.poiSrch.Search(name, lastAnchor.Lat, lastAnchor.Lng, 3)
				if len(pois) > 0 {
					wp := Waypoint{Name: pois[0].Name, Lat: pois[0].Lat, Lng: pois[0].Lng, DisplayName: pois[0].Category}
					waypoints = append(waypoints, wp)
					lastAnchor = &wp
					continue
				}
			}
			// fallback: geocode with Jakarta as context
			name = name + ", Jakarta"
		}

		// Named place: geocode it
		result := e.geocoder.Geocode(name)
		if result.Error != "" {
			return nil, fmt.Errorf("pipeline: geocode %q: %s", name, result.Error)
		}
		wp := Waypoint{Name: result.Name, Lat: result.Lat, Lng: result.Lng, DisplayName: result.DisplayName}
		waypoints = append(waypoints, wp)
		lastAnchor = &wp
	}

	if len(waypoints) < 2 {
		return nil, fmt.Errorf("pipeline: need at least 2 waypoints, got %d", len(waypoints))
	}

	return waypoints, nil
}

// categoryMap maps Indonesian category names to OSM amenity tags.
var categoryMap = map[string]string{
	"masjid": "mosque", "musholla": "mosque", "musala": "mosque",
	"restoran": "restaurant", "rumah makan": "restaurant", "makan": "restaurant",
	"kafe": "cafe", "angkringan": "cafe",
	"atm": "atm", "bank": "bank",
	"rumah sakit": "hospital", "klinik": "clinic", "apotek": "pharmacy",
	"mall": "mall", "supermarket": "supermarket", "pasar": "market",
	"pom bensin": "fuel", "spbu": "fuel",
}
