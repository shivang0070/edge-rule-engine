package model

import "errors"

// DataSourceKind identifies the type of data stream
type DataSourceKind string

const (
	SourceState       DataSourceKind = "state"
	SourceEvent       DataSourceKind = "event"
	TriggerModeRising                = "rising"
	TriggerModeEvery                 = "every"
)

type Rule struct {
	ID           string         `json:"id"`
	Name         string         `json:"name"`
	Version      int            `json:"version"` // V2
	Enabled      bool           `json:"enabled"`
	
	// V2: explicit dependency declarations
	Dependencies []Dependency   `json:"dependencies,omitempty"`
	
	// V1 compat
	Scope        *RuleScope     `json:"scope,omitempty"`
	Source       string         `json:"source,omitempty"`
	
	Condition    Condition      `json:"condition"`
	Patterns     []EventPattern `json:"patterns,omitempty"`
	Correlation  *Correlation   `json:"correlation,omitempty"`
	Window       *WindowConfig  `json:"window,omitempty"`
	Sustain      *SustainConfig `json:"sustain,omitempty"`
	Trigger      TriggerConfig  `json:"trigger"`
	Action       ActionConfig   `json:"action"`
	CreatedAt    int64          `json:"createdAt"`
	UpdatedAt    int64          `json:"updatedAt"`
}

type Dependency struct {
	ID           string         `json:"id"`
	Alias        string         `json:"alias"`
	CameraId     string         `json:"cameraId"`
	ROIId        string         `json:"roiId,omitempty"`
	Source       DataSourceKind `json:"source"`
	EventKind    string         `json:"eventKind,omitempty"`
	Window       *WindowSpec    `json:"window,omitempty"`
	Fields       []string       `json:"fields,omitempty"`
	MaxStaleness string         `json:"maxStaleness,omitempty"`
}

type WindowSpec struct {
	Duration    string `json:"duration"`
	Aggregation string `json:"aggregation"`
	GroupBy     string `json:"groupBy,omitempty"`
	Filter      string `json:"filter,omitempty"`
}

type EventPattern struct {
	Steps    []PatternStep `json:"steps"`
	Within   string        `json:"within"`
	Operator string        `json:"operator"`
}

type PatternStep struct {
	DepAlias   string `json:"depAlias"`
	Expression string `json:"expression,omitempty"`
	EventKind  string `json:"eventKind,omitempty"`
}

type Correlation struct {
	Key    string `json:"key"`
	Across string `json:"across"`
	Within string `json:"within"`
}

type RuleScope struct {
	CameraId  string `json:"cameraId"`
	ROIId     string `json:"roiId,omitempty"`
	EventKind string `json:"eventKind,omitempty"` // for event rules: filter by kind
}

type Condition struct {
	Expression string `json:"expression"`
}

type WindowConfig struct {
	DurationSeconds int `json:"durationSeconds"`
}

type SustainConfig struct {
	DurationSeconds int `json:"durationSeconds"`
}

type TriggerConfig struct {
	Mode            string `json:"mode"` // "rising" (default) or "every"
	CooldownSeconds int    `json:"cooldownSeconds"`
}

type ActionConfig struct {
	Type string `json:"type"`
}

func (r *Rule) ConvertV1() {
	if r.Scope != nil && len(r.Dependencies) == 0 {
		r.Dependencies = []Dependency{{
			ID:        "dep-default",
			Alias:     "default",
			CameraId:  r.Scope.CameraId,
			ROIId:     r.Scope.ROIId,
			Source:    DataSourceKind(r.Source),
			EventKind: r.Scope.EventKind,
		}}
	}
}

func (r *Rule) Validate() error {
	if r.ID == "" {
		return errors.New("rule ID is required")
	}
	if r.Name == "" {
		return errors.New("rule Name is required")
	}
	
	// Pre-convert to simplify validation
	r.ConvertV1()
	
	if len(r.Dependencies) == 0 {
		return errors.New("rule must have at least one dependency (or scope for V1)")
	}
	
	for _, dep := range r.Dependencies {
		if dep.CameraId == "" {
			return errors.New("dependency CameraId is required")
		}
		if dep.Source != SourceState && dep.Source != SourceEvent {
			return errors.New("dependency Source must be either state or event")
		}
	}

	if r.Condition.Expression == "" {
		return errors.New("rule Condition.Expression is required")
	}
	if r.Action.Type == "" {
		return errors.New("rule Action.Type is required")
	}
	if r.Trigger.Mode != "" && r.Trigger.Mode != TriggerModeRising && r.Trigger.Mode != TriggerModeEvery {
		return errors.New("invalid rule Trigger.Mode")
	}
	return nil
}
