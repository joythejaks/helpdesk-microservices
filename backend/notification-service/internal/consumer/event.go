package consumer

// event mirrors the wire-format envelope ticket-service publishes
// (internal/delivery/messaging/event.go there). No shared Go module exists
// between these services, so the contract is kept in sync by convention.
type event struct {
	Type         string   `json:"type"`
	TicketID     uint     `json:"ticket_id"`
	Title        string   `json:"title,omitempty"`
	Status       string   `json:"status,omitempty"`
	TargetUserID *uint    `json:"target_user_id,omitempty"`
	TargetRoles  []string `json:"target_roles,omitempty"`
}
