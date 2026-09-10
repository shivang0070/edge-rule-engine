package model

type EventPayload struct {
	Kind         string  `json:"kind"`
	EventId      string  `json:"eventId"`
	EmittedAt    int64   `json:"emittedAt"`
	LocationName string  `json:"locationName"`
	Location     string  `json:"location"`
	ROI          string  `json:"roi"`
	ROIId        string  `json:"roiId"`
	CameraName   string  `json:"cameraName"`
	Camera       string  `json:"camera"`
	Entity       *Entity `json:"entity"`
}
