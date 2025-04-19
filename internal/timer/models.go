package timer

import "github.com/google/uuid"

type Update struct {
	Name string         `json:"name"`
	Args map[string]any `json:"args,omitempty"`
}

type Command struct {
	clientId uuid.UUID         `json:"-"`
	Type     string            `json:"type"`
	Name     string            `json:"name"`
	Args     map[string]string `json:"args,omitempty"`
}

type User struct {
	Id   uuid.UUID `json:"id"`
	Name string    `json:"name"`
}
