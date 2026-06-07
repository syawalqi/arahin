package web

import (
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"math"
	"net/http"
	"strings"
	"time"

	"github.com/syawalqi/arahin/config"
	"github.com/syawalqi/arahin/engine"
	"github.com/syawalqi/arahin/llm"
	"github.com/syawalqi/arahin/render"
	"github.com/syawalqi/arahin/tools"
)

type Server struct {
	cfg    *config.Config
	server *http.Server
}

func NewServer(cfg *config.Config) *Server {
	s := &Server{cfg: cfg}
	mux := http.NewServeMux()
	mux.HandleFunc("/", s.handleIndex)
	mux.HandleFunc("/api/route", s.handleRoute)
	mux.HandleFunc("/api/route/stream", s.handleRouteStream)

	s.server = &http.Server{
		Addr:    ":" + cfg.Route.Port,
		Handler: mux,
	}
	return s
}

func (s *Server) ListenAndServe() error {
	log.Printf("[ARAHIN] Web server on :%s", s.cfg.Route.Port)
	return s.server.ListenAndServe()
}

func (s *Server) Shutdown(ctx context.Context) error {
	return s.server.Shutdown(ctx)
}

// --- Index page ---

var indexHTML = `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>ARAHIN — Route Planner</title>
<link rel="stylesheet" href="https://unpkg.com/leaflet@1.9.4/dist/leaflet.css" />
<script src="https://unpkg.com/leaflet@1.9.4/dist/leaflet.js"></script>
<script defer src="https://cdn.jsdelivr.net/npm/alpinejs@3.x.x/dist/cdn.min.js"></script>
<style>
* { margin: 0; padding: 0; box-sizing: border-box; }
body { font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif; background: #1a1a2e; color: #e0e0e0; }
.container { max-width: 900px; margin: 0 auto; padding: 16px; }
h1 { font-size: 1.5rem; color: #e94560; margin-bottom: 4px; }
.subtitle { font-size: 0.85rem; color: #888; margin-bottom: 16px; }
.input-row { display: flex; gap: 8px; margin-bottom: 12px; }
.input-row input[type="text"] { flex: 1; padding: 10px 14px; border: 1px solid #333; border-radius: 6px; background: #16213e; color: #e0e0e0; font-size: 0.95rem; outline: none; }
.input-row input[type="text"]:focus { border-color: #e94560; }
.input-row select { padding: 10px; border: 1px solid #333; border-radius: 6px; background: #16213e; color: #e0e0e0; font-size: 0.9rem; outline: none; }
.input-row button { padding: 10px 24px; background: #e94560; color: #fff; border: none; border-radius: 6px; font-size: 0.95rem; cursor: pointer; font-weight: 600; }
.input-row button:disabled { opacity: 0.5; cursor: not-allowed; }
.input-row button:hover:not(:disabled) { background: #d63851; }
#map { height: 400px; border-radius: 8px; border: 1px solid #333; margin-bottom: 12px; }
.log { background: #0f0f23; border: 1px solid #333; border-radius: 6px; padding: 10px; max-height: 300px; overflow-y: auto; font-family: 'Courier New', monospace; font-size: 0.8rem; line-height: 1.5; }
.log .tool { color: #64b5f6; }
.log .result { color: #81c784; }
.log .reasoning { color: #ffb74d; }
.log .error { color: #e57373; }
.log .info { color: #aaa; }
.result-box { margin-top: 12px; padding: 12px; background: #16213e; border-radius: 6px; border: 1px solid #333; display: none; }
.result-box.show { display: block; }
.result-box h3 { font-size: 0.9rem; color: #81c784; margin-bottom: 6px; }
.result-box table { width: 100%; font-size: 0.85rem; border-collapse: collapse; }
.result-box td { padding: 4px 8px; border-bottom: 1px solid #333; }
.result-box td:first-child { color: #aaa; width: 40%; }
.spinner { display: none; margin-left: 8px; }
.spinner.active { display: inline-block; }
@keyframes spin { to { transform: rotate(360deg); } }
.spinner:after { content: ''; display: inline-block; width: 14px; height: 14px; border: 2px solid #e94560; border-top-color: transparent; border-radius: 50%; animation: spin 0.6s linear infinite; vertical-align: middle; }
</style>
</head>
<body>
<div class="container" x-data="app()">
<h1>ARAHIN</h1>
<div class="subtitle">Multi-stop route planner &mdash; powered by LLM + OpenStreetMap</div>

<form @submit.prevent="plan()" class="input-row">
<input type="text" x-model="prompt" placeholder="dari Monas ke Taman Mini, mampir sate dekat Cawang" :disabled="busy">
<select x-model="mode" :disabled="busy">
<option value="bare">Bare LLM</option>
<option value="pipeline" selected>Pipeline</option>
<option value="agent">Agent (ReAct)</option>
</select>
<button type="submit" :disabled="!prompt.trim() || busy">
<span x-text="busy ? 'Planning...' : 'Plan Route'"></span>
<span class="spinner" :class="{active: busy}"></span>
</button>
</form>

<div id="map"></div>

<div class="log" x-ref="log">
<div class="info">Enter a route description and click Plan Route.</div>
</div>

<div class="result-box" :class="{show: result}">
<h3>Route Summary</h3>
<table>
<template x-for="(wp, i) in (result?.waypoints || [])" :key="i">
<tr><td x-text="(i===0?'Start':(i===result.waypoints.length-1?'End':'Stop '+(i+1)))+':'" style="color: #aaa;"></td><td x-text="wp.name"></td></tr>
</template>
<tr><td>Total Distance</td><td x-text="result?.total_distance_km + ' km'" style="color: #81c784;"></td></tr>
<tr><td>Total Duration</td><td x-text="result?.total_duration_min + ' min'" style="color: #81c784;"></td></tr>
</table>
</div>
</div>

<script>
var map = null;
var markers = [];
var polyline = null;

function initMap() {
if (map) return;
map = L.map('map').setView([-6.2, 106.8], 12);
L.tileLayer('https://{s}.tile.openstreetmap.org/{z}/{x}/{y}.png', {
attribution: '&copy; <a href="https://osm.org/copyright">OSM</a>',
maxZoom: 19
}).addTo(map);
}

function clearMap() {
markers.forEach(function(m) { map.removeLayer(m); });
markers = [];
if (polyline) { map.removeLayer(polyline); polyline = null; }
}

function plotRoute(wps, segs) {
clearMap();
var bounds = [];
wps.forEach(function(wp, i) {
var color = i === 0 ? '#4caf50' : (i === wps.length-1 ? '#f44336' : '#2196f3');
var icon = L.divIcon({
className: '',
html: '<div style="background:'+color+';color:#fff;width:24px;height:24px;border-radius:12px;display:flex;align-items:center;justify-content:center;font-size:12px;font-weight:bold;border:2px solid #fff;box-shadow:0 1px 3px rgba(0,0,0,0.3)">'+(i+1)+'</div>',
iconSize: [24, 24],
iconAnchor: [12, 12]
});
var m = L.marker([wp.lat, wp.lng], {icon: icon}).addTo(map);
m.bindPopup('<b>' + (i+1) + '. ' + wp.name + '</b>');
markers.push(m);
bounds.push([wp.lat, wp.lng]);
});

if (segs && segs.length > 0) {
segs.forEach(function(seg) {
if (!seg.geometry || !seg.geometry.coordinates) return;
var coords = seg.geometry.coordinates.map(function(c) { return [c[1], c[0]]; });
polyline = L.polyline(coords, {color: '#2196F3', weight: 5, opacity: 0.8}).addTo(map);
coords.forEach(function(c) { bounds.push(c); });
});
} else if (wps.length >= 2) {
var coords = wps.map(function(wp) { return [wp.lat, wp.lng]; });
polyline = L.polyline(coords, {color: '#2196F3', weight: 3, dashArray: '8,8'}).addTo(map);
}

if (bounds.length > 0) map.fitBounds(bounds, {padding: [50, 50]});
}

function app() {
	return {
		prompt: '',
		mode: 'pipeline',
		busy: false,
		result: null,
		plan: function() {
			if (!this.prompt.trim() || this.busy) return;
			this.busy = true;
			this.result = null;
			var logEl = this.$refs.log;
			logEl.innerHTML = '';

			initMap();

			var self = this;
			var encodedPrompt = encodeURIComponent(this.prompt);
			var url = '/api/route/stream?prompt=' + encodedPrompt + '&mode=' + this.mode;
			var source = new EventSource(url);
			var lastHTML = '';

			source.addEventListener('reasoning', function(evt) {
				try { var d = JSON.parse(evt.data); logEl.innerHTML += '<div class="reasoning">' + escapeHtml(d.content) + '</div>'; } catch(_) { logEl.innerHTML += '<div class="reasoning">' + escapeHtml(evt.data) + '</div>'; }
				logEl.scrollTop = logEl.scrollHeight;
			});

			source.addEventListener('waypoints', function(evt) {
				try { var d = JSON.parse(evt.data); logEl.innerHTML += '<div class="result">Waypoints: ' + escapeHtml(d.content) + '</div>'; } catch(_) { logEl.innerHTML += '<div class="result">Waypoints: ' + escapeHtml(evt.data) + '</div>'; }
				logEl.scrollTop = logEl.scrollHeight;
			});

			source.addEventListener('tool_call', function(evt) {
				try { var d = JSON.parse(evt.data); logEl.innerHTML += '<div class="tool">' + escapeHtml(d.content) + '</div>'; } catch(_) { logEl.innerHTML += '<div class="tool">' + escapeHtml(evt.data) + '</div>'; }
				logEl.scrollTop = logEl.scrollHeight;
			});

			source.addEventListener('result', function(evt) {
				try { var d = JSON.parse(evt.data); logEl.innerHTML += '<div class="result">' + escapeHtml(d.content) + '</div>'; } catch(_) { logEl.innerHTML += '<div class="result">' + escapeHtml(evt.data) + '</div>'; }
				logEl.scrollTop = logEl.scrollHeight;
			});

			source.addEventListener('error', function(evt) {
				try { var d = JSON.parse(evt.data); logEl.innerHTML += '<div class="error">Error: ' + escapeHtml(d.content) + '</div>'; } catch(_) { logEl.innerHTML += '<div class="error">Error: ' + escapeHtml(evt.data) + '</div>'; }
				logEl.scrollTop = logEl.scrollHeight;
			});

			source.addEventListener('map', function(e) {
				try {
					var data = JSON.parse(e.data);
					if (data.waypoints) {
						plotRoute(data.waypoints, data.segments);
						self.result = {waypoints: data.waypoints, total_distance_km: data.total_distance_km, total_duration_min: data.total_duration_min};
					}
				} catch(err) {
					// map HTML is too large for SSE event, use POST endpoint for final data
				}
			});

			source.addEventListener('done', function() {
				source.close();
				self.busy = false;
				// Fetch final result from POST endpoint
				fetch('/api/route', {
					method: 'POST',
					headers: {'Content-Type': 'application/json'},
					body: JSON.stringify({prompt: self.prompt, mode: self.mode})
				})
				.then(function(r) { return r.json(); })
				.then(function(data) {
					if (data.waypoints && data.waypoints.length > 0) {
						plotRoute(data.waypoints, data.segments);
						self.result = {waypoints: data.waypoints, total_distance_km: data.total_distance_km, total_duration_min: data.total_duration_min};
					}
				})
				.catch(function() {});
			});

			source.onerror = function() {
				source.close();
				self.busy = false;
				logEl.innerHTML += '<div class="error">Connection lost.</div>';
			};

			// Fetch final result after stream completes
			self._finalSource = source;
		}
	};
}

function escapeHtml(s) {
if (!s) return '';
return s.replace(/&/g,'&amp;').replace(/</g,'&lt;').replace(/>/g,'&gt;');
}
</script>
</body>
</html>`

func (s *Server) handleIndex(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	tmpl := template.Must(template.New("index").Parse(indexHTML))
	tmpl.Execute(w, nil)
}

// --- Route API ---

type routeRequest struct {
	Prompt string `json:"prompt"`
	Mode   string `json:"mode"`
}

type routeEvent struct {
	Type    string `json:"type"`
	Content string `json:"content,omitempty"`
}

type routeResponse struct {
	Events         []routeEvent         `json:"events"`
	Waypoints      []engine.Waypoint    `json:"waypoints,omitempty"`
	Segments       []*tools.RouteSegment `json:"segments,omitempty"`
	TotalDistanceKM float64              `json:"total_distance_km"`
	TotalDurationMin float64             `json:"total_duration_min"`
	MapHTML        string               `json:"map_html,omitempty"`
	Error          string               `json:"error,omitempty"`
}

func (s *Server) handleRoute(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "POST required", 405)
		return
	}

	var req routeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid JSON", 400)
		return
	}
	if strings.TrimSpace(req.Prompt) == "" {
		http.Error(w, "prompt required", 400)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 120*time.Second)
	defer cancel()

	resp := s.runRoute(ctx, req.Prompt, req.Mode, nil)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// --- SSE Stream endpoint ---

func (s *Server) handleRouteStream(w http.ResponseWriter, r *http.Request) {
	prompt := r.URL.Query().Get("prompt")
	mode := r.URL.Query().Get("mode")
	if prompt == "" {
		http.Error(w, "prompt required", 400)
		return
	}

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming not supported", 500)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	ctx, cancel := context.WithTimeout(r.Context(), 120*time.Second)
	defer cancel()

	// Run route with real-time SSE streaming via callback
	done := make(chan struct{})
	var resp routeResponse

	go func() {
		resp = s.runRoute(ctx, prompt, mode, func(typ, content string) {
			jsonData, _ := json.Marshal(map[string]string{"type": typ, "content": content})
			fmt.Fprintf(w, "event: %s\ndata: %s\n\n", typ, jsonData)
			flusher.Flush()
		})
		close(done)
	}()

	<-done

	if resp.Error != "" {
		// Error already sent via callback
	}
	if resp.MapHTML != "" {
		fmt.Fprintf(w, "event: map\ndata: %s\n\n", resp.MapHTML)
		flusher.Flush()
	}
	fmt.Fprintf(w, "event: done\ndata: {}\n\n")
	flusher.Flush()
}

// runRoute executes the full route planning pipeline.
// sendEvent is an optional callback for real-time streaming (SSE).
func (s *Server) runRoute(ctx context.Context, prompt, mode string, sendEvent func(typ, content string)) routeResponse {
	events := make([]routeEvent, 0)
	addEvent := func(typ, content string) {
		events = append(events, routeEvent{Type: typ, Content: content})
		if sendEvent != nil {
			sendEvent(typ, content)
		}
	}

	modeName := mode
	if modeName == "" {
		modeName = "pipeline"
	}
	modelName := s.cfg.Route.Model
	if modelName == "" {
		modelName = "deepseek-v4-flash"
	}

	// ARAHIN always uses OpenCode Go API (OpenAI-compatible).
	baseURL := "https://opencode.ai/zen/go/v1"
	provider := llm.NewOpenAIProvider("opencode-go", baseURL, s.cfg.APIKey)
	geocoder := tools.NewGeocoder()
	poiSearcher := tools.NewPOISearcher()

	var eng engine.RouteEngine
	switch engine.Mode(modeName) {
	case engine.ModeBare:
		eng = engine.NewBareEngine(provider, modelName, geocoder, poiSearcher)
	case engine.ModePipeline:
		eng = engine.NewPipelineEngine(provider, modelName, geocoder, poiSearcher)
	case engine.ModeAgent:
		eng = engine.NewAgentEngine(provider, modelName, geocoder, poiSearcher)
	default:
		return routeResponse{
			Error: fmt.Sprintf("unknown mode: %s", modeName),
		}
	}

	addEvent("reasoning", fmt.Sprintf("Mode: %s | Model: %s", eng.Name(), modelName))
	addEvent("reasoning", "Extracting waypoints from description...")

	waypoints, err := eng.Plan(ctx, prompt)
	if err != nil {
		return routeResponse{
			Events: append(events, routeEvent{Type: "error", Content: err.Error()}),
			Error:  err.Error(),
		}
	}

	var wpNames []string
	for _, wp := range waypoints {
		wpNames = append(wpNames, wp.Name)
	}
	addEvent("waypoints", strings.Join(wpNames, " -> "))

	// Distance matrix
	addEvent("reasoning", "Computing distance matrix via OSRM...")
	matrix, err := tools.GetDistanceMatrix(waypointsToGeocode(waypoints))
	if err != nil {
		return routeResponse{
			Events: append(events, routeEvent{Type: "error", Content: err.Error()}),
			Error:  err.Error(),
		}
	}

	// TSP
	order, _ := tools.OptimizeRoute(matrix, 0, len(waypoints)-1)
	ordered := make([]engine.Waypoint, len(order))
	orderedGeocode := make([]tools.GeocodeResult, len(order))
	for i, idx := range order {
		ordered[i] = waypoints[idx]
		orderedGeocode[i] = waypointsToGeocode([]engine.Waypoint{waypoints[idx]})[0]
	}

	addEvent("reasoning", "Fetching route segments...")

	segments := make([]*tools.RouteSegment, 0, len(ordered)-1)
	var totalDist, totalDur float64
	for i := 0; i < len(ordered)-1; i++ {
		seg, err := tools.GetRoute(orderedGeocode[i], orderedGeocode[i+1])
		if err != nil {
			addEvent("error", fmt.Sprintf("Route %s -> %s: %v", ordered[i].Name, ordered[i+1].Name, err))
			continue
		}
		segments = append(segments, seg)
		totalDist += seg.DistanceKm
		totalDur += seg.DurationMin
		addEvent("result", fmt.Sprintf("%s -> %s : %.1f km (%.0f min)", ordered[i].Name, ordered[i+1].Name, seg.DistanceKm, seg.DurationMin))
	}

	addEvent("result", fmt.Sprintf("Total: %.1f km (%.0f min)", totalDist, totalDur))

	// Render map
	html, err := render.RenderRouteHTML(render.MapConfig{
		Waypoints: ordered,
		Segments:  segments,
		TotalKM:   totalDist,
		TotalMin:  totalDur,
	})
	if err != nil {
		addEvent("error", "Map render failed: "+err.Error())
	}

	return routeResponse{
		Events:          events,
		Waypoints:       ordered,
		Segments:        segments,
		TotalDistanceKM:  math.Round(totalDist*10) / 10,
		TotalDurationMin: math.Round(totalDur),
		MapHTML:         html,
	}
}

func waypointsToGeocode(wps []engine.Waypoint) []tools.GeocodeResult {
	res := make([]tools.GeocodeResult, len(wps))
	for i, wp := range wps {
		res[i] = tools.GeocodeResult{Name: wp.Name, Lat: wp.Lat, Lng: wp.Lng, DisplayName: wp.DisplayName}
	}
	return res
}
