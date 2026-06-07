package engine

import "github.com/syawalqi/arahin/llm"

// System prompts for the 3 modes, copied from V2 Implementation Guide.

const BarePrompt = `Extract all waypoints as a JSON array of strings.
Each waypoint includes the city name.
Return ONLY the JSON array. No thinking.
Example: ["Monas, Jakarta", "sate dekat Cawang", "Taman Mini, Jakarta"]`

const PipelinePrompt = `Extract all waypoints as a JSON array of strings.
Include city context. Put vague descriptions and POI types (masjid, restoran, etc.) as separate entries.
Return ONLY the JSON array. No thinking.
Example: ["Monas, Jakarta", "sate dekat Cawang", "Taman Mini, Jakarta"]`

const AgentPrompt = `You are ARAHIN. Extract waypoints from the user's route request.
Be terse. No greetings, no emoji, no explanations.

Tools:
- geocode(place_name): convert place name to coordinates. Include city context.
- poi_search(query, lat, lng, radius?): find POIs near coordinates.
  Use keyword mode for vague descriptions: poi_search("sate enak", ...)
  Use category mode for structured: poi_search("masjid", ...)

Rules:
1. Call ONE tool per turn.
2. Call poi_search at most TWICE per search. Pick best result, move on.
3. Do NOT re-geocode the same place.
4. When ALL waypoints collected, output ONLY a JSON array:
   [{"name": "...", "lat": ..., "lng": ...}, ...]`

// ToolSchemas returns the tool definitions for Agent mode.
func ToolSchemas() []llm.ToolDef {
	return []llm.ToolDef{
		{
			Type: "function",
			Function: llm.ToolFuncDef{
				Name:        "geocode",
				Description: "Convert place name to coordinates. Include city: 'Monas, Jakarta'",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"place_name": map[string]interface{}{
							"type":        "string",
							"description": "Place with city context",
						},
					},
					"required": []string{"place_name"},
				},
			},
		},
		{
			Type: "function",
			Function: llm.ToolFuncDef{
				Name:        "poi_search",
				Description: "Find POIs near coordinates. Keyword mode for 'sate enak'. Category mode for 'cari masjid'.",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"query": map[string]interface{}{
							"type":        "string",
							"description": "Search term (keyword or category)",
						},
						"lat": map[string]interface{}{
							"type": "number",
						},
						"lng": map[string]interface{}{
							"type": "number",
						},
						"radius": map[string]interface{}{
							"type":    "integer",
							"default": 3,
						},
					},
					"required": []string{"query", "lat", "lng"},
				},
			},
		},
	}
}
