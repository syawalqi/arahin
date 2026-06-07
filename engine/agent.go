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
	Progress ProgressFunc
}

func NewAgentEngine(provider llm.Provider, model string, gc *tools.Geocoder, ps *tools.POISearcher, progress ...ProgressFunc) *AgentEngine {
	e := &AgentEngine{
		llm:      provider,
		model:    model,
		geocoder: gc,
		poiSrch:  ps,
		maxTurns: 12,
	}
	if len(progress) > 0 {
		e.Progress = progress[0]
	}
	return e
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
		"geocode":          e.handleGeocode,
		"poi_search":       e.handlePOISearch,
		"submit_waypoints": e.handleSubmitWaypoints,
	}

	seenPlaces := make(map[string]bool)          // track already-geocoded places to avoid repeats
	poiQueries := make(map[string]int)           // track POI query attempts (query → count)
	maxPOISearches := 5                          // max total poi_search calls
	maxPOIPerQuery := 2                          // max attempts per unique query

	for turn := 0; turn < e.maxTurns; turn++ {
		if e.Progress != nil {
			e.Progress("reasoning", fmt.Sprintf("Agent turn %d/%d...", turn+1, e.maxTurns))
		}

		// Use streaming ChatCollect for the agent loop.
		// OpenCode Go API requires streaming — non-streaming returns 401.
		// For tool calls, the streaming accumulator collects tool call deltas.
		// If tool calls are missed (DeepSeek outputs reasoning-only), we fall back
		// to content-based parsing of embedded tool calls.
		resp, err := e.llm.ChatCollect(ctx, llm.ChatRequest{
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
				if e.Progress != nil {
					e.Progress("result", fmt.Sprintf("Found %d waypoints", len(waypoints)))
				}
				return waypoints, nil
			}

			// If content contains embedded tool call patterns, parse them manually.
			// DeepSeek V4 Flash sometimes outputs tool calls as text instead of
			// using the streaming tool_calls mechanism.
			if tc := tryExtractToolCalls(resp.Content); len(tc) > 0 {
				if e.Progress != nil {
					e.Progress("tool_call", fmt.Sprintf("Extracted %d tool call(s) from content", len(tc)))
				}
				resp.ToolCalls = tc
				// Continue to tool execution below
			} else {
				// Content might be DeepSeek's thinking trace (not JSON).
				if e.Progress != nil {
					e.Progress("tool_call", "LLM returned text without tool calls or JSON, requesting structured output...")
				}
				messages = append(messages, llm.Message{Role: llm.RoleAssistant, Content: resp.Content, ReasoningContent: resp.ReasoningContent})
				messages = append(messages, llm.Message{
					Role:    llm.RoleUser,
					Content: "Respond ONLY with a JSON array of waypoints, or call ONE tool. No thinking. No explanations.",
				})
				continue
			}
		}

		if e.Progress != nil {
			e.Progress("tool_call", fmt.Sprintf("LLM calls %d tool(s)", len(resp.ToolCalls)))
		}

		// LLM wants to call tools
		// Append assistant message with tool calls — preserve reasoning_content for DeepSeek
		assistantMsg := llm.Message{
			Role:             llm.RoleAssistant,
			Content:          "",
			ReasoningContent: resp.ReasoningContent,
			ToolCalls:        make([]llm.ToolCall, len(resp.ToolCalls)),
		}
		copy(assistantMsg.ToolCalls, resp.ToolCalls)
		messages = append(messages, assistantMsg)

		// Execute each tool call
		for _, tc := range resp.ToolCalls {
			// submit_waypoints: extract waypoints and return
			if tc.Function.Name == "submit_waypoints" {
				var args struct {
					Waypoints []struct {
						Name string  `json:"name"`
						Lat  float64 `json:"lat"`
						Lng  float64 `json:"lng"`
					} `json:"waypoints"`
				}
				if err := json.Unmarshal([]byte(tc.Function.Arguments), &args); err != nil {
					messages = append(messages, llm.Message{
						Role:       llm.RoleTool,
						ToolCallID: tc.ID,
						Content:    fmt.Sprintf("error: invalid submit_waypoints args: %v", err),
					})
					continue
				}
				if len(args.Waypoints) < 2 {
					messages = append(messages, llm.Message{
						Role:       llm.RoleTool,
						ToolCallID: tc.ID,
						Content:    "Need at least 2 waypoints (start and destination). Keep geocoding.",
					})
					continue
				}
				// Success — convert to waypoints and return
				wps := make([]Waypoint, len(args.Waypoints))
				for i, w := range args.Waypoints {
					wps[i] = Waypoint{Name: w.Name, Lat: w.Lat, Lng: w.Lng}
				}
				if e.Progress != nil {
					e.Progress("result", fmt.Sprintf("Route: %d waypoints submitted", len(wps)))
				}
				return wps, nil
			}

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
					if e.Progress != nil {
						e.Progress("error", "geocode: invalid args: "+err.Error())
					}
					continue
				}
				// Normalize key: trim, lowercase, apply alias for dedup
				key := strings.ToLower(strings.TrimSpace(args.PlaceName))
				for aliasKey, aliasVal := range nameAliases {
					if strings.Contains(key, aliasKey) {
						key = strings.ToLower(aliasVal)
						break
					}
				}
				if seenPlaces[key] {
					messages = append(messages, llm.Message{
						Role:       llm.RoleTool,
						ToolCallID: tc.ID,
						Content:    fmt.Sprintf("%q already geocoded, use cached result", args.PlaceName),
					})
					continue
				}
				seenPlaces[key] = true
				if e.Progress != nil {
					e.Progress("tool_call", fmt.Sprintf("geocode(%q)", args.PlaceName))
				}
			} else if tc.Function.Name == "poi_search" {
				// Check total limit
				poiTotal := 0
				for _, c := range poiQueries {
					poiTotal += c
				}
				if poiTotal >= maxPOISearches {
					messages = append(messages, llm.Message{
						Role:       llm.RoleTool,
						ToolCallID: tc.ID,
						Content:    "LIMIT REACHED: Already done 5 POI searches. Pick best result from previous searches and submit_waypoints.",
					})
					if e.Progress != nil {
						e.Progress("error", "POI search limit reached (max 5)")
					}
					continue
				}
				// Parse args to extract query for per-query dedup
				var poiArgs struct {
					Query string `json:"query"`
				}
				json.Unmarshal([]byte(tc.Function.Arguments), &poiArgs)
				qKey := strings.ToLower(strings.TrimSpace(poiArgs.Query))
				if qKey != "" && poiQueries[qKey] >= maxPOIPerQuery {
					messages = append(messages, llm.Message{
						Role:       llm.RoleTool,
						ToolCallID: tc.ID,
						Content:    fmt.Sprintf("Already searched %q twice. Pick best result from previous searches or try a different query.", poiArgs.Query),
					})
					if e.Progress != nil {
						e.Progress("error", fmt.Sprintf("POI query %q already attempted twice", poiArgs.Query))
					}
					continue
				}
				if qKey != "" {
					poiQueries[qKey]++
				}
				if e.Progress != nil {
					e.Progress("tool_call", fmt.Sprintf("poi_search(%s)", tc.Function.Arguments))
				}
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

	// Apply name aliases for places that geocode to the wrong city
	placeName := params.PlaceName
	lower := strings.ToLower(strings.TrimSpace(placeName))

	// Check alias by substring match — handles variations like
	// "Masjid Agung Kauman, Yogyakarta city" → "Masjid Gedhe Kauman, Yogyakarta"
	for aliasKey, aliasVal := range nameAliases {
		if strings.Contains(lower, aliasKey) {
			placeName = aliasVal
			break
		}
	}

	result := e.geocoder.Geocode(placeName)
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

// handleSubmitWaypoints is a fallback handler in case submit_waypoints reaches the
// generic tool handler (the main loop handles it before dispatch).
func (e *AgentEngine) handleSubmitWaypoints(args json.RawMessage) string {
	// This shouldn't be reached since the main loop handles submit_waypoints
	// before the generic handler dispatch. If it does, it means the LLM called
	// it but our loop didn't intercept — return the args as JSON so the LLM
	// can re-attempt.
	return `{"status": "use submit_waypoints result directly"}`
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

// tryExtractToolCalls attempts to parse tool call patterns from LLM text content.
// DeepSeek V4 Flash sometimes outputs tool calls as text (e.g., 'geocode("UMY")')
// instead of using the streaming tool_calls delta mechanism. This extracts them.
func tryExtractToolCalls(content string) []llm.ToolCall {
	content = strings.TrimSpace(content)
	if content == "" {
		return nil
	}

	var toolCalls []llm.ToolCall
	callID := 0

	// Pattern: look for function name followed by JSON-like arguments
	// Supported: geocode(...) or poi_search(...) with either JSON or quoted-string args
	lower := content

	// Extract geocode("Place Name, City") or geocode({"place_name":"..."})
	for {
		idx := strings.Index(lower, "geocode(")
		if idx < 0 {
			break
		}
		// Find matching closing paren
		rest := content[idx+8:] // after "geocode("
		depth := 1
		pos := 0
		for pos < len(rest) && depth > 0 {
			if rest[pos] == '(' {
				depth++
			} else if rest[pos] == ')' {
				depth--
			}
			pos++
		}
		if depth != 0 {
			break
		}
		argsStr := strings.TrimSpace(rest[:pos-1])
		callID++
		tc := llm.ToolCall{
			ID:   fmt.Sprintf("text_geocode_%d", callID),
			Type: "function",
			Function: llm.ToolFunction{
				Name:      "geocode",
				Arguments: normalizeArgs(argsStr),
			},
		}
		toolCalls = append(toolCalls, tc)
		// Remove processed portion and continue
		endPos := idx + 8 + pos
		if endPos >= len(content) {
			break
		}
		lower = lower[endPos:]
		content = content[endPos:]
	}

	// Extract poi_search(args...)
	for {
		idx := strings.Index(lower, "poi_search(")
		if idx < 0 {
			break
		}
		rest := content[idx+11:] // after "poi_search("
		depth := 1
		pos := 0
		for pos < len(rest) && depth > 0 {
			if rest[pos] == '(' {
				depth++
			} else if rest[pos] == ')' {
				depth--
			}
			pos++
		}
		if depth != 0 {
			break
		}
		argsStr := strings.TrimSpace(rest[:pos-1])
		callID++
		tc := llm.ToolCall{
			ID:   fmt.Sprintf("text_poi_%d", callID),
			Type: "function",
			Function: llm.ToolFunction{
				Name:      "poi_search",
				Arguments: normalizeArgs(argsStr),
			},
		}
		toolCalls = append(toolCalls, tc)
		endPos := idx + 11 + pos
		if endPos >= len(content) {
			break
		}
		lower = lower[endPos:]
		content = content[endPos:]
	}

	if len(toolCalls) > 0 {
		return toolCalls
	}
	return nil
}

// normalizeArgs converts a quoted-string argument list to a JSON object.
// e.g., '"UMY, Yogyakarta"' -> '{"place_name":"UMY, Yogyakarta"}'
// e.g., '{"place_name":"UMY"}' -> '{"place_name":"UMY"}' (pass-through)
func normalizeArgs(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return "{}"
	}
	// Already JSON object
	if strings.HasPrefix(s, "{") {
		return s
	}
	// Strip surrounding quotes
	s = strings.Trim(s, `"'`)
	// Wrap as place_name for geocode
	return fmt.Sprintf(`{"place_name":"%s"}`, strings.ReplaceAll(s, `"`, `\"`))
}


