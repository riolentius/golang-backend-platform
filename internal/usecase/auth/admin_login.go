package auth

import (
	"context"
	"errors"
	"strconv"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrInactiveAdmin      = errors.New("admin inactive")
)

type AdminFinder interface {
	FindByEmail(ctx context.Context, email string) (*Admin, error)
}

type Admin struct {
	ID           string
	Email        string
	PasswordHash string
	IsActive     bool
}

type LoginResult struct {
	AccessToken string `json:"accessToken"`
	ExpiresIn   int    `json:"expiresIn"` // seconds
}

type AdminLoginUsecase struct {
	finder    AdminFinder
	jwtSecret []byte
	expMin    int
}

func NewAdminLoginUsecase(finder AdminFinder, jwtSecret string, expiresMinutes string) *AdminLoginUsecase {
	m, _ := strconv.Atoi(expiresMinutes)
	if m <= 0 {
		m = 60
	}
	return &AdminLoginUsecase{
		finder:    finder,
		jwtSecret: []byte(jwtSecret),
		expMin:    m,
	}
}

func (u *AdminLoginUsecase) Execute(ctx context.Context, email, password string) (*LoginResult, error) {
	admin, err := u.finder.FindByEmail(ctx, email)
	if err != nil {
		// Hide whether email exists
		return nil, ErrInvalidCredentials
	}
	if !admin.IsActive {
		return nil, ErrInactiveAdmin
	}

	if err := bcrypt.CompareHashAndPassword([]byte(admin.PasswordHash), []byte(password)); err != nil {
		return nil, ErrInvalidCredentials
	}

	now := time.Now()
	exp := now.Add(time.Duration(u.expMin) * time.Minute)

	claims := jwt.MapClaims{
		"sub":   admin.ID,
		"typ":   "admin",
		"email": admin.Email,
		"iat":   now.Unix(),
		"exp":   exp.Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString(u.jwtSecret)
	if err != nil {
		return nil, err
	}

	return &LoginResult{
		AccessToken: signed,
		ExpiresIn:   u.expMin * 60,
	}, nil
}
