package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"time"
)

// Entity represents a tracked object in the CV system
type Entity struct {
	ID        string
	SpawnTime int64
}

// ROI represents a physical region in a camera view
type ROI struct {
	ID         string
	Entities   map[string]*Entity
	StaffCount int
}

// CameraSim represents a single camera feed
type CameraSim struct {
	ID   string
	ROIs map[string]*ROI
}

// World holds the state of the simulation
type World struct {
	Cameras map[string]*CameraSim
	Target  string
	Client  *http.Client
}

func NewWorld(target string) *World {
	return &World{
		Cameras: make(map[string]*CameraSim),
		Target:  target,
		Client:  &http.Client{Timeout: 5 * time.Second},
	}
}

func (w *World) GetROI(camID, roiID string) *ROI {
	cam, ok := w.Cameras[camID]
	if !ok {
		cam = &CameraSim{ID: camID, ROIs: make(map[string]*ROI)}
		w.Cameras[camID] = cam
	}
	roi, ok := cam.ROIs[roiID]
	if !ok {
		roi = &ROI{ID: roiID, Entities: make(map[string]*Entity)}
		cam.ROIs[roiID] = roi
	}
	return roi
}

func (w *World) Spawn(camID, roiID string) {
	roi := w.GetROI(camID, roiID)
	id := fmt.Sprintf("trk-%d", rand.Intn(1000000))
	now := time.Now().Unix()
	roi.Entities[id] = &Entity{ID: id, SpawnTime: now}

	// Fire line_crossed event
	w.fireEvent(camID, roiID, "line_crossed", id)
}

func (w *World) Despawn(camID, roiID string, count int) {
	roi := w.GetROI(camID, roiID)
	despawned := 0
	for id := range roi.Entities {
		if despawned >= count {
			break
		}
		delete(roi.Entities, id)
		w.fireEvent(camID, roiID, "exit", id)
		despawned++
	}
}

func (w *World) SetStaff(camID, roiID string, count int) {
	roi := w.GetROI(camID, roiID)
	roi.StaffCount = count
}

func (w *World) fireEvent(camID, roiID, kind, trackID string) {
	payload := map[string]any{
		"kind":         kind,
		"eventId":      fmt.Sprintf("evt-%d", rand.Intn(1000000)),
		"emittedAt":    time.Now().Unix(),
		"locationName": "sim-store",
		"location":     "store-1",
		"roi":          roiID,
		"roiId":        roiID,
		"cameraName":   camID,
		"camera":       camID,
		"entity": map[string]any{
			"entityType": "person",
			"trackId":    trackID,
			"confidence": 0.99,
		},
	}
	b, _ := json.Marshal(payload)
	http.Post(w.Target+"/"+camID+"/event", "application/json", bytes.NewBuffer(b))
}

// Tick sends the 1Hz state payload for all cameras
func (w *World) Tick() {
	now := time.Now().Unix()
	for camID, cam := range w.Cameras {
		regions := []map[string]any{}

		for roiID, roi := range cam.ROIs {
			var maxDwell, minDwell, sumDwell float64
			minDwell = 999999
			
			entities := []map[string]any{}
			
			for _, ent := range roi.Entities {
				dwell := float64(now - ent.SpawnTime)
				if dwell > maxDwell {
					maxDwell = dwell
				}
				if dwell < minDwell {
					minDwell = dwell
				}
				sumDwell += dwell
				
				entities = append(entities, map[string]any{
					"entityType": "person",
					"trackId":    ent.ID,
					"dwellTime":  dwell,
					"confidence": 0.99,
				})
			}
			
			if len(roi.Entities) == 0 {
				minDwell = 0
			}
			avgDwell := 0.0
			if len(roi.Entities) > 0 {
				avgDwell = sumDwell / float64(len(roi.Entities))
			}

			regions = append(regions, map[string]any{
				"roi":        roiID,
				"roiId":      roiID,
				"staffCount": roi.StaffCount,
				"occupancy": map[string]any{
					"total":   len(roi.Entities),
					"person":  len(roi.Entities),
					"vehicle": 0,
				},
				"precalc": map[string]any{
					"dwell": map[string]any{
						"person": map[string]any{
							"avg": avgDwell,
							"min": minDwell,
							"max": maxDwell,
						},
					},
				},
				"entities": entities,
			})
		}

		payload := map[string]any{
			"schemaVersion": "1.0",
			"kind":          "state",
			"emittedAt":     now,
			"locationName":  "sim-store",
			"location":      "store-1",
			"cameraName":    camID,
			"camera":        camID,
			"regions":       regions,
		}

		b, _ := json.Marshal(payload)
		http.Post(w.Target+"/"+camID+"/state", "application/json", bytes.NewBuffer(b))
	}
}
