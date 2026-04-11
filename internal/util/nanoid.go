package util

import gonanoid "github.com/matoous/go-nanoid/v2"

const (
	// DefaultIDLength is the standard length for all entity IDs.
	DefaultIDLength = 21

	// ShortIDLength is used for deployment short hashes in preview URLs.
	ShortIDLength = 8
)

// NewID generates a nanoid of default length (21 chars).
func NewID() string {
	id, err := gonanoid.New(DefaultIDLength)
	if err != nil {
		panic("failed to generate nanoid: " + err.Error())
	}
	return id
}

// NewShortID generates a short nanoid (8 chars) for preview URL slugs.
func NewShortID() string {
	id, err := gonanoid.New(ShortIDLength)
	if err != nil {
		panic("failed to generate short nanoid: " + err.Error())
	}
	return id
}
