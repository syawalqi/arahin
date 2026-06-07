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

var knownJakarta = map[string]bool{
	"monas": true, "monumen nasional": true, "taman mini": true, "tmii": true,
	"bundaran hi": true, "bundaran hotel indonesia": true, "dufan": true, "ancol": true,
	"ragunan": true, "kota tua": true, "glodok": true, "mangga dua": true, "tanah abang": true,
	"senayan": true, "gbk": true, "gelora bung karno": true, "istora": true,
	"istiqlal": true, "katedral jakarta": true, "sarinah": true, "thamrin": true,
	"sudirman": true, "menteng": true, "kemang": true, "kelapa gading": true,
}

var knownJogja = map[string]bool{
	"umy": true, "universitas muhammadiyah yogyakarta": true,
	"ugm": true, "universitas gadjah mada": true,
	"kauman": true, "masjid agung kauman": true,
	"malioboro": true, "tugu jogja": true, "stasiun tugu": true, "tugu yogyakarta": true,
	"taman sari": true, "prambanan": true, "jogja": true, "yogyakarta": true,
	"keraton": true, "keraton jogja": true, "alun alun": true, "alun-alun": true,
	"jalan malioboro": true, "pasar beringharjo": true, "gading": true,
	"kentungan": true, "seturan": true, "babarsari": true, "gejayan": true,
	"demangan": true, "timoho": true, "janti": true, "gowongan": true,
	"candi prambanan": true, "kaliurang": true, "parangtritis": true, "gunung merapi": true,
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
	// For Bare mode, extract waypoint names using Go text parsing.
	// DeepSeek V4 Flash's reasoning mode makes JSON-from-LLM unreliable,
	// so we use a deterministic extractor for consistency.
	names := extractWaypointNames(prompt)
	if len(names) < 2 {
		// Fallback: try LLM if extraction fails
		llmNames, err := e.llmFallback(ctx, prompt)
		if err != nil {
			return nil, fmt.Errorf("bare: could not extract waypoints: %w", err)
		}
		names = llmNames
	}
	return e.resolveWaypoints(ctx, names)
}

func (e *BareEngine) llmFallback(ctx context.Context, prompt string) ([]string, error) {
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

// extractWaypointNames parses common Indonesian route patterns to extract waypoints.
func extractWaypointNames(prompt string) []string {
	original := strings.TrimSpace(prompt)
	if original == "" {
		return nil
	}
	
	// Work with lowercased version for pattern matching, but extract from original
	lower := strings.ToLower(original)
	
	// Remove common prefixes
	for _, prefix := range []string{"saya ", "tolong ", "cari ", "buatkan "} {
		lower = strings.TrimPrefix(lower, prefix)
		original = strings.TrimPrefix(strings.TrimSpace(original), strings.TrimSpace(prefix))
	}
	
	// Extract positions in the original string
	// Strategy: track substring positions
	
	// Normalize: "mau ke" -> "ke", "mau mampir" -> "mampir", etc.
	normalized := strings.ReplaceAll(lower, " mau ke ", " ke ")
	normalized = strings.ReplaceAll(normalized, "mau ke ", "ke ")
	normalized = strings.ReplaceAll(normalized, "mau mampir ", "mampir ")
	normalized = strings.ReplaceAll(normalized, "mau ", "")
	
	parts := make([]string, 0)
	remaining := normalized
	
	// Track original characters for reconstruction
	// Since we normalized, we need to work with the normalized string
	
	// Extract "dari X"
	if idx := strings.Index(remaining, "dari "); idx >= 0 {
		afterDari := remaining[idx+5:]
		end := findNextMarker(afterDari)
		if end > 0 {
			word := strings.TrimSpace(afterDari[:end])
			parts = append(parts, titleCase(word))
			remaining = afterDari[end:]
		} else {
			word := strings.TrimSpace(afterDari)
			parts = append(parts, titleCase(word))
			return filterShort(parts)
		}
	}
	
	// Extract "ke X" segments
	for {
		idx := strings.Index(remaining, " ke ")
		if idx < 0 {
			break
		}
		afterKe := remaining[idx+4:]
		end := findNextMarker(afterKe)
		if end > 0 {
			word := strings.TrimSpace(afterKe[:end])
			parts = append(parts, titleCase(word))
			remaining = afterKe[end:]
		} else {
			word := strings.TrimSpace(afterKe)
			parts = append(parts, titleCase(word))
			break
		}
	}
	
	// Extract "mampir X" segments
	for {
		idx := strings.Index(remaining, "mampir ")
		if idx < 0 {
			break
		}
		afterMampir := strings.TrimSpace(remaining[idx+7:])
		
		// Check for "mampir X untuk Y" — X is the stop, Y is the purpose (POI search)
		if untukIdx := strings.Index(afterMampir, " untuk "); untukIdx >= 0 {
			stop := strings.TrimSpace(afterMampir[:untukIdx])
			poi := strings.TrimSpace(afterMampir[untukIdx+7:])
			if stop != "" {
				parts = append(parts, titleCase(stop))
			}
			if poi != "" {
				parts = append(parts, poi)
			}
		} else {
			// mampir X — X could be a place name or a POI query
			parts = append(parts, afterMampir)
		}
		break
	}
	
	return filterShort(parts)
}

func titleCase(s string) string {
	if len(s) == 0 {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

func findNextMarker(s string) int {
	markers := []string{" ke ", " mampir ", " dan ", " untuk "}
	earliest := -1
	for _, m := range markers {
		idx := strings.Index(s, m)
		if idx >= 0 && (earliest < 0 || idx < earliest) {
			earliest = idx
		}
	}
	return earliest
}

func filterShort(parts []string) []string {
	result := make([]string, 0, len(parts))
	for _, p := range parts {
		if len(p) > 2 {
			result = append(result, p)
		}
	}
	return result
}

func (e *BareEngine) resolveWaypoints(ctx context.Context, names []string) ([]Waypoint, error) {
	waypoints := make([]Waypoint, 0, len(names))
	for _, name := range names {
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}

		if isVague(name) {
			anchor := e.geocoder.Geocode("Jakarta")
			if anchor != nil && anchor.Error == "" {
				pois := e.poiSrch.Search(name, anchor.Lat, anchor.Lng, 5)
				if len(pois) > 0 {
					wp := Waypoint{Name: pois[0].Name, Lat: pois[0].Lat, Lng: pois[0].Lng, DisplayName: pois[0].Category}
					waypoints = append(waypoints, wp)
					continue
				}
			}
		}
		// Named place: geocode it
		// Add city context for known landmarks without it
		geoName := name
		nameLower := strings.ToLower(name)
		if !strings.Contains(nameLower, "jakarta") && !strings.Contains(nameLower, "yogyakarta") && !strings.Contains(nameLower, "jogja") {
			if knownJakarta[nameLower] {
				geoName = name + ", Jakarta"
			} else if knownJogja[nameLower] {
				geoName = name + ", Yogyakarta"
			}
		}
		result := e.geocoder.Geocode(geoName)
		if result.Error != "" {
			return nil, fmt.Errorf("bare: geocode %q: %s", name, result.Error)
		}
		waypoints = append(waypoints, Waypoint{
			Name: result.Name, Lat: result.Lat, Lng: result.Lng, DisplayName: result.DisplayName,
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


