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

const AgentPrompt = `You are ARAHIN, a route planner. Your job is to identify the places in the user's request, geocode them, and submit the complete route.

STRICT RULES — READ CAREFULLY:

1. ONLY use places EXPLICITLY mentioned in the user's request. Do NOT invent or guess landmarks, attractions, or waypoints. If the user says "dari UMY ke Masjid Agung Kauman mampir makan bebek", the places are: UMY, Masjid Agung Kauman, and a duck restaurant. NOT "Taman Pintar" or any other place.

2. Identify the pattern:
   - "dari X" = start
   - "ke Y" = destination or stop
   - "mampir/mampir untuk Z" = intermediate stop or POI search

3. Geocode each place EXACTLY ONCE using geocode(). Include city context.
   If the result is wrong (city outside expected area), try a more specific name.

4. For vague requests ("makan bebek", "sate"), use poi_search() near the relevant waypoint. Max 2 searches per unique query. After 2 attempts, pick the best result.

5. When ALL waypoints are collected (each has name+lat+lng), call submit_waypoints() with the ordered waypoints list. Do NOT output JSON directly — always use submit_waypoints.

6. You may call multiple tools per turn to be efficient. When all waypoints are ready, call submit_waypoints().

7. If a geocode returns an error, try once more with a simpler name. If it still fails, skip that place and continue.

Remember: you ONLY know the places I told you. Do not add places.`

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
				Description: "Find POIs near coordinates. Keyword mode for 'sate enak'. Category mode for 'masjid'.",
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
		{
			Type: "function",
			Function: llm.ToolFuncDef{
				Name:        "submit_waypoints",
				Description: "Call this when ALL waypoints are collected. Submit the complete ordered route.",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"waypoints": map[string]interface{}{
							"type": "array",
							"items": map[string]interface{}{
								"type": "object",
								"properties": map[string]interface{}{
									"name": map[string]interface{}{"type": "string", "description": "Display name"},
									"lat":  map[string]interface{}{"type": "number"},
									"lng":  map[string]interface{}{"type": "number"},
								},
								"required": []string{"name", "lat", "lng"},
							},
						},
					},
					"required": []string{"waypoints"},
				},
			},
		},
	}
}
