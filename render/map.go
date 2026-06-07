package render

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html/template"
	"math"

	"github.com/syawalqi/arahin/engine"
	"github.com/syawalqi/arahin/tools"
)

// MapConfig holds data for rendering the Leaflet route map.
type MapConfig struct {
	Waypoints []engine.Waypoint
	Segments  []*tools.RouteSegment
	TotalKM   float64
	TotalMin  float64
}

// RenderRouteHTML generates a self-contained Leaflet HTML page as a string.
func RenderRouteHTML(cfg MapConfig) (string, error) {
	wpJSON, err := json.Marshal(cfg.Waypoints)
	if err != nil {
		return "", fmt.Errorf("marshal waypoints: %w", err)
	}

	segJSON, err := json.Marshal(cfg.Segments)
	if err != nil {
		return "", fmt.Errorf("marshal segments: %w", err)
	}

	// Compute center for initial map view
	centerLat, centerLng := 0.0, 0.0
	if len(cfg.Waypoints) > 0 {
		for _, wp := range cfg.Waypoints {
			centerLat += wp.Lat
			centerLng += wp.Lng
		}
		n := float64(len(cfg.Waypoints))
		centerLat /= n
		centerLng /= n
	}

	// Build waypoint list HTML
	type wpInfo struct {
		Index int
		Name  string
		Color string
		Icon  string
		Lat   string
		Lng   string
	}
	wpList := make([]wpInfo, len(cfg.Waypoints))
	colors := []string{"green", "blue", "blue", "blue", "red"}
	icons := []string{"play", "info-sign", "info-sign", "info-sign", "stop"}
	for i, wp := range cfg.Waypoints {
		c := colors[i]
		ic := icons[i]
		if i >= len(colors) {
			c = "blue"
			ic = "info-sign"
		}
		wpList[i] = wpInfo{
			Index: i + 1,
			Name:  wp.Name,
			Color: c,
			Icon:  ic,
			Lat:   fmt.Sprintf("%.6f", wp.Lat),
			Lng:   fmt.Sprintf("%.6f", wp.Lng),
		}
	}

	tmpl := template.Must(template.New("map").Parse(mapTemplate))

	var buf bytes.Buffer
	err = tmpl.Execute(&buf, map[string]interface{}{
		"WaypointsJSON": template.JS(string(wpJSON)),
		"SegmentsJSON":  template.JS(string(segJSON)),
		"TotalKM":       fmt.Sprintf("%.1f", cfg.TotalKM),
		"TotalMin":      fmt.Sprintf("%.0f", math.Ceil(cfg.TotalMin)),
		"CenterLat":     fmt.Sprintf("%.6f", centerLat),
		"CenterLng":     fmt.Sprintf("%.6f", centerLng),
		"WaypointCount": len(cfg.Waypoints),
		"Waypoints":     wpList,
	})
	if err != nil {
		return "", fmt.Errorf("render template: %w", err)
	}

	return buf.String(), nil
}

const mapTemplate = `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>ARAHIN — Route Map</title>
<link rel="stylesheet" href="https://unpkg.com/leaflet@1.9.4/dist/leaflet.css" />
<script src="https://unpkg.com/leaflet@1.9.4/dist/leaflet.js"></script>
<style>
* { margin: 0; padding: 0; box-sizing: border-box; }
body { font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif; background: #1a1a2e; color: #e0e0e0; }
.container { max-width: 1200px; margin: 0 auto; padding: 16px; }
h1 { font-size: 1.5rem; margin-bottom: 8px; color: #e94560; }
.info { font-size: 0.85rem; color: #aaa; margin-bottom: 12px; }
.info strong { color: #fff; }
#map { height: 500px; border-radius: 8px; border: 1px solid #333; }
.waypoint-list { margin-top: 12px; display: flex; flex-wrap: wrap; gap: 6px; }
.waypoint-list .wp { font-size: 0.8rem; background: #16213e; padding: 4px 10px; border-radius: 4px; border-left: 3px solid #555; }
.waypoint-list .wp.start { border-color: #4caf50; }
.waypoint-list .wp.mid { border-color: #2196f3; }
.waypoint-list .wp.end { border-color: #f44336; }
</style>
</head>
<body>
<div class="container">
<h1>ARAHIN — Route</h1>
<div class="info">
Total: <strong>{{.TotalKM}} km</strong> (~{{.TotalMin}} min) &middot;
<strong>{{.WaypointCount}}</strong> waypoints
</div>
<div id="map"></div>
<div class="waypoint-list">
{{range .Waypoints}}
<div class="wp {{if eq .Color "green"}}start{{else if eq .Color "red"}}end{{else}}mid{{end}}">
{{.Index}}. {{.Name}}
</div>
{{end}}
</div>
</div>
<script>
(function() {
var waypoints = {{.WaypointsJSON}};
var segments = {{.SegmentsJSON}};

var map = L.map('map').setView([{{.CenterLat}}, {{.CenterLng}}], 13);
L.tileLayer('https://{s}.tile.openstreetmap.org/{z}/{x}/{y}.png', {
attribution: '&copy; <a href="https://osm.org/copyright">OSM</a>',
maxZoom: 19
}).addTo(map);

var bounds = [];
var colors = ['green', 'blue', 'blue', 'blue', 'red'];
var icons = ['play', 'info-sign', 'info-sign', 'info-sign', 'stop'];

waypoints.forEach(function(wp, i) {
var c = colors[i] || 'blue';
var ic = icons[i] || 'info-sign';
L.marker([wp.lat, wp.lng], {
icon: L.divIcon({
className: 'custom-marker',
html: '<div style="background:'+c+';color:#fff;width:24px;height:24px;border-radius:12px;display:flex;align-items:center;justify-content:center;font-size:12px;font-weight:bold;border:2px solid #fff;box-shadow:0 1px 3px rgba(0,0,0,0.3)">'+(i+1)+'</div>',
iconSize: [24, 24],
iconAnchor: [12, 12]
})
}).bindPopup('<b>' + (i+1) + '. ' + wp.name + '</b>').addTo(map);
bounds.push([wp.lat, wp.lng]);
});

if (segments && segments.length > 0) {
segments.forEach(function(seg) {
if (!seg.geometry || !seg.geometry.coordinates) return;
var coords = seg.geometry.coordinates.map(function(c) { return [c[1], c[0]]; });
var polyline = L.polyline(coords, {color: '#2196F3', weight: 5, opacity: 0.8}).addTo(map);
polyline.bindPopup(seg.distance_km.toFixed(1) + ' km · ' + seg.duration_min.toFixed(0) + ' min');
coords.forEach(function(c) { bounds.push(c); });
});
} else if (waypoints.length >= 2) {
var coords = waypoints.map(function(wp) { return [wp.lat, wp.lng]; });
L.polyline(coords, {color: '#2196F3', weight: 3, dashArray: '8,8'}).addTo(map);
}

if (bounds.length > 0) {
map.fitBounds(bounds, {padding: [50, 50]});
}
})();
</script>
</body>
</html>`
