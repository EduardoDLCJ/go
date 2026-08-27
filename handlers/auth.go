package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"

	"apisql/config"
	"apisql/database"
	"apisql/middleware"
	"apisql/models"
	"apisql/services"
	"apisql/utils"

	"github.com/gin-gonic/gin"
)

// Config is an alias to the shared application config.
type Config = config.Config

// AuthHandler holds dependencies for authentication endpoints.
type AuthHandler struct {
	Config *Config
}

// NewAuthHandler creates a new AuthHandler.
func NewAuthHandler(cfg *Config) *AuthHandler {
	return &AuthHandler{Config: cfg}
}

// Register creates a new user account.
// POST /api/v1/auth/register
func (h *AuthHandler) Register(c *gin.Context) {
	var req models.RegisterRequest
	log.Printf("Received registration request: %+v", req)
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ValidationError(c, err.Error())
		return
	}

	// Sanitize inputs
	req.Usuario = utils.SanitizeString(req.Usuario)
	req.Correo = utils.SanitizeString(req.Correo)

	// Check if username already exists
	var exists bool
	err := database.DB.QueryRow(database.Rebind("SELECT EXISTS(SELECT 1 FROM users WHERE usuario = ?)"), req.Usuario).Scan(&exists)
	if err != nil {
		log.Printf("Error checking username: %v", err)
		utils.InternalError(c, "")
		return
	}
	if exists {
		utils.Error(c, http.StatusConflict, "El nombre de usuario ya está en uso")
		return
	}

	// Check if email already exists
	err = database.DB.QueryRow(database.Rebind("SELECT EXISTS(SELECT 1 FROM users WHERE correo = ?)"), req.Correo).Scan(&exists)
	if err != nil {
		log.Printf("Error checking email: %v", err)
		utils.InternalError(c, "")
		return
	}
	if exists {
		utils.Error(c, http.StatusConflict, "El correo electrónico ya está registrado")
		return
	}

	// Hash password
	hashedPassword, err := services.HashPassword(req.Password)
	if err != nil {
		log.Printf("Error hashing password: %v", err)
		utils.InternalError(c, "")
		return
	}

	// Insert user
	var userID int64
	err = database.DB.QueryRow(
		database.Rebind("INSERT INTO users (usuario, correo, password) VALUES (?, ?, ?) RETURNING id"),
		req.Usuario, req.Correo, hashedPassword,
	).Scan(&userID)
	if err != nil {
		log.Printf("Error inserting user: %v", err)
		utils.InternalError(c, "")
		return
	}

	// Generate tokens
	accessToken, err := middleware.GenerateToken(h.Config, uint(userID), req.Usuario, middleware.AccessToken)
	if err != nil {
		log.Printf("Error generating access token: %v", err)
		utils.InternalError(c, "")
		return
	}

	refreshToken, err := middleware.GenerateToken(h.Config, uint(userID), req.Usuario, middleware.RefreshToken)
	if err != nil {
		log.Printf("Error generating refresh token: %v", err)
		utils.InternalError(c, "")
		return
	}

	response := models.AuthResponse{
		User: models.UserResponse{
			ID:      uint(userID),
			Usuario: req.Usuario,
			Correo:  req.Correo,
		},
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    int64(h.Config.JWTAccessExpiry.Seconds()),
	}

	utils.Success(c, http.StatusCreated, "Usuario registrado exitosamente", response)
}

// Login authenticates a user and returns JWT tokens.
// POST /api/v1/auth/login
func (h *AuthHandler) Login(c *gin.Context) {
	var req models.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ValidationError(c, err.Error())
		return
	}

	// Find user by email
	var user models.User
	err := database.DB.QueryRow(
		database.Rebind("SELECT id, usuario, correo, password, created_at, updated_at FROM users WHERE correo = ?"),
		req.Correo,
	).Scan(&user.ID, &user.Usuario, &user.Correo, &user.Password, &user.CreatedAt, &user.UpdatedAt)

	if err == sql.ErrNoRows {
		// Use generic message to prevent user enumeration
		utils.Unauthorized(c, "Credenciales inválidas")
		return
	}
	if err != nil {
		log.Printf("Error querying user: %v", err)
		utils.InternalError(c, "")
		return
	}

	// Verify password
	if !services.CheckPassword(user.Password, req.Password) {
		utils.Unauthorized(c, "Credenciales inválidas")
		return
	}

	// Generate tokens
	accessToken, err := middleware.GenerateToken(h.Config, user.ID, user.Usuario, middleware.AccessToken)
	if err != nil {
		log.Printf("Error generating access token: %v", err)
		utils.InternalError(c, "")
		return
	}

	refreshToken, err := middleware.GenerateToken(h.Config, user.ID, user.Usuario, middleware.RefreshToken)
	if err != nil {
		log.Printf("Error generating refresh token: %v", err)
		utils.InternalError(c, "")
		return
	}

	response := models.AuthResponse{
		User:         user.ToResponse(),
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    int64(h.Config.JWTAccessExpiry.Seconds()),
	}

	utils.Success(c, http.StatusOK, "Inicio de sesión exitoso", response)
}

// Refresh generates a new access token from a valid refresh token.
// POST /api/v1/auth/refresh
func (h *AuthHandler) Refresh(c *gin.Context) {
	var body struct {
		RefreshToken string `json:"refresh_token" binding:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		utils.ValidationError(c, err.Error())
		return
	}

	// Parse and validate refresh token
	claims, err := middleware.ParseToken(h.Config, body.RefreshToken)
	if err != nil {
		utils.Unauthorized(c, "Refresh token inválido o expirado")
		return
	}

	// Verify user still exists
	var exists bool
	err = database.DB.QueryRow(database.Rebind("SELECT EXISTS(SELECT 1 FROM users WHERE id = ?)"), claims.UserID).Scan(&exists)
	if err != nil || !exists {
		utils.Unauthorized(c, "Usuario no encontrado")
		return
	}

	// Generate new access token
	accessToken, err := middleware.GenerateToken(h.Config, claims.UserID, claims.Usuario, middleware.AccessToken)
	if err != nil {
		log.Printf("Error generating access token: %v", err)
		utils.InternalError(c, "")
		return
	}

	utils.Success(c, http.StatusOK, "Token renovado", gin.H{
		"access_token": accessToken,
		"expires_in":   int64(h.Config.JWTAccessExpiry.Seconds()),
	})
}

// Me returns the authenticated user's profile.
// GET /api/v1/auth/me
func (h *AuthHandler) Me(c *gin.Context) {
	userID := middleware.GetUserID(c)
	if userID == 0 {
		utils.Unauthorized(c, "")
		return
	}

	var user models.User
	err := database.DB.QueryRow(
		database.Rebind("SELECT id, usuario, correo, created_at, updated_at FROM users WHERE id = ?"),
		userID,
	).Scan(&user.ID, &user.Usuario, &user.Correo, &user.CreatedAt, &user.UpdatedAt)

	if err == sql.ErrNoRows {
		utils.NotFound(c, "Usuario no encontrado")
		return
	}
	if err != nil {
		log.Printf("Error querying user: %v", err)
		utils.InternalError(c, "")
		return
	}

	utils.Success(c, http.StatusOK, "", user.ToResponse())
}

// verifyGoogleToken verifies the Google ID token using Google's tokeninfo API.
func verifyGoogleToken(idToken string) (string, string, error) {
	resp, err := http.Get("https://oauth2.googleapis.com/tokeninfo?id_token=" + idToken)
	if err != nil {
		return "", "", fmt.Errorf("failed to call Google tokeninfo: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", "", fmt.Errorf("Google tokeninfo returned status: %d", resp.StatusCode)
	}

	var info struct {
		Sub           string `json:"sub"`
		Email         string `json:"email"`
		EmailVerified string `json:"email_verified"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
		return "", "", fmt.Errorf("failed to decode Google tokeninfo response: %w", err)
	}

	if info.Email == "" || info.Sub == "" {
		return "", "", fmt.Errorf("invalid token claims: email or sub is empty")
	}

	return info.Email, info.Sub, nil
}

// GoogleAuth handles authenticating a user with a Google token.
// POST /api/v1/auth/google
func (h *AuthHandler) GoogleAuth(c *gin.Context) {
	var req models.GoogleAuthRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ValidationError(c, err.Error())
		return
	}

	// Verify and decode the Google Token
	email, googleID, err := verifyGoogleToken(req.GoogleToken)
	if err != nil {
		log.Printf("Google token verification failed: %v", err)
		utils.Unauthorized(c, "Token de Google inválido o expirado")
		return
	}

	// Update user to set google_id where email matches, and return the user details
	var user models.User
	err = database.DB.QueryRow(
		database.Rebind("UPDATE users SET google_id = ? WHERE correo = ? RETURNING id, usuario, correo, created_at, updated_at"),
		googleID, email,
	).Scan(&user.ID, &user.Usuario, &user.Correo, &user.CreatedAt, &user.UpdatedAt)

	if err == sql.ErrNoRows {
		utils.Error(c, http.StatusNotFound, "No existe ningún usuario registrado con el correo de esta cuenta de Google")
		return
	}
	if err != nil {
		log.Printf("Error linking Google ID to user: %v", err)
		if strings.Contains(err.Error(), "unique constraint") || strings.Contains(err.Error(), "23505") {
			utils.Error(c, http.StatusConflict, "Esta cuenta de Google ya está vinculada a otro usuario")
			return
		}
		utils.InternalError(c, "")
		return
	}

	// Generate tokens
	accessToken, err := middleware.GenerateToken(h.Config, user.ID, user.Usuario, middleware.AccessToken)
	if err != nil {
		log.Printf("Error generating access token: %v", err)
		utils.InternalError(c, "")
		return
	}

	refreshToken, err := middleware.GenerateToken(h.Config, user.ID, user.Usuario, middleware.RefreshToken)
	if err != nil {
		log.Printf("Error generating refresh token: %v", err)
		utils.InternalError(c, "")
		return
	}

	response := models.AuthResponse{
		User:         user.ToResponse(),
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    int64(h.Config.JWTAccessExpiry.Seconds()),
	}

	utils.Success(c, http.StatusOK, "Autenticación de Google exitosa", response)
}
