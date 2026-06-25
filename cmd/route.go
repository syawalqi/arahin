package cmd

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/syawalqi/arahin/config"
	"github.com/syawalqi/arahin/engine"
	"github.com/syawalqi/arahin/llm"
	"github.com/syawalqi/arahin/render"
	"github.com/syawalqi/arahin/tools"
)

// Route runs the full route planning pipeline on a prompt and prints results.
func Route(cfg *config.Config, prompt string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	// Set up LLM provider
	modeName := cfg.Route.Mode
	if modeName == "" {
		modeName = "pipeline"
	}
	modelName := cfg.Route.Model
	if modelName == "" {
		modelName = "deepseek-v4-flash"
	}

	// ARAHIN always uses OpenCode Go API (OpenAI-compatible).
	// The model name is bare (e.g. "deepseek-v4-flash") with no provider prefix.
	baseURL := "https://opencode.ai/zen/go/v1"
	provider := llm.NewOpenAIProvider("opencode-go", baseURL, cfg.APIKey)
	geocoder := tools.NewGeocoder()
	poiSearcher := tools.NewPOISearcher()

	// Select engine
	var eng engine.RouteEngine
	switch engine.Mode(modeName) {
	case engine.ModeBare:
		eng = engine.NewBareEngine(provider, modelName, geocoder, poiSearcher)
	case engine.ModePipeline:
		eng = engine.NewPipelineEngine(provider, modelName, geocoder, poiSearcher)
	case engine.ModeAgent:
		eng = engine.NewAgentEngine(provider, modelName, geocoder, poiSearcher)
	default:
		return fmt.Errorf("unknown mode: %s (use: bare, pipeline, agent)", modeName)
	}

	fmt.Fprintf(os.Stderr, "[ARAHIN] Mode: %s | Model: %s\n", eng.Name(), modelName)
	fmt.Fprintf(os.Stderr, "[ARAHIN] Extracting waypoints...\n")

	start := time.Now()
	waypoints, err := eng.Plan(ctx, prompt)
	if err != nil {
		return fmt.Errorf("plan: %w", err)
	}
	fmt.Fprintf(os.Stderr, "[ARAHIN] Got %d waypoints in %v\n", len(waypoints), time.Since(start))

	// Print waypoints
	for i, wp := range waypoints {
		marker := "•"
		if i == 0 {
			marker = "START"
		} else if i == len(waypoints)-1 {
			marker = "END"
		}
		fmt.Printf("  %s %s (%.4f, %.4f)\n", marker, wp.Name, wp.Lat, wp.Lng)
	}

	// Get distance matrix
	fmt.Fprintf(os.Stderr, "[ARAHIN] Computing distance matrix...\n")
	matrix, err := tools.GetDistanceMatrix(waypointsToGeocode(waypoints))
	if err != nil {
		return fmt.Errorf("distance matrix: %w", err)
	}

	// Optimize with TSP
	order, cost := tools.OptimizeRoute(matrix, 0, len(waypoints)-1)
	fmt.Fprintf(os.Stderr, "[ARAHIN] TSP optimized: %v (%.0f sec total)\n", order, cost)

	// Reorder waypoints
	ordered := make([]engine.Waypoint, len(order))
	orderedGeocode := make([]tools.GeocodeResult, len(order))
	for i, idx := range order {
		ordered[i] = waypoints[idx]
		orderedGeocode[i] = waypointsToGeocode([]engine.Waypoint{waypoints[idx]})[0]
	}

	// Get route segments
	fmt.Fprintf(os.Stderr, "[ARAHIN] Fetching route segments...\n")
	segments := make([]*tools.RouteSegment, 0, len(ordered)-1)
	var totalDist, totalDur float64
	for i := 0; i < len(ordered)-1; i++ {
		seg, err := tools.GetRoute(orderedGeocode[i], orderedGeocode[i+1])
		if err != nil {
			fmt.Fprintf(os.Stderr, "  [WARN] route %s -> %s: %v\n", ordered[i].Name, ordered[i+1].Name, err)
			segments = append(segments, nil)
			continue
		}
		segments = append(segments, seg)
		totalDist += seg.DistanceKm
		totalDur += seg.DurationMin
	}

	// Print route table
	fmt.Println()
	fmt.Println("Route (after TSP optimization):")
	for i := 0; i < len(ordered)-1; i++ {
		seg := segments[i]
		if seg != nil {
			fmt.Printf("  %s -> %s : %.1f km (%.0f min)\n",
				ordered[i].Name, ordered[i+1].Name, seg.DistanceKm, seg.DurationMin)
		} else {
			fmt.Printf("  %s -> %s : [route unavailable]\n", ordered[i].Name, ordered[i+1].Name)
		}
	}
	fmt.Printf("\nTotal: %.1f km (%.0f min)\n", totalDist, totalDur)

	// Render map
	html, err := render.RenderRouteHTML(render.MapConfig{
		Waypoints: ordered,
		Segments:  segments,
		TotalKM:   totalDist,
		TotalMin:  totalDur,
	})
	if err != nil {
		return fmt.Errorf("render map: %w", err)
	}

	mapPath := "route.html"
	if err := os.WriteFile(mapPath, []byte(html), 0644); err != nil {
		return fmt.Errorf("write map: %w", err)
	}
	fmt.Printf("\nMap saved: %s\n", mapPath)

	return nil
}

func waypointsToGeocode(wps []engine.Waypoint) []tools.GeocodeResult {
	res := make([]tools.GeocodeResult, len(wps))
	for i, wp := range wps {
		res[i] = tools.GeocodeResult{Name: wp.Name, Lat: wp.Lat, Lng: wp.Lng, DisplayName: wp.DisplayName}
	}
	return res
}
