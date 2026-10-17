package services

import (
	"time"

	"github.com/lib/pq"
	"gorm.io/gorm"
)

type Router struct {
	ID        string    `json:"id" gorm:"type:uuid;primaryKey"`
	Name      string    `json:"name"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (r *Router) BeforeCreate(_ *gorm.DB) error {
	if r.ID == "" {
		r.ID = newID()
	}
	return nil
}

type Device struct {
	ID          string     `json:"id" gorm:"type:uuid;primaryKey"`
	RouterID    string     `json:"router_id" gorm:"type:uuid"`
	Name        string     `json:"name"`
	MACAddress  string     `json:"mac_address"`
	LastKnownIP *string    `json:"last_known_ip,omitempty"`
	LastSeen    *time.Time `json:"last_seen,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

func (d *Device) BeforeCreate(_ *gorm.DB) error {
	if d.ID == "" {
		d.ID = newID()
	}
	return nil
}

type DeviceGroup struct {
	ID          string    `json:"id" gorm:"type:uuid;primaryKey"`
	RouterID    string    `json:"router_id" gorm:"type:uuid"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

func (g *DeviceGroup) BeforeCreate(_ *gorm.DB) error {
	if g.ID == "" {
		g.ID = newID()
	}
	return nil
}

type DNSPolicy struct {
	ID            string    `json:"id" gorm:"type:uuid;primaryKey"`
	Name          string    `json:"name"`
	Description   string    `json:"description"`
	Enabled       bool      `json:"enabled"`
	DefaultAction string    `json:"default_action"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

func (p *DNSPolicy) BeforeCreate(_ *gorm.DB) error {
	if p.ID == "" {
		p.ID = newID()
	}
	return nil
}

type DNSRule struct {
	ID        string    `json:"id" gorm:"type:uuid;primaryKey"`
	PolicyID  string    `json:"policy_id" gorm:"type:uuid"`
	Domain    string    `json:"domain"`
	Action    string    `json:"action"`
	Enabled   bool      `json:"enabled"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (r *DNSRule) BeforeCreate(_ *gorm.DB) error {
	if r.ID == "" {
		r.ID = newID()
	}
	return nil
}

type PolicyAssignment struct {
	ID        string    `json:"id" gorm:"type:uuid;primaryKey"`
	PolicyID  string    `json:"policy_id" gorm:"type:uuid"`
	DeviceID  *string   `json:"device_id,omitempty" gorm:"type:uuid"`
	GroupID   *string   `json:"group_id,omitempty" gorm:"type:uuid"`
	Priority  int       `json:"priority"`
	CreatedAt time.Time `json:"created_at"`
}

func (a *PolicyAssignment) BeforeCreate(_ *gorm.DB) error {
	if a.ID == "" {
		a.ID = newID()
	}
	return nil
}

type RouterAPIKey struct {
	ID         string         `json:"id" gorm:"type:uuid;primaryKey"`
	RouterID   string         `json:"router_id" gorm:"type:uuid"`
	KeyHash    string         `json:"-"`
	Scopes     pq.StringArray `json:"scopes" gorm:"type:text[]"`
	CreatedAt  time.Time      `json:"created_at"`
	ExpiresAt  *time.Time     `json:"expires_at,omitempty"`
	RevokedAt  *time.Time     `json:"revoked_at,omitempty"`
	LastUsedAt *time.Time     `json:"last_used_at,omitempty"`
}

type DeviceLease struct {
	RouterID   string `gorm:"type:uuid;primaryKey"`
	DeviceID   string `gorm:"type:uuid"`
	IP         string `gorm:"type:inet;primaryKey"`
	ObservedAt time.Time
	ExpiresAt  time.Time
}
