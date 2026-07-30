package donations

import (
	"strings"

	"gorm.io/gorm/clause"
)

// lockForUpdate takes a row lock so two concurrent webhook deliveries for the
// same reference cannot both pass the already-settled check and double-count.
// Postgres-specific, which is fine — the whole stack is Postgres.
func lockForUpdate() clause.Locking {
	return clause.Locking{Strength: "UPDATE"}
}

// isUniqueViolation reports whether an error is a unique-constraint breach.
//
// Matched on message text rather than by importing the pgx error type: this
// package should not take a driver dependency just to classify one error, and
// the surrounding code treats a false negative as a plain failure rather than
// doing anything unsafe. Postgres reports SQLSTATE 23505 for this.
func isUniqueViolation(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "23505") ||
		strings.Contains(msg, "duplicate key value") ||
		strings.Contains(msg, "unique constraint")
}
