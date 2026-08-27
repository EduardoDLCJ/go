package models

import "time"

// User represents a registered user in the database.
type User struct {
	ID        uint      `json:"id"`
	Usuario   string    `json:"usuario"`
	Correo    string    `json:"correo"`
	Password  string    `json:"-"` // Never expose in JSON
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// --- Request DTOs ---

// RegisterRequest is the expected body for user registration.
type RegisterRequest struct {
	Usuario  string `json:"usuario" binding:"required,min=3,max=50"`
	Correo   string `json:"correo" binding:"required,email,max=100"`
	Password string `json:"password" binding:"required,min=8,max=72"`
}

// LoginRequest is the expected body for user login.
type LoginRequest struct {
	Correo   string `json:"correo" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

// --- Response DTOs ---

// UserResponse is the public representation of a user.
type UserResponse struct {
	ID        uint      `json:"id"`
	Usuario   string    `json:"usuario"`
	Correo    string    `json:"correo"`
	CreatedAt time.Time `json:"created_at"`
}

// AuthResponse is returned after login/register with JWT tokens.
type AuthResponse struct {
	User         UserResponse `json:"user"`
	AccessToken  string       `json:"access_token"`
	RefreshToken string       `json:"refresh_token"`
	ExpiresIn    int64        `json:"expires_in"` // seconds
}

// ToResponse converts a User model to its public response DTO.
func (u *User) ToResponse() UserResponse {
	return UserResponse{
		ID:        u.ID,
		Usuario:   u.Usuario,
		Correo:    u.Correo,
		CreatedAt: u.CreatedAt,
	}
}

// GoogleAuthRequest is the expected body for Google authentication.
type GoogleAuthRequest struct {
	GoogleToken string `json:"google_token" binding:"required"`
}

