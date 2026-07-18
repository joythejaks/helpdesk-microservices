package repository

import (
	"time"

	"ticket-service/internal/domain"

	"gorm.io/gorm"
)

type ticketRepository struct {
	db *gorm.DB
}

func NewTicketRepository(db *gorm.DB) domain.TicketRepository {
	return &ticketRepository{db}
}

func applyFilter(q *gorm.DB, f domain.TicketFilter) *gorm.DB {
	if f.Status != "" {
		q = q.Where("status = ?", f.Status)
	}
	if f.Priority != "" {
		q = q.Where("priority = ?", f.Priority)
	}
	if f.Department != "" {
		q = q.Where("department = ?", f.Department)
	}
	if f.From != nil {
		q = q.Where("created_at >= ?", *f.From)
	}
	if f.To != nil {
		q = q.Where("created_at <= ?", *f.To)
	}
	return q
}

func (r *ticketRepository) Create(ticket *domain.Ticket) error {
	return r.db.Create(ticket).Error
}

func (r *ticketRepository) FindAll(filter domain.TicketFilter, limit, offset int) ([]domain.Ticket, error) {
	var tickets []domain.Ticket

	err := applyFilter(r.db, filter).
		Order("id desc").
		Limit(limit).
		Offset(offset).
		Find(&tickets).Error

	return tickets, err
}

func (r *ticketRepository) FindByUser(userID uint, filter domain.TicketFilter, limit, offset int) ([]domain.Ticket, error) {
	var tickets []domain.Ticket

	err := applyFilter(r.db.Where("user_id = ?", userID), filter).
		Order("id desc").
		Limit(limit).
		Offset(offset).
		Find(&tickets).Error

	return tickets, err
}

func (r *ticketRepository) FindByAgent(agentID uint, filter domain.TicketFilter, limit, offset int) ([]domain.Ticket, error) {
	var tickets []domain.Ticket

	err := applyFilter(r.db.Where("assigned_agent_id = ?", agentID), filter).
		Order("id desc").
		Limit(limit).
		Offset(offset).
		Find(&tickets).Error

	return tickets, err
}

func (r *ticketRepository) FindUnassigned(filter domain.TicketFilter, limit, offset int) ([]domain.Ticket, error) {
	var tickets []domain.Ticket

	err := applyFilter(r.db.Where("assigned_agent_id IS NULL"), filter).
		Order("id desc").
		Limit(limit).
		Offset(offset).
		Find(&tickets).Error

	return tickets, err
}

func (r *ticketRepository) FindByID(id uint) (*domain.Ticket, error) {
	var ticket domain.Ticket

	err := r.db.First(&ticket, id).Error
	return &ticket, err
}

func (r *ticketRepository) AssignAndTransition(ticketID, agentID, changedBy uint, changedAt time.Time) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		var ticket domain.Ticket
		if err := tx.First(&ticket, ticketID).Error; err != nil {
			return err
		}

		fromStatus := ticket.Status
		if err := tx.Model(&ticket).Updates(map[string]interface{}{
			"assigned_agent_id": agentID,
			"assigned_at":       changedAt,
			"status":            domain.StatusAssigned,
		}).Error; err != nil {
			return err
		}

		return tx.Create(&domain.TicketStatusHistory{
			TicketID:   ticketID,
			FromStatus: fromStatus,
			ToStatus:   domain.StatusAssigned,
			ChangedBy:  changedBy,
			ChangedAt:  changedAt,
		}).Error
	})
}

func (r *ticketRepository) TransitionStatus(ticketID uint, fromStatus, toStatus string, changedBy uint, changedAt time.Time) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		updates := map[string]interface{}{"status": toStatus}
		if toStatus == domain.StatusResolved {
			updates["resolved_at"] = changedAt
		}
		if toStatus == domain.StatusClosed {
			updates["closed_at"] = changedAt
		}

		if err := tx.Model(&domain.Ticket{}).Where("id = ?", ticketID).Updates(updates).Error; err != nil {
			return err
		}

		return tx.Create(&domain.TicketStatusHistory{
			TicketID:   ticketID,
			FromStatus: fromStatus,
			ToStatus:   toStatus,
			ChangedBy:  changedBy,
			ChangedAt:  changedAt,
		}).Error
	})
}

func (r *ticketRepository) FindHistory(ticketID uint) ([]domain.TicketStatusHistory, error) {
	var history []domain.TicketStatusHistory
	err := r.db.
		Where("ticket_id = ?", ticketID).
		Order("changed_at asc").
		Find(&history).Error
	return history, err
}
