package models

import (
	"database/sql"
	"time"
)

const timeLayout = time.RFC3339

// ParseTime parses an ISO 8601 timestamp string from SQLite.
func ParseTime(s string) (time.Time, error) {
	return time.Parse(timeLayout, s)
}

// FormatTime formats a time.Time as ISO 8601 for SQLite storage.
func FormatTime(t time.Time) string {
	return t.UTC().Format(timeLayout)
}

// NullableString converts a *string to sql.NullString for queries.
func NullableString(s *string) sql.NullString {
	if s == nil {
		return sql.NullString{}
	}
	return sql.NullString{String: *s, Valid: true}
}

// NullStringToPtr converts a sql.NullString to *string.
func NullStringToPtr(ns sql.NullString) *string {
	if !ns.Valid {
		return nil
	}
	return &ns.String
}

// NullableInt64 converts a *int64 to sql.NullInt64 for queries.
func NullableInt64(n *int64) sql.NullInt64 {
	if n == nil {
		return sql.NullInt64{}
	}
	return sql.NullInt64{Int64: *n, Valid: true}
}

// NullInt64ToPtr converts a sql.NullInt64 to *int64.
func NullInt64ToPtr(ni sql.NullInt64) *int64 {
	if !ni.Valid {
		return nil
	}
	return &ni.Int64
}

// NullableInt converts a *int to sql.NullInt64 for queries.
func NullableInt(n *int) sql.NullInt64 {
	if n == nil {
		return sql.NullInt64{}
	}
	return sql.NullInt64{Int64: int64(*n), Valid: true}
}

// NullInt64ToIntPtr converts a sql.NullInt64 to *int.
func NullInt64ToIntPtr(ni sql.NullInt64) *int {
	if !ni.Valid {
		return nil
	}
	v := int(ni.Int64)
	return &v
}

// NullableTime converts a *time.Time to sql.NullString for queries.
func NullableTime(t *time.Time) sql.NullString {
	if t == nil {
		return sql.NullString{}
	}
	return sql.NullString{String: FormatTime(*t), Valid: true}
}

// NullStringToTimePtr converts a sql.NullString to *time.Time.
func NullStringToTimePtr(ns sql.NullString) *time.Time {
	if !ns.Valid || ns.String == "" {
		return nil
	}
	t, err := ParseTime(ns.String)
	if err != nil {
		return nil
	}
	return &t
}

// StringPtr returns a pointer to a string.
func StringPtr(s string) *string {
	return &s
}

// Int64Ptr returns a pointer to an int64.
func Int64Ptr(n int64) *int64 {
	return &n
}

// IntPtr returns a pointer to an int.
func IntPtr(n int) *int {
	return &n
}

// BoolPtr returns a pointer to a bool.
func BoolPtr(b bool) *bool {
	return &b
}
