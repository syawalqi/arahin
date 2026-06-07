package engine

import "github.com/syawalqi/arahin/llm"

// System prompts for the 3 modes, copied from V2 Implementation Guide.

const BarePrompt = `Extract all waypoints as a flat JSON array of strings.
Include city: ["Monas, Jakarta", "sate dekat Cawang", "Taman Mini, Jakarta"]
Output ONLY the JSON array.`

const PipelinePrompt = `Classify each waypoint in this route description into task types.
Output ONLY a JSON array of task objects:
- "type": "named" (exact place — geocode it), "query" (vague description — POI search), or "category" (amenity type — POI search)
- "place": for named types, the place name with city context
- "query" or "category": for POI search types
- "anchor": for POI search types, the nearest named place as search center

Output tasks in route sequence order: start → mid stops → destination.
Search/category tasks belong immediately after their anchor point.

Example: [{"type": "named", "place": "Monas, Jakarta"}, {"type": "query", "query": "sate", "anchor": "Cawang, Jakarta"}, {"type": "named", "place": "Taman Mini, Jakarta"}]`

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
