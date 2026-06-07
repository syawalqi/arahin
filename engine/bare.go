package engine

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/syawalqi/arahin/llm"
	"github.com/syawalqi/arahin/tools"
)

type BareEngine struct {
	llm      llm.Provider
	model    string
	geocoder *tools.Geocoder
	poiSrch  *tools.POISearcher
}

func NewBareEngine(provider llm.Provider, model string, gc *tools.Geocoder, ps *tools.POISearcher) *BareEngine {
	return &BareEngine{
		llm:      provider,
		model:    model,
		geocoder: gc,
		poiSrch:  ps,
	}
}

func (e *BareEngine) Name() Mode { return ModeBare }

// placeNameHintKeywords help distinguish vague descriptions from specific place names.
var vagueIndicators = []string{"dekat", "deket", "deketan", "sate", "bakso", "nasi", "mie",
	"masjid", "musholla", "restoran", "rumah", "kafe", "angkringan", "atm", "bank", "mall",
	"supermarket", "rumah sakit", "klinik", "apotek", "pom", "spbu", "enak", "murah", "dekat",
}

func isVague(name string) bool {
	lower := strings.ToLower(name)
	for _, v := range vagueIndicators {
		if strings.Contains(lower, v) {
			return true
		}
	}
	return false
}

func (e *BareEngine) Plan(ctx context.Context, prompt string) ([]Waypoint, error) {
	resp, err := e.llm.Chat(ctx, llm.ChatRequest{
		Model:       e.model,
		Temperature: 0.1,
		MaxTokens:   1024,
		Messages: []llm.Message{
			{Role: llm.RoleSystem, Content: BarePrompt},
			{Role: llm.RoleUser, Content: prompt},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("bare: llm: %w", err)
	}

	// Parse JSON array of strings
	names := tryParseStringArray(resp.Content)
	if names == nil {
		return nil, fmt.Errorf("bare: LLM did not return JSON array (got: %s)", truncate(resp.Content, 200))
	}

	waypoints := make([]Waypoint, 0, len(names))
	for _, name := range names {
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}

		if isVague(name) {
			// Try to find via POI search
			// For bare mode without an anchor, try geocoding the whole phrase
			// then searching from center of Jakarta
			anchor := e.geocoder.Geocode("Jakarta")
			if anchor != nil && anchor.Error == "" {
				pois := e.poiSrch.Search(name, anchor.Lat, anchor.Lng, 5)
				if len(pois) > 0 {
					wp := Waypoint{
						Name:        pois[0].Name,
						Lat:         pois[0].Lat,
						Lng:         pois[0].Lng,
						DisplayName: pois[0].Category,
					}
					waypoints = append(waypoints, wp)
					continue
				}
			}
		}

		result := e.geocoder.Geocode(name)
		if result.Error != "" {
			return nil, fmt.Errorf("bare: geocode %q: %s", name, result.Error)
		}
		waypoints = append(waypoints, Waypoint{
			Name:        result.Name,
			Lat:         result.Lat,
			Lng:         result.Lng,
			DisplayName: result.DisplayName,
		})
	}

	if len(waypoints) < 2 {
		return nil, fmt.Errorf("bare: need at least 2 waypoints, got %d", len(waypoints))
	}

	return waypoints, nil
}

// tryParseStringArray attempts to parse content as a JSON string array.
func tryParseStringArray(content string) []string {
	content = strings.TrimSpace(content)
	// Try direct parse first
	var arr []string
	if err := json.Unmarshal([]byte(content), &arr); err == nil {
		return arr
	}
	// Try finding JSON array within text (for cases where LLM adds explanation)
	start := strings.Index(content, "[")
	end := strings.LastIndex(content, "]")
	if start >= 0 && end > start {
		if err := json.Unmarshal([]byte(content[start:end+1]), &arr); err == nil {
			return arr
		}
	}
	return nil
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}


