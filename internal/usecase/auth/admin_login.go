package auth

import (
	"context"
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrAdminInactive      = errors.New("admin is inactive")
)

type Admin struct {
	ID           string
	Email        string
	PasswordHash string
	IsActive     bool
}

type AdminFinder interface {
	FindByEmail(ctx context.Context, email string) (*Admin, error)
}

type AdminLoginUsecase struct {
	finder         AdminFinder
	jwtSecret      string
	expiresMinutes int
}

func NewAdminLoginUsecase(finder AdminFinder, jwtSecret string, expiresMinutes int) *AdminLoginUsecase {
	return &AdminLoginUsecase{
		finder:         finder,
		jwtSecret:      jwtSecret,
		expiresMinutes: expiresMinutes,
	}
}

type LoginInput struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type LoginOutput struct {
	AccessToken string `json:"accessToken"`
	ExpiresIn   int    `json:"expiresInSeconds"`
}

func (u *AdminLoginUsecase) Execute(ctx context.Context, in LoginInput) (*LoginOutput, error) {
	admin, err := u.finder.FindByEmail(ctx, in.Email)
	if err != nil {
		return nil, ErrInvalidCredentials
	}
	if !admin.IsActive {
		return nil, ErrAdminInactive
	}
	if err := bcrypt.CompareHashAndPassword([]byte(admin.PasswordHash), []byte(in.Password)); err != nil {
		return nil, ErrInvalidCredentials
	}

	now := time.Now()
	exp := now.Add(time.Duration(u.expiresMinutes) * time.Minute)

	claims := jwt.MapClaims{
		"sub":   admin.ID,
		"email": admin.Email,
		"role":  "admin",
		"iat":   now.Unix(),
		"exp":   exp.Unix(),
	}

	t := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := t.SignedString([]byte(u.jwtSecret))
	if err != nil {
		return nil, err
	}

	return &LoginOutput{
		AccessToken: signed,
		ExpiresIn:   int(time.Until(exp).Seconds()),
	}, nil
}
