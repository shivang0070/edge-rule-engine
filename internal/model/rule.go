package model

import "errors"

const (
	SourceState       = "state"
	SourceEvent       = "event"
	TriggerModeRising = "rising"
	TriggerModeEvery  = "every"
)

type Rule struct {
	ID        string         `json:"id"`
	Name      string         `json:"name"`
	Enabled   bool           `json:"enabled"`
	Scope     RuleScope      `json:"scope"`
	Source    string         `json:"source"` // "state" or "event"
	Condition Condition      `json:"condition"`
	Window    *WindowConfig  `json:"window,omitempty"`
	Sustain   *SustainConfig `json:"sustain,omitempty"`
	Trigger   TriggerConfig  `json:"trigger"`
	Action    ActionConfig   `json:"action"`
	CreatedAt int64          `json:"createdAt"`
	UpdatedAt int64          `json:"updatedAt"`
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

func (r *Rule) Validate() error {
	if r.ID == "" {
		return errors.New("rule ID is required")
	}
	if r.Name == "" {
		return errors.New("rule Name is required")
	}
	if r.Scope.CameraId == "" {
		return errors.New("rule Scope.CameraId is required")
	}
	if r.Source != SourceState && r.Source != SourceEvent {
		return errors.New("rule Source must be either state or event")
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
