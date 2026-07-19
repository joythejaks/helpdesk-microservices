package http

import "ticket-service/internal/domain"

// ticketActivityTarget decides who should be notified about an activity on
// a ticket (a new comment, an attachment upload): staff (admin/assigned
// agent) acting notifies the ticket owner; the owner acting notifies the
// assigned agent, or the admin role in general if the ticket isn't
// assigned yet. Shared by CommentHandler and AttachmentHandler.
func ticketActivityTarget(ticket *domain.Ticket, actorRole string) (targetUserID *uint, targetRoles []string) {
	if actorRole == "user" {
		if ticket.AssignedAgentID != nil {
			return ticket.AssignedAgentID, nil
		}
		return nil, []string{"admin"}
	}
	return &ticket.UserID, nil
}
