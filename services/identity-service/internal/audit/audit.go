// Package audit writes immutable admin-action records to the shared
// audit_logs table. It's designed to be called from handlers after a
// successful admin write.
//
// Usage:
//
//	audit.Log(auditor, audit.Entry{
//	    Actor:      auditor.From(c),                  // pulls id/name/role from Gin ctx
//	    Action:     "flag.resolved",
//	    TargetType: "CONTENT_FLAG",
//	    TargetID:   flag.ID,
//	    Metadata:   map[string]any{"status": "HIDDEN"},
//	    Request:    c.Request,                         // for IP + UA
//	})
//
// The package intentionally never returns errors to the caller — a
// failed audit write is logged to stdout but the request continues.
// Audit is an addition, not a gate on the primary action.
package audit

import (
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/civicos/identity-service/internal/domain"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

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

// FromContext extracts the actor from a Gin context populated by JWTAuth.
// Callers use this at the top of an admin handler so the audit block
// stays a single line.
func FromContext(c *gin.Context) Actor {
	name, _ := c.Get("userName")
	role, _ := c.Get("userRole")
	id, _ := c.Get("userID")
	nameStr, _ := name.(string)
	roleStr, _ := role.(string)
	idStr, _ := id.(string)
	return Actor{ID: idStr, Name: nameStr, Role: roleStr}
}

// Log writes an entry. Never returns an error — if the DB is down or the
// metadata is unmarshalable, we log to stdout and move on so the primary
// admin action isn't gated on the audit write.
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
	row := &domain.AuditLog{
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

// clientIP prefers the first X-Forwarded-For entry over RemoteAddr since
// requests come through the gateway. Gin's ClientIP would work too but
// pulling directly keeps this package framework-agnostic.
func clientIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		// take the first comma-separated address
		for i := 0; i < len(xff); i++ {
			if xff[i] == ',' {
				return xff[:i]
			}
		}
		return xff
	}
	return r.RemoteAddr
}
