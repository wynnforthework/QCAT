package database

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

// User represents a user in the system
type User struct {
	ID           uuid.UUID  `json:"id" db:"id"`
	Username     string     `json:"username" db:"username"`
	Email        string     `json:"email" db:"email"`
	PasswordHash string     `json:"-" db:"password_hash"`
	Role         string     `json:"role" db:"role"`
	Status       string     `json:"status" db:"status"`
	LastLogin    *time.Time `json:"last_login" db:"last_login"`
	CreatedAt    time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at" db:"updated_at"`
}

// UserSession represents a user session with refresh token
type UserSession struct {
	ID           uuid.UUID `json:"id" db:"id"`
	UserID       uuid.UUID `json:"user_id" db:"user_id"`
	RefreshToken string    `json:"refresh_token" db:"refresh_token"`
	ExpiresAt    time.Time `json:"expires_at" db:"expires_at"`
	CreatedAt    time.Time `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time `json:"updated_at" db:"updated_at"`
}

// CreateUser creates a new user
func (db *DB) CreateUser(ctx context.Context, username, email, password, role string) (*User, error) {
	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	user := &User{
		ID:           uuid.UUID{},
		Username:     username,
		Email:        email,
		PasswordHash: string(hashedPassword),
		Role:         role,
		Status:       "active",
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	query := `
		INSERT INTO users (id, username, email, password_hash, role, status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`

	_, err = db.ExecContext(ctx, query,
		user.ID.String(), user.Username, user.Email, user.PasswordHash,
		user.Role, user.Status, user.CreatedAt, user.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	return user, nil
}

// GetUserByUsername retrieves a user by username
func (db *DB) GetUserByUsername(ctx context.Context, username string) (*User, error) {
	query := `
		SELECT id, username, email, password_hash, role, status, last_login, created_at, updated_at
		FROM users WHERE username = $1 AND status = 'active'
	`

	var idStr string
	user := &User{}
	err := db.QueryRowContext(ctx, query, username).Scan(
		&idStr, &user.Username, &user.Email, &user.PasswordHash,
		&user.Role, &user.Status, &user.LastLogin, &user.CreatedAt, &user.UpdatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("user not found")
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	user.ID, err = uuid.Parse(idStr)
	if err != nil {
		return nil, fmt.Errorf("failed to parse user ID: %w", err)
	}

	return user, nil
}

// GetUserByEmail retrieves a user by email
func (db *DB) GetUserByEmail(ctx context.Context, email string) (*User, error) {
	query := `
		SELECT id, username, email, password_hash, role, status, last_login, created_at, updated_at
		FROM users WHERE email = $1 AND status = 'active'
	`

	var idStr string
	user := &User{}
	err := db.QueryRowContext(ctx, query, email).Scan(
		&idStr, &user.Username, &user.Email, &user.PasswordHash,
		&user.Role, &user.Status, &user.LastLogin, &user.CreatedAt, &user.UpdatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("user not found")
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	user.ID, err = uuid.Parse(idStr)
	if err != nil {
		return nil, fmt.Errorf("failed to parse user ID: %w", err)
	}

	return user, nil
}

// GetUserByID retrieves a user by ID
func (db *DB) GetUserByID(ctx context.Context, userID uuid.UUID) (*User, error) {
	query := `
		SELECT id, username, email, password_hash, role, status, last_login, created_at, updated_at
		FROM users WHERE id = $1 AND status = 'active'
	`

	var idStr string
	user := &User{}
	err := db.QueryRowContext(ctx, query, userID.String()).Scan(
		&idStr, &user.Username, &user.Email, &user.PasswordHash,
		&user.Role, &user.Status, &user.LastLogin, &user.CreatedAt, &user.UpdatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("user not found")
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	user.ID, err = uuid.Parse(idStr)
	if err != nil {
		return nil, fmt.Errorf("failed to parse user ID: %w", err)
	}

	return user, nil
}

// UpdateUserLastLogin updates the last login time for a user
func (db *DB) UpdateUserLastLogin(ctx context.Context, userID uuid.UUID) error {
	query := `
		UPDATE users SET last_login = $1, updated_at = $2
		WHERE id = $3
	`

	_, err := db.ExecContext(ctx, query, time.Now(), time.Now(), userID.String())
	if err != nil {
		return fmt.Errorf("failed to update last login: %w", err)
	}

	return nil
}

// CheckUserExists checks if a user exists by username or email
func (db *DB) CheckUserExists(ctx context.Context, username, email string) (bool, error) {
	query := `
		SELECT COUNT(*) FROM users WHERE username = $1 OR email = $2
	`

	var count int
	err := db.QueryRowContext(ctx, query, username, email).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("failed to check user existence: %w", err)
	}

	return count > 0, nil
}

// CreateUserSession creates a new user session with refresh token
func (db *DB) CreateUserSession(ctx context.Context, userID uuid.UUID, refreshToken string, expiresAt time.Time) (*UserSession, error) {
	session := &UserSession{
		ID:           uuid.New(),
		UserID:       userID,
		RefreshToken: refreshToken,
		ExpiresAt:    expiresAt,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	query := `
		INSERT INTO user_sessions (id, user_id, refresh_token, expires_at, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`

	_, err := db.ExecContext(ctx, query,
		session.ID.String(), session.UserID.String(), session.RefreshToken,
		session.ExpiresAt, session.CreatedAt, session.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to create user session: %w", err)
	}

	return session, nil
}

// GetUserSessionByToken retrieves a user session by refresh token
func (db *DB) GetUserSessionByToken(ctx context.Context, refreshToken string) (*UserSession, error) {
	query := `
		SELECT id, user_id, refresh_token, expires_at, created_at, updated_at
		FROM user_sessions WHERE refresh_token = $1 AND expires_at > $2
	`

	var idStr, userIDStr string
	session := &UserSession{}
	err := db.QueryRowContext(ctx, query, refreshToken, time.Now()).Scan(
		&idStr, &userIDStr, &session.RefreshToken,
		&session.ExpiresAt, &session.CreatedAt, &session.UpdatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("session not found or expired")
		}
		return nil, fmt.Errorf("failed to get user session: %w", err)
	}

	session.ID, err = uuid.Parse(idStr)
	if err != nil {
		return nil, fmt.Errorf("failed to parse session ID: %w", err)
	}

	session.UserID, err = uuid.Parse(userIDStr)
	if err != nil {
		return nil, fmt.Errorf("failed to parse user ID: %w", err)
	}

	return session, nil
}

// DeleteUserSession deletes a user session
func (db *DB) DeleteUserSession(ctx context.Context, sessionID uuid.UUID) error {
	query := `DELETE FROM user_sessions WHERE id = $1`

	_, err := db.ExecContext(ctx, query, sessionID.String())
	if err != nil {
		return fmt.Errorf("failed to delete user session: %w", err)
	}

	return nil
}

// DeleteExpiredSessions deletes expired user sessions
func (db *DB) DeleteExpiredSessions(ctx context.Context) error {
	query := `DELETE FROM user_sessions WHERE expires_at <= $1`

	_, err := db.ExecContext(ctx, query, time.Now())
	if err != nil {
		return fmt.Errorf("failed to delete expired sessions: %w", err)
	}

	return nil
}

// ValidatePassword validates a password against a user's password hash
func ValidatePassword(password, hashedPassword string) error {
	return bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password))
}
