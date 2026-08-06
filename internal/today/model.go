package today

import "time"

type AttentionItem struct {
	ID               string    `json:"id"`
	Type             string    `json:"type"`
	Title            string    `json:"title"`
	WhyNow           string    `json:"why_now"`
	Scope            string    `json:"scope"`
	State            string    `json:"state"`
	Evidence         string    `json:"evidence"`
	Owner            string    `json:"owner"`
	DueAt            time.Time `json:"due_at"`
	PrimaryAction    string    `json:"primary_action"`
	ActionTargetType string    `json:"action_target_type,omitempty"`
	ActionTargetID   string    `json:"action_target_id,omitempty"`
}
