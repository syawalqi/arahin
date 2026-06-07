package tools

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type POI struct {
	Name     string  `json:"name"`
	Lat      float64 `json:"lat"`
	Lng      float64 `json:"lng"`
	Category string  `json:"category"`
}

type POISearcher struct {
	client *http.Client
}

var categoryMap = map[string]string{
	"masjid": "mosque", "musholla": "mosque", "musala": "mosque",
	"restoran": "restaurant", "rumah makan": "restaurant",
	"kafe": "cafe", "angkringan": "cafe",
	"atm": "atm", "bank": "bank",
	"rumah sakit": "hospital", "klinik": "clinic", "apotek": "pharmacy",
	"mall": "mall", "supermarket": "supermarket",
	"pom bensin": "fuel", "spbu": "fuel",
}

var stopWords = map[string]bool{
	"di": true, "ke": true, "dari": true, "dan": true, "untuk": true,
	"yang": true, "saya": true, "kami": true, "kita": true, "anda": true,
	"ini": true, "itu": true, "adalah": true, "bahwa": true, "karena": true,
	"jika": true, "maka": true, "lalu": true, "kemudian": true, "setelah": true,
	"sebelum": true, "pada": true, "oleh": true, "dengan": true, "serta": true,
	"juga": true, "atau": true, "tidak": true, "bisa": true, "dapat": true,
	"harus": true, "telah": true, "sudah": true, "akan": true,
	"mau": true, "makan": true, "minum": true, "beli": true, "cari": true,
	"dekat": true, "tolong": true, "sama": true, "seperti": true, "secara": true,
	"saja": true, "a": true, "an": true, "the": true, "is": true, "it": true,
	"at": true, "on": true, "in": true, "of": true, "to": true, "for": true,
	"and": true, "or": true, "with": true, "near": true, "from": true, "by": true,
	"want": true, "need": true, "can": true, "please": true, "some": true, "any": true,
}

func NewPOISearcher() *POISearcher {
	return &POISearcher{
		client: &http.Client{Timeout: 25 * time.Second},
	}
}

// Search searches for POIs near coordinates.
// If query matches a known category, uses amenity tag search (category mode).
// Otherwise extracts keywords and searches OSM names via regex (keyword mode).
func (s *POISearcher) Search(query string, lat, lng float64, radiusKm int) []POI {
	if radiusKm <= 0 {
		radiusKm = 3
	}
	queryLower := strings.ToLower(strings.TrimSpace(query))

	var overpassQuery string
	if cat, ok := categoryMap[queryLower]; ok {
		// Category mode
		overpassQuery = fmt.Sprintf(
			`[out:json][timeout:25];node["amenity"="%s"](around:%d,%f,%f);out 5;`,
			cat, radiusKm*1000, lat, lng,
		)
	} else {
		// Keyword mode — search both node and way for name regex + amenity
		keywords := extractKeywords(query)
		if len(keywords) == 0 {
			// Fallback: dump nearby amenities
			overpassQuery = fmt.Sprintf(
				`[out:json][timeout:25];(node["amenity"](around:%d,%f,%f);way["amenity"](around:%d,%f,%f););out center 5;`,
				radiusKm*1000, lat, lng, radiusKm*1000, lat, lng,
			)
		} else {
			patterns := strings.Join(keywords, "|")
			// Search nodes and ways by name OR by amenity+cuisine match
			overpassQuery = fmt.Sprintf(
				`[out:json][timeout:25];(`+
					`node["name"~"%s",i](around:%d,%f,%f);`+
					`way["name"~"%s",i](around:%d,%f,%f);`+
					`node["cuisine"~"%s",i](around:%d,%f,%f);`+
					`way["cuisine"~"%s",i](around:%d,%f,%f);`+
				`);out center 5;`,
				patterns, radiusKm*1000, lat, lng,
				patterns, radiusKm*1000, lat, lng,
				patterns, radiusKm*1000, lat, lng,
				patterns, radiusKm*1000, lat, lng,
			)
		}
	}

	resp, err := s.client.Post(
		"https://overpass-api.de/api/interpreter",
		"application/x-www-form-urlencoded",
		strings.NewReader(url.Values{"data": {overpassQuery}}.Encode()),
	)
	if err != nil {
		return nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil
	}

	var result struct {
		Elements []struct {
			Lat    float64          `json:"lat"`
			Lon    float64          `json:"lon"`
			Center struct {
				Lat float64 `json:"lat"`
				Lon float64 `json:"lon"`
			} `json:"center"`
			Tags map[string]string `json:"tags"`
			Type string            `json:"type"`
		} `json:"elements"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil
	}

	pois := make([]POI, 0)
	seen := make(map[string]bool)

	for _, el := range result.Elements {
		name := el.Tags["name"]
		if name == "" {
			continue
		}
		// Use center for way/relation, direct lat/lon for nodes
		lat, lng := el.Lat, el.Lon
		if el.Type == "way" || el.Type == "relation" {
			lat, lng = el.Center.Lat, el.Center.Lon
		}
		// Dedup by ~100m grid
		key := fmt.Sprintf("%.3f,%.3f", lat, lng)
		if seen[key] {
			continue
		}
		seen[key] = true
		pois = append(pois, POI{
			Name:     name,
			Lat:      lat,
			Lng:      lng,
			Category: el.Tags["amenity"],
		})
		if len(pois) >= 5 {
			break
		}
	}

	return pois
}

func extractKeywords(query string) []string {
	words := strings.Fields(strings.ToLower(query))
	kw := make([]string, 0)
	for _, w := range words {
		if !stopWords[w] && len(w) > 1 {
			kw = append(kw, w)
		}
	}
	if len(kw) > 3 {
		kw = kw[:3]
	}
	return kw
}
