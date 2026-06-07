#!/usr/bin/env plan.md
# ARAHIN V2 — Route Planner Implementation Plan

**Goal:** Implement a Go-based multi-stop route planner with 3 modes (Bare/Pipeline/Agent), shared post-processing (OSRM → TSP → Leaflet map), and a web server for the demo video.

**Architecture:** Add new packages alongside existing FLARE-derived code. Reuse existing `llm` package (OpenAI client with streaming + tool calling). Build 4 spatial tools, 3 engine modes, a Leaflet HTML renderer, and an HTTP server with SSE streaming.

**Tech Stack:** Go 1.26, standard lib only (net/http, html/template, encoding/json). Leaflet.js from CDN. External APIs: Nominatim, Overpass, OSRM (all free, no keys required).

**Dataset:** MSNTB — 100 prompts, 2-5 waypoints per route. Brute-force TSP sufficient (no OR-Tools needed).

---

## Project Structure (new files only)

```
/root/arahin/
├── main.go                           # +2 subcommands: "route", "serve"
├── cmd/
│   └── route.go                      # (NEW) CLI route subcommand handler
│   └── serve.go                      # (NEW) web server subcommand handler
├── tools/
│   ├── geocode.go                    # (NEW) Nominatim → coordinates
│   ├── poiscarch.go                  # (NEW) Overpass → POI list
│   ├── route.go                      # (NEW) OSRM route + table API
│   └── tsp.go                        # (NEW) brute-force TSP optimizer
├── engine/
│   ├── types.go                      # (NEW) shared types + waypoint struct
│   ├── prompts.go                    # (NEW) 3 system prompts + tool schemas
│   ├── bare.go                       # (NEW) Bare mode: 1 LLM call, no tools
│   ├── pipeline.go                   # (NEW) Pipeline mode: 1 LLM call → deterministic dispatch
│   └── agent.go                      # (NEW) Agent mode: ReAct loop with tool calling
├── web/
│   ├── server.go                     # (NEW) HTTP server + SSE handler
│   └── handler.go                    # (NEW) route API handler
├── render/
│   └── map.go                        # (NEW) Leaflet HTML page generator
```

---

## Pre-Build Verification

Before implementing, verify:
1. Go compiles the existing project: `cd /root/arahin && go build .` (use full Go path)
2. OpenCode Go API is reachable (existing `llm` package already uses it)
3. External APIs respond:
   - `curl -s "https://nominatim.openstreetmap.org/search?q=Monas,Jakarta&format=json&limit=1"`
   - `curl -s "https://router.project-osrm.org/table/v1/driving/106.8,-6.2;106.9,-6.3?annotations=distance"`
   - `curl -s "https://overpass-api.de/api/interpreter" -d "data=[out:json];node[\"amenity\"=\"restaurant\"](around:1000,-6.2,106.8);out 3;"`

---

## Tasks

### Phase 0: Config & Setup

#### Task 0: Add config defaults and project scaffolding

**Objective:** Add ARAHIN-specific config fields and create empty package directories.

**Files:**
- Modify: `/root/arahin/config/defaults.go` — add default mode, port
- Modify: `/root/arahin/config/config.go` — add RouteConfig struct
- Create: `tools/`, `engine/`, `web/`, `render/` directories
- Create: package stub files

**Step 1: Add config fields**

Add to `defaults.go`:
```go
const (
    DefaultRouteMode  = "pipeline"
    DefaultRoutePort  = "9122"
    DefaultRouteModel = "opencode-go/deepseek-v4-flash"
)
```

Add to `config.go` `Config` struct:
```go
type Config struct {
    // ... existing fields ...
    
    Route RouteConfig `yaml:"route"`
}

type RouteConfig struct {
    Mode  string `yaml:"mode"`
    Port  string `yaml:"port"`
    Model string `yaml:"model"`
}
```

Update `Default()`:
```go
Route: RouteConfig{
    Mode:  DefaultRouteMode,
    Port:  DefaultRoutePort,
    Model: DefaultRouteModel,
},
```

**Step 2: Create stub packages**
```go
// tools/tools.go — package tools
package tools

// NewGeocoder() etc. live in their own files
```

**Step 3: Verify compilation**
```bash
cd /root/arahin && go build .
```
Expected: compiles clean.

**Step 4: Commit**
```bash
git add config/ tools/ engine/ web/ render/
git commit -m "feat: add route planner scaffolding and config"
```

---

### Phase 1: Tools (no dependencies between them)

#### Task 1: geocode tool (Nominatim API)

**Objective:** Convert place name to coordinates via Nominatim. Rate-limited (1 req/s). Includes caching.

**Files:**
- Create: `/root/arahin/tools/geocode.go`

**Implementation details:**
```go
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
    DisplayName string  `json:"display_name"`
    Error       string  `json:"error,omitempty"`
}

type Geocoder struct {
    client    *http.Client
    cache     map[string]*GeocodeResult
    mu        sync.Mutex
    lastReq   time.Time
}

func NewGeocoder() *Geocoder {
    return &Geocoder{
        client: &http.Client{Timeout: 15 * time.Second},
        cache:  make(map[string]*GeocodeResult),
    }
}

func (g *Geocoder) Geocode(placeName string) *GeocodeResult {
    // 1. Check cache (with mutex)
    // 2. Rate limit: wait if <1s since last request
    // 3. GET nominatim.openstreetmap.org/search
    //    params: q=placeName, format=json, limit=5
    //    headers: User-Agent=ARAHIN/1.0, Accept-Language=id,en
    // 4. Parse response, pick best result (name match priority)
    // 5. Cache and return
    // 6. On failure: return GeocodeResult{Error: msg}
}

func (g *Geocoder) pickBest(candidates []geocodeCandidate, query string) geocodeCandidate {
    // Prefer exact name match, prefer known Indonesian cities
    // See V2 Implementation Guide for detailed scoring logic
}
```

**Key behaviors:**
- Cache by lowercase place name
- Rate limit: 1 request per second (Nominatim TOS)
- Timeout: 15 seconds
- On 0 results: return Error
- Indonesia city bias in pickBest

**Verification:**
- Test: `go build ./tools/`

#### Task 2: poi_search tool (Overpass API)

**Objective:** Find POIs near coordinates via Overpass API. Two modes: keyword regex search and category (amenity tag) search.

**Files:**
- Create: `/root/arahin/tools/poisearch.go`

**Implementation details:**
```go
type POI struct {
    Name     string  `json:"name"`
    Lat      float64 `json:"lat"`
    Lng      float64 `json:"lng"`
    Category string  `json:"category"`
}

type POISearcher struct {
    client *http.Client
}

func NewPOISearcher() *POISearcher

func (s *POISearcher) Search(query string, lat, lng float64, radiusKm int) []POI
```

**Key behaviors:**
- Category mode: if query matches a known category key (masjid→mosque, restoran→restaurant, etc.), query by amenity tag
- Keyword mode: extract keywords (remove stop words), search OSM names with regex case-insensitive
- Dedup by rounded coordinates (~100m grid)
- Return max 5 results
- Timeout: 25 seconds
- Overpass endpoint: `https://overpass-api.de/api/interpreter`

**Verification:**
- Test: `go build ./tools/`

#### Task 3: route tool (OSRM API)

**Objective:** Get driving routes and distance/duration matrices via OSRM.

**Files:**
- Create: `/root/arahin/tools/route.go`

**Two functions:**
```go
type RouteSegment struct {
    DistanceKm  float64   `json:"distance_km"`
    DurationMin float64   `json:"duration_min"`
    Geometry    Geometry  `json:"geometry"` // GeoJSON LineString
    Steps       []Step    `json:"steps"`
}

func GetRoute(origin, destination GeocodeResult) (*RouteSegment, error)
// GET /route/v1/driving/{lng},{lat};{lng},{lat}?overview=full&geometries=geojson&steps=true

type DistanceMatrix struct {
    Durations [][]float64 `json:"durations"`
    Distances [][]float64 `json:"distances"`
}

func GetDistanceMatrix(waypoints []GeocodeResult) (*DistanceMatrix, error)
// GET /table/v1/driving/{lng},{lat};{lng},{lat}...?annotations=distance,duration
```

**Key behaviors:**
- OSRM public instance: `router.project-osrm.org`
- Timeout: 15s for route, 30s for table (multiple waypoints)
- Error handling: return nil on non-200 or code != "Ok"

#### Task 4: TSP optimizer (brute-force)

**Objective:** Optimal route ordering for n ≤ 5 waypoints via brute-force permutation.

**Files:**
- Create: `/root/arahin/tools/tsp.go`

```go
// OptimizeRoute finds the optimal ordering using brute-force permutation.
// startIdx and endIdx are fixed (user's start/destination).
// For n ≤ 5: brute-force all permutations of middle waypoints.
// For n > 5: return original order (thesis scope is 2-5 waypoints).
func OptimizeRoute(matrix *DistanceMatrix, startIdx, endIdx int) ([]int, float64)
```

**Key behaviors:**
- Uses duration matrix (seconds) as cost
- Fixed first/last waypoint (user's "dari X...ke Y")
- Returns ordered indices + total cost
- Brute-force: itertools.Permutations equivalent via Heap's algorithm or recursive

---

### Phase 2: Engine

#### Task 5: Shared types and prompts

**Objective:** Define shared waypoint types, system prompts, and tool schemas for all 3 modes.

**Files:**
- Create: `/root/arahin/engine/types.go`
- Create: `/root/arahin/engine/prompts.go`

**types.go:**
```go
package engine

type Mode string
const (
    ModeBare     Mode = "bare"
    ModePipeline Mode = "pipeline"
    ModeAgent    Mode = "agent"
)

type Waypoint struct {
    Name        string  `json:"name"`
    Lat         float64 `json:"lat"`
    Lng         float64 `json:"lng"`
    DisplayName string  `json:"display_name,omitempty"`
}

type RouteResult struct {
    Waypoints []Waypoint        `json:"waypoints"`
    DurationMin float64         `json:"duration_min"`
    DistanceKm  float64         `json:"distance_km"`
    Order       []int           `json:"order"`
    Segments    []*RouteSegment `json:"segments"`
    Error       string          `json:"error,omitempty"`
}

// Engine interface (all 3 modes implement)
type RouteEngine interface {
    Plan(ctx context.Context, prompt string) ([]Waypoint, error)
    Name() Mode
}
```

**prompts.go** — Copy exactly from V2 Implementation Guide:
- `BARE_PROMPT` — flat JSON array of strings
- `PIPELINE_PROMPT` — classify waypoints into named/query/category tasks
- `AGENT_PROMPT` — ReAct tool description + rules
- `TOOLS_SCHEMA` — geocode + poi_search function definitions for OpenAI tool calling

#### Task 6: Bare mode engine

**Objective:** Single LLM call, no tools. LLM extracts waypoint names as a flat string array, then geocode each.

**Files:**
- Create: `/root/arahin/engine/bare.go`

```go
type BareEngine struct {
    llm      llm.Provider
    model    string
    geocoder *tools.Geocoder
}

func (e *BareEngine) Plan(ctx context.Context, prompt string) ([]Waypoint, error) {
    // 1. Call LLM with BARE_PROMPT + user prompt
    //    LLM returns: ["Monas, Jakarta", "sate dekat Cawang", "Taman Mini, Jakarta"]
    // 2. Parse JSON string array
    // 3. For each name: geocode(name)
    //    If name is vague (contains search keywords), call poi_search instead
    // 4. Return waypoints
    // 5. Error handling: if LLM returns bad JSON, retry once
}
```

#### Task 7: Pipeline mode engine

**Objective:** Single LLM call classifies waypoints into task types. Deterministic dispatch for each task.

**Files:**
- Create: `/root/arahin/engine/pipeline.go`

```go
type PipelineEngine struct {
    llm        llm.Provider
    model      string
    geocoder   *tools.Geocoder
    poiSearch  *tools.POISearcher
}

func (e *PipelineEngine) Plan(ctx context.Context, prompt string) ([]Waypoint, error) {
    // 1. Call LLM with PIPELINE_PROMPT + user prompt
    //    LLM returns task JSON array:
    //    [{"type":"named","place":"Monas, Jakarta"},
    //     {"type":"query","query":"sate","anchor":"Cawang, Jakarta"},
    //     {"type":"named","place":"Taman Mini, Jakarta"}]
    // 2. Parse task array
    // 3. For each task:
    //    - "named": geocode(place)
    //    - "query": geocode(anchor) → poi_search(query, lat, lng) in keyword mode
    //    - "category": geocode(anchor) → poi_search(category, lat, lng) in category mode
    // 4. Cache anchor geocode results
    // 5. Return waypoints in route order
}
```

#### Task 8: Agent mode engine (ReAct loop)

**Objective:** ReAct loop where LLM iteratively calls geocode and poi_search tools until all waypoints collected.

**Files:**
- Create: `/root/arahin/engine/agent.go`

```go
type AgentEngine struct {
    llm       llm.Provider
    model     string
    geocoder  *tools.Geocoder
    poiSearch *tools.POISearcher
}

func (e *AgentEngine) Plan(ctx context.Context, prompt string) ([]Waypoint, error) {
    // ReAct loop:
    // 1. Start with AGENT_PROMPT + user prompt
    // 2. Call LLM with tool definitions (geocode, poi_search)
    // 3. If LLM returns JSON waypoint array → done, parse and return
    // 4. If LLM returns tool call → execute tool, append result as tool message
    // 5. Loop with updated message history
    // 6. Max 8 iterations
    // 7. Return waypoints or error
}
```

**Tool schemas for Agent mode** (copied from V2):
```json
{
  "type": "function",
  "function": {
    "name": "geocode",
    "description": "Convert place name to coordinates. Include city: 'Monas, Jakarta'",
    "parameters": {
      "type": "object",
      "properties": {
        "place_name": {"type": "string", "description": "Place with city context"}
      },
      "required": ["place_name"]
    }
  }
}
```

The agent loop pattern already exists in `/root/arahin/agent/loop.go` — we create a lighter version here that only handles geocode + poi_search tools (no bash execution).

---

### Phase 3: Output

#### Task 9: Map renderer (Leaflet HTML)

**Objective:** Generate an interactive Leaflet map HTML page showing waypoints, route polyline, and distance labels.

**Files:**
- Create: `/root/arahin/render/map.go`

```go
package render

type MapConfig struct {
    Waypoints []engine.Waypoint
    Segments  []*tools.RouteSegment
    TotalKM   float64
    TotalMin  float64
}

func RenderRouteHTML(cfg MapConfig) (string, error) {
    // html/template with Leaflet CDN
    // Shows:
    //   - Colored markers: green=start, blue=mid, red=end
    //   - Polyline along actual road geometry
    //   - Distance labels per segment
    //   - Info panel with total distance + duration
    //   - Fit bounds to all waypoints
}
```

**Key constraint:** Pure Go `html/template`, no external template engine.

#### Task 10: Shared post-processing (route + TSP + map)

**Objective:** After waypoints are extracted by any engine, apply shared post-processing: OSRM routing → TSP optimization → map render → output.

**Implementation in `cmd/route.go`:**
```go
func runRoute(ctx context.Context, cfg *config.Config, prompt string) (*engine.RouteResult, string, error) {
    // 1. Select engine based on cfg.Route.Mode
    // 2. engine.Plan(ctx, prompt) → []Waypoint
    // 3. OSRM GetDistanceMatrix(waypoints) → matrix
    // 4. TSP OptimizeRoute(matrix, 0, len-1) → []int order
    // 5. Reorder waypoints by TSP order
    // 6. OSRM GetRoute for each consecutive pair → []RouteSegment
    // 7. RenderRouteHTML → HTML string
    // 8. Save route.html to disk
    // 9. Return RouteResult + HTML path
}
```

---

### Phase 4: Web Server

#### Task 11: HTTP server with SSE streaming

**Objective:** Single-page web app with input form, SSE streaming of LLM progress, and Leaflet map display.

**Files:**
- Create: `/root/arahin/web/server.go`
- Create: `/root/arahin/web/handler.go`

**server.go:**
```go
package web

import (
    "html/template"
    "net/http"
)

type Server struct {
    cfg    *config.Config
    router http.Handler
}

func NewServer(cfg *config.Config) *Server
func (s *Server) ListenAndServe() error
```

**Endpoints:**
- `GET /` — serve HTML page with:
  - Text input for route description
  - Mode selector (bare / pipeline / agent)
  - Submit button
  - Streaming output area (tool calls, reasoning)
  - Leaflet map div
  
- `POST /api/route` — JSON API for non-streaming route planning
  - Request: `{"prompt": "...", "mode": "agent"}`
  - Response: `{"waypoints": [...], "segments": [...], "map_html": "..."}`
  
- `GET /api/route/stream` — SSE endpoint for streaming
  - Query: `?prompt=...&mode=agent`
  - Events: `event: reasoning`, `event: tool_call`, `event: waypoints`, `event: map`, `event: error`
  - Maps to Agent mode streaming output

**HTML page** (embedded in handler.go via `//go:embed` template):
```html
<!DOCTYPE html>
<html>
<head>
    <title>ARAHIN — Route Planner</title>
    <!-- Leaflet CSS + JS from CDN -->
    <!-- Tailwind CSS from CDN -->
    <!-- Alpine.js for reactive UI -->
</head>
<body>
    <!-- Input form with x-data -->
    <!-- Streaming log area -->
    <!-- Leaflet map container -->
    <!-- JavaScript: EventSource for SSE -->
</body>
</html>
```

#### Task 12: Wire into main.go

**Files:**
- Create: `/root/arahin/cmd/route.go`
- Create: `/root/arahin/cmd/serve.go`
- Modify: `/root/arahin/main.go` — add "route" and "serve" subcommands

Add to `main.go` switch statement:
```go
case "route":
    if len(os.Args) < 3 {
        fmt.Fprintln(os.Stderr, "usage: arahin route <prompt>")
        os.Exit(1)
    }
    prompt := strings.Join(os.Args[2:], " ")
    if err := cmd.Route(cfg, prompt); err != nil {
        fmt.Fprintf(os.Stderr, "route error: %v\n", err)
        os.Exit(1)
    }
case "serve":
    if err := cmd.Serve(cfg); err != nil {
        fmt.Fprintf(os.Stderr, "serve error: %v\n", err)
        os.Exit(1)
    }
```

---

### Phase 5: Build & Test

#### Task 13: Build, verify, smoke test

**Files:** none (terminal commands)

```bash
# Build
cd /root/arahin && /usr/local/go/bin/go build -ldflags="-s -w" -o arahin .

# Smoke tests
## 1. CLI route test
./arahin route "dari Monas ke Taman Mini, mampir sate dekat Cawang"

## 2. Web server
./arahin serve &
curl -s http://localhost:9122/ | head -5
curl -s -X POST http://localhost:9122/api/route \
    -H 'Content-Type: application/json' \
    -d '{"prompt":"dari Monas ke Taman Mini","mode":"pipeline"}' | head -10

## 3. Verify route.html exists
ls -la route.html

## 4. Kill server
kill %1
```

**Success criteria:**
- Binary compiles, runs without errors
- CLI mode produces waypoint output + saves route.html
- Web server responds on port 9122
- Web page shows Leaflet map with markers
- SSE endpoint streams events

---

## Dependencies

**Go:** Zero new imports. All APIs use stdlib `net/http`, `encoding/json`, `html/template`.

**External APIs (free, no keys):**
- Nominatim (OpenStreetMap geocoding)
- Overpass API (OSM POI query)
- OSRM (routing engine)

**CDN resources (web page):**
- Leaflet.js + Leaflet CSS
- Tailwind CSS (CDN)
- Alpine.js (CDN)

---

## Key Design Decisions

| Decision | Rationale |
|----------|-----------|
| Zero new Go dependencies | All HTTP/JSON/templates in stdlib. Simpler build, faster compile. |
| Brute-force TSP for n ≤ 5 | Thesis dataset is 2-5 waypoints. Permutations optimal and instant. No OR-Tools needed. |
| Web server with SSE streaming | Demo needs real-time LLM output. SSE is simpler than WebSockets for one-way streaming. |
| `go:embed` for HTML template | Single binary deploy. No external template files to manage. |
| Reuse existing `llm` package | Already handles OpenAI-compatible API, streaming, tool calling, reasoning_content. |
| Separate agent from FLARE's agent loop | FLARE's agent runs bash/system tools. ARAHIN's agent runs geocode/poi_search. Different tools, different loop logic. |
| No FAISS/embeddings | Overpass keyword search handles POI matching. LLM does semantic matching in ReAct loop. Not needed for thesis scope. |

---

## Verification Strategy

After each task:
1. `go build ./<package>/` — compiles clean
2. For tools: verify HTTP response parsing with printed test calls
3. For engines: basic parse tests on mock LLM responses

After all tasks:
1. Full build
2. Smoke test against live APIs
3. Web server demo

---

## Risks & Mitigations

| Risk | Mitigation |
|------|------------|
| Nominatim rate limits (1 req/s) | Built-in rate limiter + caching |
| Overpass timeout on complex queries | 25s timeout, 5-result limit, keyword simplification |
| OSRM public instance rate limits | Occasional 429. Handle gracefully. Use retry with backoff if needed. |
| DeepSeek V4 Flash reasoning_content | Already handled by existing llm package |
| SSE connection lost during streaming | Client reconnects. Server-side events carry complete state per event. |
