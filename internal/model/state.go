package model

type StatePayload struct {
	SchemaVersion string   `json:"schemaVersion"`
	Kind          string   `json:"kind"`
	EmittedAt     int64    `json:"emittedAt"`
	LocationName  string   `json:"locationName"`
	Location      string   `json:"location"`
	CameraName    string   `json:"cameraName"`
	Camera        string   `json:"camera"`
	Regions       []Region `json:"regions"`
}

type Region struct {
	ROI        string      `json:"roi"`
	ROIId      string      `json:"roiId"`
	Occupancy  Occupancy   `json:"occupancy"`
	Precalc    Precalc     `json:"precalc"`
	Entities   []Entity    `json:"entities"`
	StaffCount int         `json:"staffCount"`
}

type Occupancy struct {
	Total   int `json:"total"`
	Person  int `json:"person"`
	Vehicle int `json:"vehicle"`
}

type Precalc struct {
	Dwell map[string]DwellMetrics `json:"dwell"`
}

type DwellMetrics struct {
	Avg float64 `json:"avg"`
	Min float64 `json:"min"`
	Max float64 `json:"max"`
}

type Entity struct {
	EntityType string            `json:"entityType"`
	TrackId    string            `json:"trackId"`
	DwellTime  *float64          `json:"dwellTime"`
	Confidence float64           `json:"confidence"`
	Attributes map[string]string `json:"attributes"`
}
