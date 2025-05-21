package types

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v4"
	"gorm.io/gorm"
)

type Timestamps struct {
	CreatedAt time.Time      `gorm:"autoCreateTime:nano" json:"created_at,omitempty"`
	UpdatedAt time.Time      `gorm:"autoUpdateTime:nano" json:"updated_at,omitempty"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"deleted_at,omitempty,omitnil"`
}

type JSONB map[string]any
type JSONBArray []any
type JSONBAny struct {
	Inner any
}

func (a JSONB) Value() (driver.Value, error) {
	valueString, err := json.Marshal(a)
	return string(valueString), err
}
func (a *JSONB) Scan(value any) error {
	b, ok := value.([]byte)
	if !ok {
		return errors.New("type assertion to []byte failed")
	}
	if err := json.Unmarshal(b, &a); err != nil {
		return err
	}
	return nil
}

func (a JSONBArray) Value() (driver.Value, error) {
	valueString, err := json.Marshal(a)
	return string(valueString), err
}
func (a *JSONBArray) Scan(value any) error {
	b, ok := value.([]byte)
	if !ok {
		return errors.New("type assertion to []byte failed")
	}
	if err := json.Unmarshal(b, &a); err != nil {
		return err
	}
	return nil
}

func (a JSONBAny) Value() (driver.Value, error) {
	valueString, err := json.Marshal(a.Inner)
	return string(valueString), err
}
func (a *JSONBAny) Scan(value any) error {
	b, ok := value.([]byte)
	if !ok {
		return errors.New("type assertion to []byte failed")
	}
	var inner any
	if err := json.Unmarshal(b, &inner); err != nil {
		return err
	}
	a.Inner = inner
	return nil
}

type Model struct {
	Timestamps

	ID uint `gorm:"id,primaryKey"`
}

type EventQueryFilters struct {
	OpensAt       string
	OpensBefore   string
	OpensAfter    string
	Organizer     string
	CreatedAt     string
	CreatedBefore string
	CreatedAfter  string
}

type CreateEventRequestBody struct {
	Title        string  `json:"title" binding:"required"`
	Name         string  `json:"name" binding:"required"`
	Description  string  `json:"description,omitempty"`
	Location     string  `json:"location,omitempty" binding:"required"`
	DateTime     string  `json:"date_time" binding:"required,bookabledate" time_format:"2006-01-02 15:04:05 -07:00"`
	Deadline     string  `json:"deadline" binding:"required,bookabledate,ltdate=DateTime" time_format:"2006-01-02 15:04:05 -07:00"`
	Seats        uint    `json:"seats,omitempty"`
	Organization uint    `json:"organization" binding:"required"`
	Publish      bool    `json:"publish,omitempty"`
	OpensAt      *string `json:"opens_at,omitempty" binding:"omitempty,bookabledate,ltdate=Deadline" time_format:"2006-01-02 15:04:05 -07:00"`
	Mode         string  `json:"mode,omitempty"`
}

type CreateTicketRequestBody struct {
	Tier     string  `json:"tier" binding:"required"`
	Type     string  `json:"type" binding:"required"`
	Currency string  `json:"currency" binding:"required"`
	Price    float32 `json:"price" binding:"required"`
	EventID  uint    `json:"event" binding:"required"`
	Limited  bool    `json:"limited,omitempty"`
	Limit    uint    `json:"limit,omitempty"`
}

type CreateOrganizationRequestBody struct {
	Name         string `json:"name" binding:"required"`
	About        string `json:"about,omitempty"`
	Country      string `json:"country,omitempty"`
	OwnerID      uint   `json:"owner" binding:"required"`
	ContactEmail string `json:"email" binding:"required"`
	Type         string `json:"type,omitempty"`
}

type ReservationTicket struct {
	TicketID uint  `json:"ticket" binding:"required"`
	Qty      uint8 `json:"qty" binding:"required"`
}

type SimpleRequestParams struct {
	ID uint `uri:"id" binding:"required"`
}

type CreateBookingRequestBody struct {
	Items []ReservationTicket `json:"items" binding:"required,min=1" `
}

type RegisterUserRequestBody struct {
	Email string `json:"email" binding:"required"`
}

type CreateAdmissionRequestBody struct {
	ReservationID uint   `json:"reservation_id"`
	Code          string `json:"code" binding:"required"`
}

type Status string

const (
	DRAFT    Status = "draft"
	OPEN     Status = "open"
	CLOSED   Status = "closed"
	ARCHIVED Status = "archived"
)

type EventStatus string

const (
	EVENT_DRAFT          EventStatus = "draft"
	EVENT_TICKETS_NOTIFY EventStatus = "notify"
	EVENT_TICKETS_OPEN   EventStatus = "open"
	EVENT_TICKETS_CLOSED EventStatus = "closed"
	EVENT_COMPLETED      EventStatus = "completed"
	EVENT_EXPIRED        EventStatus = "expired"
	EVENT_CANCELED       EventStatus = "canceled"
	EVENT_ARCHIVED       EventStatus = "archived"
)

type EventSubscriptionStatus string

const (
	EVENT_SUBSCRIPTION_NOTIFY   EventSubscriptionStatus = "notify"
	EVENT_SUBSCRIPTION_ACTIVE   EventSubscriptionStatus = "active"
	EVENT_SUBSCRIPTION_DISABLED EventSubscriptionStatus = "disabled"
)

type TicketStatus string

const (
	TICKET_DRAFT       = "draft"
	TICKET_OPEN        = "open"
	TICKET_CLOSED      = "closed"
	TICKET_ARCHIVED    = "archived"
	TICKET_UNAVAILABLE = "unavailable"
)

type ReservationStatus string

const (
	RESERVATION_PENDING   ReservationStatus = "pending"
	RESERVATION_CANCELED  ReservationStatus = "canceled"
	RESERVATION_COMPLETED ReservationStatus = "completed"
)

type BookingStatus string

const (
	BOOKING_PENDING   BookingStatus = "pending"
	BOOKING_COMPLETED BookingStatus = "completed"
	BOOKING_CANCELED  BookingStatus = "canceled"
	BOOKING_EXPIRED   BookingStatus = "expired"
)

type TransactionStatus string

const (
	TRANSACTION_PENDING    TransactionStatus = "pending"
	TRANSACTION_PROCESSING TransactionStatus = "processing"
	TRANSACTION_COMPLETED  TransactionStatus = "paid"
	TRANSACTION_CANCELED   TransactionStatus = "canceled"
	TRANSACTION_EXPIRED    TransactionStatus = "expired"
)

type OrganizationType string

const (
	ORG_STANDARD OrganizationType = "standard"
	ORG_PERSONAL OrganizationType = "personal"
)

type Metadata map[string]any

type APIResponseEvent struct {
	ID          uint           `json:"id,omitempty"`
	CreatedAt   *time.Time     `json:"created_at,omitempty"`
	UpdatedAt   *time.Time     `json:"updated_at,omitempty"`
	DeletedAt   gorm.DeletedAt `json:"deleted_at,omitempty"`
	Title       string         `json:"title,omitempty"`
	Name        string         `json:"name,omitempty"`
	About       *string        `json:"about,omitempty"`
	Location    string         `json:"location,omitempty"`
	DateTime    *time.Time     `json:"date_time,omitempty"`
	Status      *string        `json:"status,omitempty"`
	OrganizerID *uint          `json:"organizer,omitempty"`
	Seats       *uint          `json:"seats,omitempty"`
	CreatedBy   *uint          `json:"created_by,omitempty"`

	Timestamps
}

type APIResponseTicket struct {
	ID       uint     `json:"id"`
	Type     *string  `json:"type,omitempty"`
	Tier     *string  `json:"tier,omitempty"`
	Status   *string  `json:"status,omitempty"`
	Price    *float32 `json:"price,omitempty"`
	Currency string   `json:"currency,omitempty"`
	Limited  bool     `json:"limited,omitempty"`
	Limit    uint     `json:"limit,omitempty"`
	EventID  *uint    `json:"event_id,omitempty"`

	Timestamps
}

type APIResponseBooking struct {
	ID        uint    `json:"id,omitempty"`
	TicketID  uint    `json:"ticket_id,omitempty"`
	Status    string  `json:"status,omitempty"`
	Qty       uint8   `json:"qty,omitempty"`
	UnitPrice float32 `json:"unit_price,omitempty"`
	Subtotal  float32 `json:"subtotal,omitempty"`
	Currency  string  `json:"currency,omitempty"`
	UserID    uint    `json:"user_id,omitempty"`
	EventID   uint    `json:"event_id,omitempty"`

	Event   *APIResponseEvent    `json:"event,omitempty"`
	Tickets []*APIResponseTicket `json:"reserved_tickets,omitempty"`

	Timestamps
}

type APIResponseReservation struct {
	ID         uint       `json:"id"`
	TicketID   *uint      `json:"ticket_id,omitempty"`
	BookingID  *uint      `json:"booking_id,omitempty"`
	ValidUntil *time.Time `json:"valid_until,omitempty"`

	Ticket  *APIResponseTicket  `json:"ticket,omitempty"`
	Booking *APIResponseBooking `json:"booking,omitempty"`
}

type APIResponseOrganization struct {
	ID      uint   `json:"id"`
	Name    string `json:"name,omitempty"`
	About   string `json:"about,omitempty"`
	Country string `json:"country,omitempty"`
	OwnerID uint   `json:"owner_id,omitempty"`
	Type    string `json:",omitempty"`

	Events []*APIResponseEvent `json:"events,omitempty"`
	Owner  *APIResponseUser    `json:"owner,omitempty"`
}

type APIResponseUser struct {
	ID        uint    `json:"id"`
	Name      string  `json:"name,omitempty"`
	Email     string  `json:"email,omitempty"`
	Role      string  `json:"role,omitempty"`
	UID       *string `json:"uid,omitempty"`
	ActiveOrg *uint   `json:"active_org,omitempty"`

	Bookings      []*APIResponseBooking      `json:"bookings,omitempty"`
	Organizations []*APIResponseOrganization `json:"organizations,omitempty"`
}

type OpenEventStatusJobFn func(id uint)

type TicketDownloadURIParams struct {
	TicketID      uint `uri:"id" binding:"required"`
	ReservationID uint `uri:"resId" binding:"required"`
}

type TicketReservationsURIParams struct {
	TicketID uint `uri:"id" binding:"required"`
}

type OrganizationsQueryFilters struct {
	Type  string `form:"type" binding:"required"`
	Owned bool   `form:"owned,omitempty" binding:"omitempty"`
}

type Claims struct {
	Username     string   `json:"username"`
	Role         string   `json:"role"`
	Permissions  []string `json:"permissions"`
	Organization uint
	jwt.RegisteredClaims
}

type CreateSettingRequestBody struct {
	Key   string `json:"key" binding:"required"`
	Value any    `json:"value" binding:"required"`
	Group string `json:"group" binding:"required"`
}

type Handler func(payload string)
