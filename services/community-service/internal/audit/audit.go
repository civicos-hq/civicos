// Package audit writes admin-action records to the shared audit_logs
// table. Identity-service owns the schema (AutoMigrate lives there) —
// this service just inserts. Same pattern as
// services/identity-service/internal/audit.
//
// Cross-service note: if services ever split databases, replace the
// direct db.Create with an HTTP POST to identity-service's internal
// audit endpoint.
package audit

import (
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// AuditLog mirrors the domain type in identity-service. Field names are
// identical so GORM writes into the same physical rows without a schema
// duplication. Do NOT AutoMigrate this — identity-service is the source
// of truth for the audit_logs schema.
type AuditLog struct {
	ID         string    `gorm:"type:uuid;primaryKey" json:"id"`
	ActorID    string    `gorm:"type:uuid;not null;index" json:"actorId"`
	ActorName  string    `gorm:"not null" json:"actorName"`
	ActorRole  string    `gorm:"not null" json:"actorRole"`
	Action     string    `gorm:"not null;index" json:"action"`
	TargetType string    `gorm:"not null;index" json:"targetType"`
	TargetID   string    `gorm:"type:uuid;not null;index" json:"targetId"`
	Metadata   string    `gorm:"type:jsonb;default:'{}'" json:"metadata"`
	IPAddress  *string   `json:"ipAddress,omitempty"`
	UserAgent  *string   `json:"userAgent,omitempty"`
	CreatedAt  time.Time `json:"createdAt"`
}

// TableName pins us to the shared table even if GORM's pluralizer
// changes its mind about audit_log vs audit_logs.
func (AuditLog) TableName() string { return "audit_logs" }

type Auditor struct{ db *gorm.DB }

func New(db *gorm.DB) *Auditor { return &Auditor{db: db} }

type Actor struct {
	ID   string
	Name string
	Role string
}

type Entry struct {
	Actor      Actor
	Action     string
	TargetType string
	TargetID   string
	Metadata   map[string]any
	Request    *http.Request
}

func FromContext(c *gin.Context) Actor {
	name, _ := c.Get("userName")
	role, _ := c.Get("userRole")
	id, _ := c.Get("userID")
	nameStr, _ := name.(string)
	roleStr, _ := role.(string)
	idStr, _ := id.(string)
	return Actor{ID: idStr, Name: nameStr, Role: roleStr}
}

// Log writes an audit row. Never returns an error — audit is an
// addition, never a gate on the primary action. See identity-service's
// audit.go for the same guarantee.
func (a *Auditor) Log(e Entry) {
	if e.Actor.ID == "" || e.Action == "" || e.TargetID == "" {
		log.Printf("⚠️ audit: refusing empty entry (action=%q target=%q)", e.Action, e.TargetID)
		return
	}
	metaBytes := []byte("{}")
	if len(e.Metadata) > 0 {
		if b, err := json.Marshal(e.Metadata); err == nil {
			metaBytes = b
		}
	}
	var ip, ua *string
	if e.Request != nil {
		if v := clientIP(e.Request); v != "" {
			ip = &v
		}
		if v := e.Request.UserAgent(); v != "" {
			ua = &v
		}
	}
	row := &AuditLog{
		ID:         uuid.New().String(),
		ActorID:    e.Actor.ID,
		ActorName:  e.Actor.Name,
		ActorRole:  e.Actor.Role,
		Action:     e.Action,
		TargetType: e.TargetType,
		TargetID:   e.TargetID,
		Metadata:   string(metaBytes),
		IPAddress:  ip,
		UserAgent:  ua,
		CreatedAt:  time.Now().UTC(),
	}
	if err := a.db.Create(row).Error; err != nil {
		log.Printf("⚠️ audit: write failed action=%q target=%q err=%v", e.Action, e.TargetID, err)
	}
}

func clientIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		for i := 0; i < len(xff); i++ {
			if xff[i] == ',' {
				return xff[:i]
			}
		}
		return xff
	}
	return r.RemoteAddr
}
