package engine

import (
	"context"
	"encoding/json"
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
	resp, err := e.llm.Chat(ctx, llm.ChatRequest{
		Model:       e.model,
		Temperature: 0.1,
		MaxTokens:   1024,
		Messages: []llm.Message{
			{Role: llm.RoleSystem, Content: PipelinePrompt},
			{Role: llm.RoleUser, Content: prompt},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("pipeline: llm: %w", err)
	}

	tasks := tryParseTasks(resp.Content)
	if tasks == nil {
		return nil, fmt.Errorf("pipeline: LLM did not return task JSON (got: %s)", truncate(resp.Content, 200))
	}

	waypoints := make([]Waypoint, 0, len(tasks))
	anchorCache := make(map[string]*Waypoint)

	for _, t := range tasks {
		switch t.Type {
		case "named":
			if t.Place == "" {
				continue
			}
			r := e.geocoder.Geocode(t.Place)
			if r.Error != "" {
				return nil, fmt.Errorf("pipeline: geocode %q: %s", t.Place, r.Error)
			}
			wp := Waypoint{Name: r.Name, Lat: r.Lat, Lng: r.Lng, DisplayName: r.DisplayName}
			anchorCache[strings.ToLower(t.Place)] = &wp
			waypoints = append(waypoints, wp)

		case "query":
			poi, err := e.searchPOI(t.Query, t.Anchor, anchorCache)
			if err != nil {
				return nil, fmt.Errorf("pipeline: %w", err)
			}
			waypoints = append(waypoints, *poi)

		case "category":
			poi, err := e.searchPOI(t.Category, t.Anchor, anchorCache)
			if err != nil {
				return nil, fmt.Errorf("pipeline: %w", err)
			}
			waypoints = append(waypoints, *poi)

		default:
			return nil, fmt.Errorf("pipeline: unknown task type: %s", t.Type)
		}
	}

	if len(waypoints) < 2 {
		return nil, fmt.Errorf("pipeline: need at least 2 waypoints, got %d", len(waypoints))
	}

	return waypoints, nil
}

func (e *PipelineEngine) searchPOI(query, anchor string, cache map[string]*Waypoint) (*Waypoint, error) {
	anchorKey := strings.ToLower(anchor)

	// Check cache for already-geocoded anchor
	anchorWP, ok := cache[anchorKey]
	if !ok {
		// Anchor not cached — geocode it now
		r := e.geocoder.Geocode(anchor)
		if r.Error != "" {
			return nil, fmt.Errorf("anchor geocode %q: %s", anchor, r.Error)
		}
		anchorWP = &Waypoint{Name: r.Name, Lat: r.Lat, Lng: r.Lng, DisplayName: r.DisplayName}
		cache[anchorKey] = anchorWP
	}

	pois := e.poiSrch.Search(query, anchorWP.Lat, anchorWP.Lng, 3)
	if len(pois) == 0 {
		// Fallback: return the anchor point itself with a note
		return &Waypoint{
			Name:        query + " (near " + anchorWP.Name + ")",
			Lat:         anchorWP.Lat,
			Lng:         anchorWP.Lng,
			DisplayName: "no nearby POI found, using anchor",
		}, nil
	}

	return &Waypoint{
		Name:        pois[0].Name,
		Lat:         pois[0].Lat,
		Lng:         pois[0].Lng,
		DisplayName: pois[0].Category,
	}, nil
}

func tryParseTasks(content string) []pipelineTask {
	content = strings.TrimSpace(content)
	var tasks []pipelineTask
	if err := json.Unmarshal([]byte(content), &tasks); err == nil {
		return tasks
	}
	// Try finding JSON array
	start := strings.Index(content, "[")
	end := strings.LastIndex(content, "]")
	if start >= 0 && end > start {
		if err := json.Unmarshal([]byte(content[start:end+1]), &tasks); err == nil {
			return tasks
		}
	}
	return nil
}
