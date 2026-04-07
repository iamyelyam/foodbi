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

// CreateInvite generates an invite token for a new employee.
func (s *Service) CreateInvite(ctx context.Context, companyID, createdBy uuid.UUID, email, role string) (string, error) {
	token := uuid.New().String()
	expiresAt := time.Now().Add(7 * 24 * time.Hour)

	_, err := s.db.Exec(ctx,
		`INSERT INTO invites (id, company_id, email, role, token, created_by, expires_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		uuid.New(), companyID, email, role, token, createdBy, expiresAt)
	if err != nil {
		return "", fmt.Errorf("create invite: %w", err)
	}
	return token, nil
}

// AcceptInvite registers a user from an invite token.
func (s *Service) AcceptInvite(ctx context.Context, token, password, firstName, lastName string) (*TokenPair, error) {
	var inviteID, companyID uuid.UUID
	var email, role string
	var accepted bool
	var expiresAt time.Time

	err := s.db.QueryRow(ctx,
		`SELECT id, company_id, email, role, accepted, expires_at FROM invites WHERE token = $1`,
		token).Scan(&inviteID, &companyID, &email, &role, &accepted, &expiresAt)
	if err != nil {
		return nil, fmt.Errorf("invalid invite token")
	}
	if accepted {
		return nil, fmt.Errorf("invite already used")
	}
	if time.Now().After(expiresAt) {
		return nil, fmt.Errorf("invite expired")
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), 12)
	if err != nil {
		return nil, fmt.Errorf("hash password: %w", err)
	}

	tx, err := s.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	user := &models.User{
		ID:        uuid.New(),
		CompanyID: companyID,
		Email:     email,
		FirstName: firstName,
		LastName:  lastName,
		Role:      models.Role(role),
		IsActive:  true,
	}

	_, err = tx.Exec(ctx,
		`INSERT INTO users (id, company_id, email, password_hash, first_name, last_name, role, is_active, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, true, NOW(), NOW())`,
		user.ID, user.CompanyID, user.Email, string(hash), user.FirstName, user.LastName, user.Role)
	if err != nil {
		return nil, fmt.Errorf("create user: %w", err)
	}

	_, err = tx.Exec(ctx, "UPDATE invites SET accepted = true WHERE id = $1", inviteID)
	if err != nil {
		return nil, fmt.Errorf("update invite: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit: %w", err)
	}

	return s.generateTokenPair(ctx, user)
}

// ForgotPassword generates a reset token.
func (s *Service) ForgotPassword(ctx context.Context, email string) error {
	token := uuid.New().String()
	expires := time.Now().Add(1 * time.Hour)

	tag, err := s.db.Exec(ctx,
		"UPDATE users SET reset_token = $1, reset_token_expires = $2, updated_at = NOW() WHERE email = $3",
		token, expires, email)
	if err != nil || tag.RowsAffected() == 0 {
		// SECURITY: don't reveal if email exists
		return nil
	}
	// In production: send email with reset link containing token
	return nil
}

// ResetPassword validates reset token and updates password.
func (s *Service) ResetPassword(ctx context.Context, token, newPassword string) error {
	var userID uuid.UUID
	var expires time.Time

	err := s.db.QueryRow(ctx,
		"SELECT id, reset_token_expires FROM users WHERE reset_token = $1",
		token).Scan(&userID, &expires)
	if err != nil {
		return fmt.Errorf("invalid reset token")
	}
	if time.Now().After(expires) {
		return fmt.Errorf("reset token expired")
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(newPassword), 12)
	if err != nil {
		return fmt.Errorf("hash password: %w", err)
	}

	_, err = s.db.Exec(ctx,
		"UPDATE users SET password_hash = $1, reset_token = NULL, reset_token_expires = NULL, updated_at = NOW() WHERE id = $2",
		string(hash), userID)
	return err
}

func generateOTP() (string, error) {
	max := big.NewInt(999999)
	n, err := rand.Int(rand.Reader, max)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%06d", n.Int64()), nil
}
