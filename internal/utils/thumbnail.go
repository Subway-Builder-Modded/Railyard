package utils

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"math"
	"net/http"
	"strings"
	"time"

	"railyard/internal/types"

	svg "github.com/ajstarks/svgo"
	"github.com/paulmach/orb"
	"github.com/paulmach/orb/encoding/mvt"
)

type projection struct {
	minTileX, minTileY int
	scale              float64
	offsetX, offsetY   float64
}

var thumbnailHTTPClient = &http.Client{Timeout: types.RequestTimeout}

func newProjection(minTileX, minTileY, maxTileX, maxTileY int) projection {
	tileGridWidth := float64((maxTileX - minTileX + 1) * 4096)
	tileGridHeight := float64((maxTileY - minTileY + 1) * 4096)

	scaleX := 800.0 / tileGridWidth
	scaleY := 800.0 / tileGridHeight
	scale := math.Min(scaleX, scaleY)

	scaledWidth := tileGridWidth * scale
	scaledHeight := tileGridHeight * scale

	return projection{
		minTileX: minTileX,
		minTileY: minTileY,
		scale:    scale,
		offsetX:  (800.0 - scaledWidth) / 2,
		offsetY:  (800.0 - scaledHeight) / 2,
	}
}

func (p *projection) project(tileX, tileY int, pixelX, pixelY float64) (float64, float64) {
	absoluteX := float64((tileX-p.minTileX)*4096) + pixelX
	absoluteY := float64((tileY-p.minTileY)*4096) + pixelY
	return absoluteX*p.scale + p.offsetX, absoluteY*p.scale + p.offsetY
}

func lon2tile(lon float64, zoom int) int {
	return int(math.Floor((lon + 180.0) / 360.0 * math.Pow(2, float64(zoom))))
}

func lat2tile(lat float64, zoom int) int {
	return int(math.Floor((1.0 - math.Log(math.Tan(lat*math.Pi/180.0)+1.0/math.Cos(lat*math.Pi/180.0))/math.Pi) / 2.0 * math.Pow(2, float64(zoom))))
}

func fetchWithRetry(url string, retries int, delay time.Duration) ([]byte, error) {
	var lastErr error
	for attempt := 1; attempt <= retries; attempt++ {
		resp, err := thumbnailHTTPClient.Get(url)
		if err != nil {
			lastErr = err
			time.Sleep(delay)
			delay *= 2
			continue
		}
		body, readErr := io.ReadAll(resp.Body)
		resp.Body.Close()
		if resp.StatusCode == http.StatusOK && readErr == nil {
			return body, nil
		}
		if readErr != nil {
			lastErr = readErr
		} else {
			lastErr = fmt.Errorf("HTTP %d for %s", resp.StatusCode, url)
		}
		time.Sleep(delay)
		delay *= 2
	}
	return nil, fmt.Errorf("failed to fetch %s after %d attempts: %w", url, retries, lastErr)
}

func GenerateThumbnail(cityCode string, cityConfig types.ConfigData, port int) (string, error) {
	bboxToUse := cityConfig.ThumbnailBbox
	if bboxToUse == nil {
		bboxToUse = cityConfig.Bbox
	}
	if bboxToUse == nil {
		// Derive bbox from initialViewState
		lat := cityConfig.InitialViewState.Latitude
		lng := cityConfig.InitialViewState.Longitude
		zoom := cityConfig.InitialViewState.Zoom
		if lat == 0 && lng == 0 {
			return "", fmt.Errorf("no bounding box or initial view state found for city %s", cityCode)
		}
		// Approximate span based on zoom level
		latSpan := 360.0 / math.Pow(2, zoom)
		lngSpan := 360.0 / math.Pow(2, zoom)
		derived := [4]float64{lng - lngSpan, lat - latSpan, lng + lngSpan, lat + latSpan}
		bboxToUse = &derived
	}

	minXTile := lon2tile(bboxToUse[0], 12)
	maxXTile := lon2tile(bboxToUse[2], 12)
	maxYTile := lat2tile(bboxToUse[1], 12)
	minYTile := lat2tile(bboxToUse[3], 12)

	proj := newProjection(minXTile, minYTile, maxXTile, maxYTile)

	type tileData struct {
		X, Y int
		Data []byte
	}
	var allTiles []tileData

	for x := minXTile; x <= maxXTile; x++ {
		for y := minYTile; y <= maxYTile; y++ {
			tileURL := fmt.Sprintf("http://127.0.0.1:%d/%s/12/%d/%d.mvt", port, cityCode, x, y)
			buffer, err := fetchWithRetry(tileURL, 5, 200*time.Millisecond)
			if err != nil {
				log.Printf("Error fetching tile: %v\n", err)
				continue
			}
			allTiles = append(allTiles, tileData{X: x, Y: y, Data: buffer})
		}
	}

	var svgBuffer bytes.Buffer
	svgCanvas := svg.New(&svgBuffer)
	svgCanvas.Start(800, 800)
	svgCanvas.Rect(0, 0, 800, 800, "fill:rgb(242,231,211)")

	for _, tile := range allTiles {
		layers, err := mvt.Unmarshal(tile.Data)
		if err != nil {
			log.Printf("failed to parse tile data: %v\n", err)
			continue
		}

		var waterLayer *mvt.Layer
		for _, layer := range layers {
			if layer.Name == "water" {
				waterLayer = layer
				break
			}
		}

		if waterLayer == nil {
			continue
		}

		for _, feature := range waterLayer.Features {
			renderGeometry(svgCanvas, &proj, tile.X, tile.Y, feature.Geometry)
		}
	}

	svgCanvas.End()
	return svgBuffer.String(), nil
}

func renderGeometry(canvas *svg.SVG, proj *projection, tileX, tileY int, geometry orb.Geometry) {
	switch g := geometry.(type) {
	case orb.Point:
		x, y := proj.project(tileX, tileY, g.X(), g.Y())
		p := fmt.Sprintf("M %f,%f m -2,0 a 2,2 0 1,0 4,0 a 2,2 0 1,0 -4,0", x, y)
		canvas.Path(p, "fill:rgb(159,201,234)")
	case orb.LineString:
		if p := buildLineStringPath(proj, tileX, tileY, g); p != "" {
			canvas.Path(p, "stroke:rgb(159,201,234);fill:none")
		}
	case orb.Polygon:
		for _, ring := range g {
			if p := buildRingPath(proj, tileX, tileY, ring); p != "" {
				canvas.Path(p, "fill:rgb(159,201,234);stroke:none")
			}
		}
	case orb.MultiPoint:
		for _, pt := range g {
			x, y := proj.project(tileX, tileY, pt.X(), pt.Y())
			p := fmt.Sprintf("M %f,%f m -2,0 a 2,2 0 1,0 4,0 a 2,2 0 1,0 -4,0", x, y)
			canvas.Path(p, "fill:rgb(159,201,234)")
		}
	case orb.MultiLineString:
		for _, ls := range g {
			if p := buildLineStringPath(proj, tileX, tileY, ls); p != "" {
				canvas.Path(p, "stroke:rgb(159,201,234);fill:none")
			}
		}
	case orb.MultiPolygon:
		for _, polygon := range g {
			for _, ring := range polygon {
				if p := buildRingPath(proj, tileX, tileY, ring); p != "" {
					canvas.Path(p, "fill:rgb(159,201,234);stroke:none")
				}
			}
		}
	}
}

func buildLineStringPath(proj *projection, tileX, tileY int, ls orb.LineString) string {
	if len(ls) == 0 {
		return ""
	}
	var b strings.Builder
	x0, y0 := proj.project(tileX, tileY, ls[0].X(), ls[0].Y())
	fmt.Fprintf(&b, "M %f %f", x0, y0)
	for _, pt := range ls[1:] {
		x, y := proj.project(tileX, tileY, pt.X(), pt.Y())
		fmt.Fprintf(&b, " L %f %f", x, y)
	}
	return b.String()
}

func buildRingPath(proj *projection, tileX, tileY int, ring orb.Ring) string {
	if len(ring) == 0 {
		return ""
	}
	var b strings.Builder
	x0, y0 := proj.project(tileX, tileY, ring[0].X(), ring[0].Y())
	fmt.Fprintf(&b, "M %f %f", x0, y0)
	for _, pt := range ring[1:] {
		x, y := proj.project(tileX, tileY, pt.X(), pt.Y())
		fmt.Fprintf(&b, " L %f %f", x, y)
	}
	b.WriteString(" Z")
	return b.String()
}
