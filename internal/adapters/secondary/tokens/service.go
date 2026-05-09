package tokens

import (
	"errors"
	"time"

	"GolangTemplateProject/internal/domain"
	"GolangTemplateProject/internal/ports"
	pkgjwt "GolangTemplateProject/pkg/jwt"
	"github.com/golang-jwt/jwt/v4"
)

type Service struct {
	access     *pkgjwt.JWTManager
	secret     []byte
	refreshTTL time.Duration
}

type yandexStateClaims struct {
	Kind string `json:"kind"`
	jwt.RegisteredClaims
}

func New(secretKey string, accessTTL time.Duration, refreshTTL time.Duration) *Service {
	if refreshTTL <= 0 {
		refreshTTL = 7 * 24 * time.Hour
	}

	return &Service{
		access:     pkgjwt.NewJWTManager(secretKey, accessTTL),
		secret:     []byte(secretKey),
		refreshTTL: refreshTTL,
	}
}

func (s *Service) IssueTokens(user *domain.UserAccount) (ports.TokenPair, error) {
	accessToken, err := s.access.Generate(user.ID.String(), user.Email)
	if err != nil {
		return ports.TokenPair{}, err
	}
	refreshToken, err := s.generateRefreshToken(user)
	if err != nil {
		return ports.TokenPair{}, err
	}

	return ports.TokenPair{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}, nil
}

func (s *Service) NewYandexState() (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, yandexStateClaims{
		Kind: "yandex_oauth_state",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(10 * time.Minute)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	})
	return token.SignedString(s.secret)
}

func (s *Service) VerifyYandexState(state string) error {
	token, err := jwt.ParseWithClaims(state, &yandexStateClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return s.secret, nil
	})
	if err != nil {
		return err
	}
	claims, ok := token.Claims.(*yandexStateClaims)
	if !ok || !token.Valid {
		return errors.New("invalid yandex state token")
	}
	if claims.Kind != "yandex_oauth_state" {
		return errors.New("unexpected yandex state token")
	}
	return nil
}

func (s *Service) generateRefreshToken(user *domain.UserAccount) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id": user.ID.String(),
		"type":    "refresh",
		"exp":     time.Now().Add(s.refreshTTL).Unix(),
		"iat":     time.Now().Unix(),
	})
	return token.SignedString(s.secret)
}
