package auth

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"math/big"
	"os"
	"time"

	"github.com/foodbi/backend/internal/models"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrUserExists     = errors.New("user with this email already exists")
	ErrInvalidCreds   = errors.New("invalid email or password")
	ErrUserNotActive  = errors.New("account not activated")
	ErrInvalidOTP     = errors.New("invalid or expired OTP code")
	ErrSessionExpired = errors.New("session expired")
)

type Service struct {
	db *pgxpool.Pool
}

func NewService(db *pgxpool.Pool) *Service {
	return &Service{db: db}
}

type RegisterInput struct {
	Email     string      `json:"email" validate:"required,email"`
	Password  string      `json:"password" validate:"required,min=8"`
	FirstName string      `json:"first_name" validate:"required"`
	LastName  string      `json:"last_name" validate:"required"`
	Phone     string      `json:"phone"`
	Role      models.Role `json:"role" validate:"required,oneof=owner employee"`
	CompanyName string    `json:"company_name"` // required for owner
}

type LoginInput struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required"`
}

type TokenPair struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresAt    int64  `json:"expires_at"`
}

func (s *Service) Register(ctx context.Context, input RegisterInput) (*models.User, error) {
	var exists bool
	err := s.db.QueryRow(ctx,
		"SELECT EXISTS(SELECT 1 FROM users WHERE email = $1)", input.Email).Scan(&exists)
	if err != nil {
		return nil, fmt.Errorf("check user exists: %w", err)
	}
	if exists {
		return nil, ErrUserExists
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(input.Password), 12)
	if err != nil {
		return nil, fmt.Errorf("hash password: %w", err)
	}

	tx, err := s.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	var companyID uuid.UUID
	if input.Role == models.RoleOwner {
		companyID = uuid.New()
		_, err = tx.Exec(ctx,
			"INSERT INTO companies (id, name, created_at, updated_at) VALUES ($1, $2, NOW(), NOW())",
			companyID, input.CompanyName)
		if err != nil {
			return nil, fmt.Errorf("create company: %w", err)
		}
	}

	otp, err := generateOTP()
	if err != nil {
		return nil, fmt.Errorf("generate OTP: %w", err)
	}
	otpExpiry := time.Now().Add(10 * time.Minute)

	user := &models.User{
		ID:           uuid.New(),
		CompanyID:    companyID,
		Email:        input.Email,
		PasswordHash: string(hash),
		FirstName:    input.FirstName,
		LastName:     input.LastName,
		Phone:        input.Phone,
		Role:         input.Role,
		IsActive:     false,
		OTPCode:      otp,
		OTPExpiresAt: &otpExpiry,
	}

	_, err = tx.Exec(ctx,
		`INSERT INTO users (id, company_id, email, password_hash, first_name, last_name, phone, role, is_active, otp_code, otp_expires_at, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, NOW(), NOW())`,
		user.ID, user.CompanyID, user.Email, user.PasswordHash,
		user.FirstName, user.LastName, user.Phone, user.Role,
		user.IsActive, user.OTPCode, user.OTPExpiresAt)
	if err != nil {
		return nil, fmt.Errorf("insert user: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit: %w", err)
	}

	// SECURITY: In production, send OTP via email, never return in response
	return user, nil
}

func (s *Service) Login(ctx context.Context, input LoginInput) (*TokenPair, error) {
	var user models.User
	err := s.db.QueryRow(ctx,
		`SELECT id, company_id, email, password_hash, first_name, last_name, role, is_active
		 FROM users WHERE email = $1`,
		input.Email).Scan(
		&user.ID, &user.CompanyID, &user.Email, &user.PasswordHash,
		&user.FirstName, &user.LastName, &user.Role, &user.IsActive)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrInvalidCreds
		}
		return nil, fmt.Errorf("query user: %w", err)
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(input.Password)); err != nil {
		return nil, ErrInvalidCreds
	}

	if !user.IsActive {
		return nil, ErrUserNotActive
	}

	return s.generateTokenPair(ctx, &user)
}

func (s *Service) VerifyOTP(ctx context.Context, email, code string) (*TokenPair, error) {
	var user models.User
	var otpCode string
	var otpExpiry *time.Time

	err := s.db.QueryRow(ctx,
		`SELECT id, company_id, email, first_name, last_name, role, otp_code, otp_expires_at
		 FROM users WHERE email = $1`,
		email).Scan(&user.ID, &user.CompanyID, &user.Email, &user.FirstName, &user.LastName, &user.Role, &otpCode, &otpExpiry)
	if err != nil {
		return nil, ErrInvalidOTP
	}

	if otpCode != code || otpExpiry == nil || time.Now().After(*otpExpiry) {
		return nil, ErrInvalidOTP
	}

	_, err = s.db.Exec(ctx,
		"UPDATE users SET is_active = true, otp_code = NULL, otp_expires_at = NULL, updated_at = NOW() WHERE id = $1",
		user.ID)
	if err != nil {
		return nil, fmt.Errorf("activate user: %w", err)
	}

	user.IsActive = true
	return s.generateTokenPair(ctx, &user)
}

func (s *Service) RefreshToken(ctx context.Context, refreshToken string) (*TokenPair, error) {
	var session models.Session
	var user models.User

	err := s.db.QueryRow(ctx,
		`SELECT s.id, s.user_id, s.expires_at, u.company_id, u.email, u.first_name, u.last_name, u.role
		 FROM sessions s JOIN users u ON s.user_id = u.id
		 WHERE s.refresh_token = $1`,
		refreshToken).Scan(&session.ID, &session.UserID, &session.ExpiresAt,
		&user.CompanyID, &user.Email, &user.FirstName, &user.LastName, &user.Role)
	if err != nil {
		return nil, ErrSessionExpired
	}

	if time.Now().After(session.ExpiresAt) {
		_, _ = s.db.Exec(ctx, "DELETE FROM sessions WHERE id = $1", session.ID)
		return nil, ErrSessionExpired
	}

	_, _ = s.db.Exec(ctx, "DELETE FROM sessions WHERE id = $1", session.ID)

	user.ID = session.UserID
	return s.generateTokenPair(ctx, &user)
}

func (s *Service) Logout(ctx context.Context, userID uuid.UUID) error {
	_, err := s.db.Exec(ctx, "DELETE FROM sessions WHERE user_id = $1", userID)
	return err
}

func (s *Service) GetUser(ctx context.Context, userID uuid.UUID) (*models.User, error) {
	var user models.User
	err := s.db.QueryRow(ctx,
		`SELECT id, company_id, email, first_name, last_name, phone, role, is_active, created_at, updated_at
		 FROM users WHERE id = $1`,
		userID).Scan(&user.ID, &user.CompanyID, &user.Email, &user.FirstName, &user.LastName,
		&user.Phone, &user.Role, &user.IsActive, &user.CreatedAt, &user.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("get user: %w", err)
	}
	return &user, nil
}

func (s *Service) generateTokenPair(ctx context.Context, user *models.User) (*TokenPair, error) {
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		secret = "dev-secret-change-in-production"
	}

	expiresAt := time.Now().Add(15 * time.Minute)
	claims := jwt.MapClaims{
		"user_id":    user.ID.String(),
		"company_id": user.CompanyID.String(),
		"role":       string(user.Role),
		"email":      user.Email,
		"exp":        expiresAt.Unix(),
		"iat":        time.Now().Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	accessToken, err := token.SignedString([]byte(secret))
	if err != nil {
		return nil, fmt.Errorf("sign token: %w", err)
	}

	refreshToken := uuid.New().String()
	refreshExpiry := time.Now().Add(7 * 24 * time.Hour)

	_, err = s.db.Exec(ctx,
		"INSERT INTO sessions (id, user_id, refresh_token, expires_at, created_at) VALUES ($1, $2, $3, $4, NOW())",
		uuid.New(), user.ID, refreshToken, refreshExpiry)
	if err != nil {
		return nil, fmt.Errorf("create session: %w", err)
	}

	return &TokenPair{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresAt:    expiresAt.Unix(),
	}, nil
}

func generateOTP() (string, error) {
	max := big.NewInt(999999)
	n, err := rand.Int(rand.Reader, max)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%06d", n.Int64()), nil
}
