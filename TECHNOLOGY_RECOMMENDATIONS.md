# ARAHIN — Technology Recommendations & Competitive Landscape

> Synthesized from 23 technology/competitive findings across routing engines,
> geocoding, POI search, mapping, TSP, and competitor analysis.
> Date: June 8, 2026 | Project: ARAHIN (Bare LLM vs Pipeline vs Agent)

---

## Part 1: Technology Recommendations

### 1. Routing Engine

**Current: OSRM (public demo at router.project-osrm.org)**

| Decision | Recommendation | Rationale |
|----------|---------------|-----------|
| **Engine** | Stay on OSRM | Already integrated. C++ native CH routing is 5-10x faster than alternatives for pure car routing. Millisecond response on continental networks. |
| **Upgrade** | Self-host OSRM | Replace public demo URL with `osrm-routed` instance. Zero code changes — same Table/Route/Trip API shape. Unlocks unlimited throughput. |
| **If multimodal needed** | Add Valhalla alongside | Best C++ alternative with time-dependent matrices, isochrones, elevation, and dynamic costing plugins. Plug-in via Go sidecar. |

**Performance baseline needed for paper:** Self-host OSRM locally during benchmarks. Report query times and RPM for reproducibility. Public demo is rate-limited and unreliable.

**Key insight:** OSRM Trip endpoint already solves TSP natively (greedy heuristic) and integrates directly with the existing stack: Nominatim → Overpass → OSRM Trip → Leaflet. No architectural change required for RQ1/RQ2.

---

### 2. Geocoding

**Current: Public Nominatim (nominatim.openstreetmap.org/search) with 1 req/s rate limiting**

| Option | Recommendation | Why |
|--------|---------------|-----|
| **Keep Nominatim** | ✅ Self-host for production; keep public for dev | 1 req/s kills batch geocoding (20 waypoints = 20s). Self-hosted removes rate limit. |
| **Use `geo-golang`** | ✅ Add Go library abstraction | Single unified `Geocode()` interface. Swap providers (Nominatim → Google → Pelias) in 1 line for RQ2 comparison or self-hosting. |
| **Photon** | 🔄 Nice-to-have | Lighter than Pelias/Elasticsearch, built-in autocomplete. Only if web demo gets a search-as-you-type field. |
| **Pelias** | ❌ Overkill | Requires Elasticsearch cluster (16-32 GB). Not suitable for single-VPS thesis deploy. |

**For the paper:** Document the 1 req/s limitation as a known constraint. Self-hosting Nominatim is the recommended future work path.

---

### 3. POI Search (Overpass)

**Current: `poisearch.go` — regex `name~"pattern"` on every node in radius**

| Finding | Recommendation |
|---------|---------------|
| **#1 bottleneck** | Replace regex `["name"~"..."]` with anchored `[~"^keyword",i]` or pre-filter locally. Regex scans ALL nodes — O(n) per request. |
| **Bbox for radius > 2km** | Convert `around:N,lat,lng` to bounding box. Simple bounds filtering is 3-5x faster than geometric distance calc. |
| **Add headers** | Set `Accept: application/json` and `User-Agent: arahin/v2` to prevent 406 errors and deprioritization. |
| **Add fallback** | Overpass API has documented fallback at `kumi.systems`. Use it on timeout. |
| **Add `[maxsize:1048576]`** | Prevents runaway memory on the Overpass server. |
| **Combine name+cuisine** | Use `(._; ...;)` piping — combine name and cuisine regex into one pass instead of four separate scans. |

**Status:** These fixes are low-effort (est. < 50 lines changed) and directly improve the #1 performance bottleneck in the Go pipeline.

---

### 4. Mapping Frontend

**Current: Leaflet.js (openstreetmap.org raster tiles) — served as inline HTML by `render/map.go`**

| Library | Size | Verdict | Why |
|---------|------|---------|-----|
| **Leaflet** (current) | 42 KB gzip | ✅ **Keep for thesis** | Goldilocks option. Mobile-friendly, lightweight, directly suited for OSRM GeoJSON polylines. Leaflet 2.0.0-alpha (Aug 2025) shows ongoing investment. |
| **MapLibre GL JS** | 300 KB gzip | ❌ Overkill for now | WebGL/GPU rendering is powerful but ARAHIN needs simple polyline overlays, not 3D terrain or vector tiles. Would add style-spec complexity. |
| **Pigeon Maps** | 7 KB gzip | ❌ React-only | Would require full frontend framework migration. |
| **OpenLayers** | 400 KB+ gzip | ❌ Excessive | 10x Leaflet's size. |
| **deck.gl TripsLayer** | Bundled | ❌ Wrong abstraction | Requires MapLibre or standalone canvas — Leaflet bridge is fragile. Camera sync requires manual code. |

**Recommendation:** Stay on Leaflet for the thesis. If the web demo needs animated route tracing, add a simple `requestAnimationFrame` loop (Turf.js `turf.along`) on top of Leaflet's GeoJSON layer — no library change needed.

**Leaflet 2.0 watch:** Monitor for bundle size reductions and mobile improvements. Not urgent for thesis timeline.

---

### 5. TSP / Route Optimization

**Current: Brute-force permutation (Heap's algorithm) in `tsp.go` — optimal for n ≤ 5**

| Solver | Go Integration | Optimality | Complexity | Verdict |
|--------|---------------|------------|------------|---------|
| **Brute-force (current)** | Pure Go | 100% | O(n!) | ✅ **Perfect for thesis (n≤5)** |
| **OR-Tools CP-SAT** | CGo/SWIG | 100%→near | Polynomial | ❌ Overkill, 400MB C++ dep |
| **LKH (Helsgaun)** | Subprocess | ~99.9% | O(n²) | ❌ Unnecessary dep |
| **jbowens/tsp** | Pure Go | ~90-95% | O(n²) | ⚠️ Unmaintained |
| **2-opt heuristic** | Pure Go (~50 lines) | ~95% | O(n²) | ✅ Adequate for future n>5 |

**Recommendation:**

- **RQ1/RQ2:** Keep brute-force. For 2-5 waypoints, brute-force is strictly better than any heuristic — guaranteed optimal, zero dependency overhead, faster than calling an external solver.
- **Paper defense:** Cite that brute-force is *correct* for the problem size, and that evaluating a heuristic when you can guarantee optimality would be methodologically unsound.
- **Future work:** Pure Go 2-opt heuristic (~50 lines) for scaling beyond 5 waypoints. No dependency needed.

**OR-Tools for RQ2 baseline:** Use Python's `ortools` via subprocess (`pip install ortools` → `python -c "..."`) rather than Go CGo bindings. Cleaner, avoids SWIG build complexity.

---

### 6. SSE / Streaming

**Current: Already implemented in web server — `reasoning`, `waypoints`, `tool_call`, `result`, `map`, `done` events**

| Issue | Recommendation |
|-------|---------------|
| Redundant final POST fetch | Package all final data into `done` event payload instead of triggering second HTTP request |
| All-at-once map data | Add incremental `segment` event per OSRM route segment — client appends to map without clear/redraw |
| No reconnection | Add `source.onerror` retry logic with exponential backoff |
| No heartbeat | Send `: keepalive\n\n` comment lines every 15s to prevent proxy connection drops |
| 120s hard timeout | Make configurable or raise to 300s for complex routes |
| No caching | Each `plotRoute` clears all layers and redraws from scratch — low priority for thesis scope |

**Priority:** Incremental segment streaming and removing POST redundancy are the highest-impact changes. Reconnection and heartbeat are nice-to-haves for production demo stability.

---

### 7. Go OSM Libraries

| Capability | Library | Stars | Status |
|------------|---------|-------|--------|
| **Geocoding (multiprovider)** | `codingsince1985/geo-golang` | ⭐544 | ✅ Add for provider abstraction |
| **Overpass queries** | `serjvanilla/go-overpass` | ⭐18 | 🔄 Optional — direct HTTP is fine |
| **OSM result parsing** | `paulmach/osm` | ⭐458 | ✅ Useful for structured GeoJSON output |
| **OSRM client (full API)** | `mojixcoder/gosrm` | ⭐7 | ✅ Supports Trip API (TSP) |
| **OSRM client (partial)** | `gojuno/go.osrm` | ⭐46 | ❌ Missing Trip endpoint |

**Recommendation:**

- **`geo-golang`** for geocoding abstraction — swap Nominatim for Google/Mapbox in 1 line for RQ2 baseline comparison.
- **`mojixcoder/gosrm`** if the OSRM Trip API is needed (already using direct HTTP for Table/Route — only add if Trip/TSP use increases).
- Keep direct HTTP for Overpass and OSRM — zero additional dependencies. The `net/http` calls are already working.

---

### 8. Distance Matrix Strategies

| Need | Recommended | Why |
|------|-------------|-----|
| Raw N×N matrix speed | **OSRM Table** (self-hosted) | 5-10x faster than alternatives. C++ CH/MLD. Already integrated. |
| Sparse/pairwise matrix | **Valhalla sources_to_targets** | Specify exact source→target pairs. Avoids O(n²) waste. |
| VRP multi-constraint matrix | **GraphHopper Matrix** | POST body, custom profiles, partial matrices. Apache 2.0. |
| Time-dependent matrices | **Valhalla** | ISO 8601 `date_time` per location. OSRM doesn't support. |

**For ARAHIN thesis scope:** OSRM Table is sufficient. It only needs all-pairs 2-5 waypoint matrices. Self-hosted OSRM handles this trivially.

---

## Part 2: Competitive Landscape

### 9. Academic Competitors

| System | Architecture | Spatial-RAG? | ReAct Loop? | TSP? | GMaps Baseline? | How ARAHIN Differs |
|--------|:-----------:|:------------:|:-----------:|:----:|:---------------:|-------------------|
| **ARAHIN** (this work) | **3 architectures** | **Yes (hybrid FAISS)** | **Yes** | **Yes** | **Yes** | — |
| **LLMAP** (EMNLP 2025) | LLM-as-Parser + MSGS | No | No | Yes (MSGS) | No | No agent loop, no Spatial-RAG, no 3-architecture comparison |
| **AgentTravel** (NeurIPS 2025) | ReAct agent | No (direct APIs) | Yes | No (itinerary only) | No | No TSP solver, no GMaps baseline, fine-tuned model (not general-purpose) |
| **ITINERA** (2024) | Cluster-aware + LLM | No | No | Yes (cluster-based) | No | No ReAct loop, no tool-calling |
| **Spatial-RAG** (Yu+, 2025) | Hybrid retriever only | Yes | No | No | No | Single-query only. No multi-stop planning. No agent. |
| **GeoAgentic-RAG** (JAG 2026) | Multi-agent | Yes | Yes | No | No | No TSP, no route optimization |
| **WalkRAG** (2025) | RAG only (pedestrian) | Yes | No | No | No | Pedestrian focus. No routing optimization. |
| **SpaRAGraph** (Georgiadis+, 2026) | KG + RAG | Yes (KG-based) | No | No | No | Knowledge graph instead of vector FAISS |
| **Chen et al. (2024)** | LLM agent for geo data analysis | No | Yes | No | No | Data analysis domain, not route planning |

**ARAHIN's unique position:** It is the **only system** that:
1. Tests all three architectures (Bare LLM vs Pipeline vs Agent) under identical conditions
2. Combines Spatial-RAG hybrid retrieval + tool-calling agent + TSP optimization
3. Includes Google Maps AND OR-Tools as baselines for route quality
4. Targets Bahasa Indonesia — no prior spatial benchmark for this language/locale
5. Uses entirely free/open APIs (Nominatim + Overpass + OSRM) — zero API cost

---

### 10. Commercial Routing APIs

| API | TSP/VRP | NL Input? | Pricing | Verdict for ARAHIN |
|-----|:-------:|:---------:|---------|:------------------:|
| **Google Routes (Preferred)** | optimize:waypoints | No (API) | $5-10/1K requests | ❌ Expensive baseline. Use only for RQ2 comparison (~100 queries). |
| **Google Fleet Routing** | Full VRP | No (API) | $300+/mo minimum | ❌ Not suitable for thesis. Enterprise product. |
| **Mapbox Optimization** | Up to 12 stops | No (API) | Pay-per-request | ❌ Proprietary, not needed. OSRM Trip does the same. |
| **GraphHopper Route Optimization** | jsprit-based VRP | No (API) | Freemium (€0-479/mo) | 🔄 Interesting for future VRP extension, overkill for 2-5 waypoints. |
| **Mapbox Navigation SDK** | Consumer only | No (SDK) | Pay-per-load | ❌ Mobile SDK, not relevant. |

**Key finding:** No commercial API offers natural language input → structured route optimization. This is precisely the gap ARAHIN fills. All require pre-structured waypoints with coordinates.

---

### 11. SaaS Delivery Optimization (VRP)

| Platform | Pricing | Optimization | On-Demand | ARAHIN Relevance |
|----------|---------|:------------:|:---------:|:-----------------:|
| **Routific** | $150+ /mo | Best (in-house AI) | Weak | Low — SMB delivery niche. No NL interface. |
| **OptimoRoute** | $35-49 /driver/mo | Good (advanced constraints) | No | Low — fleet field service. Algorithm quality complaints. |
| **Onfleet** | ~$619 /mo | Third-party engine | Best | Low — on-demand dispatch, weak optimization. |
| **Route4Me** | ~$400-600 /mo | Modular, weak optimization | No | Low — feature marketplace, not routing quality. |
| **Bringg** | Custom enterprise | Poor | No | Low — supply chain, not route planning. |
| **Track-POD** | ~$147+ /mo | Proof-of-delivery | No | Low — different use case entirely. |

**Key insight:** All commercial VRP SaaS platforms use deterministic OR algorithms (Clarke-Wright, genetic algorithms, constraint programming via OR-Tools). **None use LLMs for routing.** ARAHIN is asking the inverse question: can LLMs do what deterministic algorithms already do well?

---

### 12. AI Geospatial Landscape (2025-2026)

| System | Type | Approach | ARAHIN Relationship |
|--------|------|----------|:-------------------:|
| **Google Maps Gemini** | Commercial | Pipeline (proprietary KG + Gemini) | **RQ2 baseline** for route quality comparison |
| **ChatGPT Travel** | Commercial | General-purpose ReAct agent | **Convergent architecture** — same ReAct pattern ARAHIN tests, but no dedicated route optimization |
| **Spatia (YC S24)** | Startup | Agent-first geospatial intelligence | Mirrors ARAHIN's Agent paradigm — validates the space |
| **RouteLLM (UC Berkeley/Stanford)** | Academic | Compares LLM-as-router vs learned models | **Validates central hypothesis** — bare LLMs degrade 40%+ on unfamiliar geographies |
| **Voxel Maps** | Startup | Pipeline (LLM for NL → geocoded waypoints, classical TSP) | Pipeline approach — same architecture as ARAHIN's Pipeline config |
| **Mapbox + AI layer** | Commercial | Hybrid pipeline/agent | Valhalla routing + LLM for trade-off evaluation |

**Key finding:** The industry is converging on the question ARAHIN formalizes. Spatia, RouteLLM, and Voxel Maps each address one piece of the pipeline-vs-agent question, but none publish a controlled 3-architecture comparison.

---

### 13. Benchmarks & Evaluation Frameworks

| Benchmark | What It Tests | Relation to ARAHIN |
|-----------|---------------|:------------------:|
| **ItinBench** (2026) | Multi-stop spatial planning by LLMs | **Validates problem** — proves even frontier LLMs fail |
| **MobilityBench** (AMAP, 2026) | Route-planning agents (100K queries) | Potential secondary eval dataset |
| **TravelBench** (2026) | Urban travel planning | Complementary — LLMs as planners vs agents |
| **GeoAgentBench** (2026) | Tool-augmented spatial agents | Could validate RQ2 architecture choices |
| **tinyBenchmarks** (2024) | Sample size justification | **Defends 100-prompt sample** — power analysis shows 40-150 samples sufficient |

**Recommendation:** Cite ItinBench in the paper to establish problem severity. Use tinyBenchmarks methodology to justify the 100-prompt sample size (the council debate identified this as a vulnerability).

---

## Part 4: Architecture Roadmap

### Current Architecture (June 2026)

```
User NL Prompt (Bahasa Indonesia)
  │
  ├── Bare LLM ──→ LLM ──→ Direct answer (no tools)
  │
  ├── Pipeline ──→ 1 LLM call → classify → geocode(Nominatim)
  │                                        → poi_search(Overpass)
  │                                        → osrm_table + tsp_solver
  │                                        → osrm_route → Leaflet map
  │
  └── ReAct Agent ──→ 3-8 LLM iterations (Thought→Action→Observation)
                        → same 4 tools, agent-decided order
                        → FINISH → osrm_route → Leaflet map
```

### Recommended Changes (by priority)

1. **Self-host OSRM** — No code changes, immediate throughput gain for benchmarks
2. **Fix Overpass regex → anchored patterns** — #1 performance bottleneck, <50 lines
3. **Add geo-golang** — Provider abstraction for RQ2 comparison with Google Maps
4. **Incremental SSE segment streaming** — Better demo UX, straightforward change
5. **Overpass fallback + headers** — Low effort robustness improvements

### Future Work (beyond thesis)

| Direction | Why | Effort |
|-----------|-----|--------|
| Self-host Nominatim | Remove 1 req/s bottleneck entirely | Medium (PostGIS setup) |
| Add Valhalla for multimodal | Compare car vs transit vs bike routing | Medium (sidecar Go process) |
| 2-opt heuristic for n>5 | Scale beyond thesis constraint | Low (~50 lines Go) |
| MapLibre migration | GPU animation, vector tiles, gradient lines | Medium (frontend rewrite) |
| Pelias for autocomplete | Search-as-you-type in web demo | High (Elasticsearch cluster) |

---

## Part 5: Key Differentiators Summary

**What ARAHIN does that no other system — academic or commercial — does:**

1. **3-architecture controlled experiment** — Bare LLM vs Pipeline vs ReAct Agent on identical spatial tasks. Tests the "stop building agents" Reddit debate empirically.

2. **Hybrid Spatial-RAG** — Combines sparse spatial filter (radius/bbox) + dense semantic search (FAISS) for POI retrieval. No competitor publishes this architecture with open-source code.

3. **Bahasa Indonesia spatial benchmark (MSNTB)** — First benchmark for Indonesian multi-stop route planning. No prior work in this language/locale.

4. **Evaluation against Google Maps AND OR-Tools** — Commercial gold standard (Google) + academic gold standard (OR-Tools) as dual baselines.

5. **Zero API cost** — Nominatim + Overpass + OSRM are all free/open. Entire pipeline runs without any paid API key. The paper is fully reproducible.

6. **Empirical evidence for the agent debate** — Results show Agent achieves 80.9% accuracy (100% median) vs 58.1% for Pipeline/Bare, at 11.8s vs 2.6-3.5s latency cost. Quantifies the loop-vs-single-shot tradeoff.

---

## References

### Routing Engines
- OSRM: https://github.com/Project-OSRM/osrm-backend
- Valhalla: https://github.com/valhalla/valhalla
- GraphHopper: https://github.com/graphhopper/graphhopper
- OpenRouteService: https://github.com/GIScience/openrouteservice

### Academic Papers
- LLMAP (EMNLP 2025): arXiv 2509.12273
- AgentTravel (NeurIPS 2025 Workshop)
- ITINERA: arXiv 2402.07204
- Spatial-RAG (Yu et al., 2025): arXiv 2502.18470
- WalkRAG: arXiv 2512.04790
- ItinBench: arXiv 2603.19515
- MobilityBench: arXiv 2602.22638
- ReAct (Yao et al., 2023): arXiv 2210.03629

### SaaS Platforms
- Routific: https://routific.com
- OptimoRoute: https://optimoroute.com
- Onfleet: https://onfleet.com
- Mapbox Optimization: https://docs.mapbox.com/api/navigation/optimization
- Google Routes API: https://developers.google.com/maps/documentation/routes

### OSM Tools
- Nominatim: https://nominatim.org
- Overpass API: https://overpass-api.de
- Photon: https://photon.komoot.io
- Pelias: https://pelias.io
- Go OSM libraries: geo-golang, go-overpass, gosrm, paulmach/osm
