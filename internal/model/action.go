package model

type ActionRecord struct {
	RuleID      string         `json:"ruleId"`
	RuleName    string         `json:"ruleName"`
	TriggeredAt int64          `json:"triggeredAt"`
	ActionType  string         `json:"action"`
	CameraId    string         `json:"cameraId"`
	ROIId       string         `json:"roiId,omitempty"`
	Context     map[string]any `json:"context,omitempty"`
}

type RuleExecution struct {
	ID          int64  `json:"id"`
	RuleID      string `json:"ruleId"`
	TriggeredAt int64  `json:"triggeredAt"`
	Status      string `json:"status"` // "success", "error"
	ActionType  string `json:"actionType"`
	Context     string `json:"context"` // JSON string
	Error       string `json:"error,omitempty"`
}
