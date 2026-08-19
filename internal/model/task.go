// Package model contains the application's data structures.
package model

import "time"

// Sada required data structure.
// Go ch har attribute case sensitive hunda hega, capital letter nal start karna
// penda take doojhe packages vich access ho sake. JSON tags data nu easily
// encode te decode karan layi add kitte gaye ne.
type Task struct {
	ID          int       `json:"id"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Status      string    `json:"status"`
	Priority    string    `json:"priority"`
	CreatedAt   time.Time `json:"created_at"`
}
