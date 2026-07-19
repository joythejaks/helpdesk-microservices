package http

import (
	"testing"

	"ticket-service/internal/domain"
)

func TestTicketActivityTarget_StaffActionNotifiesOwner(t *testing.T) {
	ticket := &domain.Ticket{ID: 1, UserID: 7}

	targetUserID, targetRoles := ticketActivityTarget(ticket, "agent")
	if targetUserID == nil || *targetUserID != 7 {
		t.Fatalf("expected agent action to target owner (7), got %v", targetUserID)
	}
	if targetRoles != nil {
		t.Fatalf("expected no role target, got %v", targetRoles)
	}
}

func TestTicketActivityTarget_OwnerActionNotifiesAssignedAgent(t *testing.T) {
	agentID := uint(5)
	ticket := &domain.Ticket{ID: 1, UserID: 7, AssignedAgentID: &agentID}

	targetUserID, targetRoles := ticketActivityTarget(ticket, "user")
	if targetUserID == nil || *targetUserID != 5 {
		t.Fatalf("expected owner action to target assigned agent (5), got %v", targetUserID)
	}
	if targetRoles != nil {
		t.Fatalf("expected no role target, got %v", targetRoles)
	}
}

func TestTicketActivityTarget_OwnerActionOnUnassignedTicketNotifiesAdmins(t *testing.T) {
	ticket := &domain.Ticket{ID: 1, UserID: 7}

	targetUserID, targetRoles := ticketActivityTarget(ticket, "user")
	if targetUserID != nil {
		t.Fatalf("expected no user target, got %v", targetUserID)
	}
	if len(targetRoles) != 1 || targetRoles[0] != "admin" {
		t.Fatalf("expected admin role target, got %v", targetRoles)
	}
}
