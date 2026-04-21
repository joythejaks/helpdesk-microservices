package domain

type TicketRepository interface {
	Create(ticket *Ticket) error
	FindAll(limit, offset int) ([]Ticket, error)
	FindByUser(userID uint, limit, offset int) ([]Ticket, error)
	FindByID(id uint) (*Ticket, error)
}
