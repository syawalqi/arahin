package engine

import (
	"context"

	"github.com/syawalqi/arahin/tools"
)

// Mode represents the route planning mode.
type Mode string

const (
	ModeBare     Mode = "bare"
	ModePipeline Mode = "pipeline"
	ModeAgent    Mode = "agent"
)

// Waypoint represents a resolved geographic point.
type Waypoint struct {
	Name        string  `json:"name"`
	Lat         float64 `json:"lat"`
	Lng         float64 `json:"lng"`
	DisplayName string  `json:"display_name,omitempty"`
}

// RouteResult holds the complete route planning result.
type RouteResult struct {
	Waypoints   []Waypoint          `json:"waypoints"`
	Order       []int               `json:"order"`
	Segments    []*tools.RouteSegment `json:"segments,omitempty"`
	TotalDistKM float64             `json:"total_distance_km"`
	TotalDurMin float64             `json:"total_duration_min"`
	Error       string              `json:"error,omitempty"`
	MapHTML     string              `json:"map_html,omitempty"`
}

// ProgressFunc is an optional callback for real-time progress events.
type ProgressFunc func(typ, msg string)

// RouteEngine is the interface all 3 modes implement.
type RouteEngine interface {
	Plan(ctx context.Context, prompt string) ([]Waypoint, error)
	Name() Mode
}
