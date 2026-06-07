package engine

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/syawalqi/arahin/llm"
	"github.com/syawalqi/arahin/tools"
)

type AgentEngine struct {
	llm      llm.Provider
	model    string
	geocoder *tools.Geocoder
	poiSrch  *tools.POISearcher
	maxTurns int
}

func NewAgentEngine(provider llm.Provider, model string, gc *tools.Geocoder, ps *tools.POISearcher) *AgentEngine {
	return &AgentEngine{
		llm:      provider,
		model:    model,
		geocoder: gc,
		poiSrch:  ps,
		maxTurns: 8,
	}
}

func (e *AgentEngine) Name() Mode { return ModeAgent }

// toolFuncs maps tool names to their implementations.
type toolFunc func(args json.RawMessage) string

func (e *AgentEngine) Plan(ctx context.Context, prompt string) ([]Waypoint, error) {
	messages := []llm.Message{
		{Role: llm.RoleSystem, Content: AgentPrompt},
		{Role: llm.RoleUser, Content: prompt},
	}

	tools := ToolSchemas()
	toolHandlers := map[string]toolFunc{
		"geocode": e.handleGeocode,
		"poi_search": e.handlePOISearch,
	}

	seenPlaces := make(map[string]bool) // track already-geocoded places to avoid repeats

	for turn := 0; turn < e.maxTurns; turn++ {
		resp, err := e.llm.Chat(ctx, llm.ChatRequest{
			Model:       e.model,
			Temperature: 0.3,
			MaxTokens:   2048,
			Messages:    messages,
			Tools:       tools,
		})
		if err != nil {
			return nil, fmt.Errorf("agent turn %d: %w", turn, err)
		}

		// Check if LLM produced final waypoints (JSON array)
		if len(resp.ToolCalls) == 0 {
			waypoints := tryParseWaypoints(resp.Content)
			if waypoints != nil {
				return waypoints, nil
			}
			// LLM produced text but not valid JSON — push retry
			messages = append(messages, llm.Message{Role: llm.RoleAssistant, Content: resp.Content})
			messages = append(messages, llm.Message{
				Role:    llm.RoleUser,
				Content: "Respond with ONLY a JSON array: [{\"name\": \"...\", \"lat\": ..., \"lng\": ...}, ...]",
			})
			continue
		}

		// LLM wants to call tools
		// Append assistant message with tool calls
		assistantMsg := llm.Message{
			Role:      llm.RoleAssistant,
			Content:   "",
			ToolCalls: make([]llm.ToolCall, len(resp.ToolCalls)),
		}
		copy(assistantMsg.ToolCalls, resp.ToolCalls)
		messages = append(messages, assistantMsg)

		// Execute each tool call
		for _, tc := range resp.ToolCalls {
			if tc.Function.Name == "geocode" {
				var args struct {
					PlaceName string `json:"place_name"`
				}
				if err := json.Unmarshal([]byte(tc.Function.Arguments), &args); err != nil {
					messages = append(messages, llm.Message{
						Role:       llm.RoleTool,
						ToolCallID: tc.ID,
						Content:    fmt.Sprintf("error: invalid args: %v", err),
					})
					continue
				}
				key := strings.ToLower(strings.TrimSpace(args.PlaceName))
				if seenPlaces[key] {
					messages = append(messages, llm.Message{
						Role:       llm.RoleTool,
						ToolCallID: tc.ID,
						Content:    fmt.Sprintf("%q already geocoded, use cached result", args.PlaceName),
					})
					continue
				}
				seenPlaces[key] = true
			}

			result := toolHandlers[tc.Function.Name](json.RawMessage(tc.Function.Arguments))
			messages = append(messages, llm.Message{
				Role:       llm.RoleTool,
				ToolCallID: tc.ID,
				Content:    result,
			})
		}
	}

	return nil, fmt.Errorf("agent: max %d turns reached without collecting all waypoints", e.maxTurns)
}

func (e *AgentEngine) handleGeocode(args json.RawMessage) string {
	var params struct {
		PlaceName string `json:"place_name"`
	}
	if err := json.Unmarshal(args, &params); err != nil {
		return fmt.Sprintf(`{"error": "invalid arguments: %v"}`, err)
	}
	result := e.geocoder.Geocode(params.PlaceName)
	if result.Error != "" {
		return fmt.Sprintf(`{"error": "%s"}`, result.Error)
	}
	b, _ := json.Marshal(result)
	return string(b)
}

func (e *AgentEngine) handlePOISearch(args json.RawMessage) string {
	var params struct {
		Query  string  `json:"query"`
		Lat    float64 `json:"lat"`
		Lng    float64 `json:"lng"`
		Radius int     `json:"radius"`
	}
	if err := json.Unmarshal(args, &params); err != nil {
		return fmt.Sprintf(`{"error": "invalid arguments: %v"}`, err)
	}
	pois := e.poiSrch.Search(params.Query, params.Lat, params.Lng, params.Radius)
	if len(pois) == 0 {
		return `[]`
	}
	b, _ := json.Marshal(pois)
	return string(b)
}

// tryParseWaypoints attempts to parse LLM output as a Waypoint JSON array.
func tryParseWaypoints(content string) []Waypoint {
	content = strings.TrimSpace(content)
	var wps []Waypoint
	if err := json.Unmarshal([]byte(content), &wps); err == nil && len(wps) > 0 {
		return wps
	}
	// Try finding JSON array
	start := strings.Index(content, "[")
	end := strings.LastIndex(content, "]")
	if start >= 0 && end > start {
		if err := json.Unmarshal([]byte(content[start:end+1]), &wps); err == nil && len(wps) > 0 {
			return wps
		}
	}
	return nil
}


