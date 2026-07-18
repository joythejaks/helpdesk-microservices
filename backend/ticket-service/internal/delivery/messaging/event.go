package messaging

// Event is the wire-format envelope published to RabbitMQ and consumed by
// notification-service. TargetUserID routes to one specific connected
// client; TargetRoles routes to every connected client with a matching
// role. A message should set exactly one of the two.
type Event struct {
	Type         string   `json:"type"`
	TicketID     uint     `json:"ticket_id"`
	Title        string   `json:"title,omitempty"`
	Status       string   `json:"status,omitempty"`
	TargetUserID *uint    `json:"target_user_id,omitempty"`
	TargetRoles  []string `json:"target_roles,omitempty"`
}

const (
	EventTicketCreated       = "ticket_created"
	EventTicketAssigned      = "ticket_assigned"
	EventTicketStatusChanged = "ticket_status_changed"
)
