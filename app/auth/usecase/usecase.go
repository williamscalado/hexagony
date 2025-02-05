package usecase

import (
	"context"
	"errors"
	authDomain "hexagony/app/auth/domain"
	usersDomain "hexagony/app/users/domain"
	"hexagony/lib/crypto"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v4"
	"github.com/google/uuid"
)

type authUseCase struct {
	authRepo authDomain.AuthRepository
}

func NewAuthUsecase(auth authDomain.AuthRepository) authDomain.AuthUseCase {
	return &authUseCase{
		authRepo: auth,
	}
}

func (a *authUseCase) Authenticate(ctx context.Context, email, password string) (*authDomain.AuthToken, error) {
	user, err := a.authRepo.Authenticate(ctx, email)
	if err != nil {
		return nil, err
	}

	bcrypt := crypto.New()

	if match := bcrypt.CheckPasswordHash(password, user.Password); !match {
		return nil, errors.New("authentication failed")
	}

	customClaims := &usersDomain.User{
		UUID:  user.UUID,
		Name:  user.Name,
		Email: user.Email,
	}

	jwtDuration := os.Getenv("JWT_DURATION")

	if jwtDuration == "" {
		jwtDuration = "60m"
	}

	duration, err := time.ParseDuration(jwtDuration)
	if err != nil {
		return nil, err
	}

	expiration := time.Duration(time.Minute * duration)
	tokenExpiration := time.Now().Add(expiration)

	token, err := a.generateToken("user", customClaims, tokenExpiration)
	if err != nil {
		return nil, err
	}

	authToken := authDomain.AuthToken{Token: token}

	return &authToken, nil
}

func (a *authUseCase) generateToken(
	claimKey string,
	claimValue *usersDomain.User,
	expiration time.Time,
) (string, error) {
	if claimKey == "" || claimValue == nil {
		return "", authDomain.ErrEmptyClaim
	}

	signingKey := []byte(os.Getenv("JWT_SECRET"))

	claims := struct {
		jwt.RegisteredClaims
		UUID  uuid.UUID `json:"id"`
		Name  string    `json:"name"`
		Email string    `json:"email"`
	}{
		jwt.RegisteredClaims{
			Issuer:    "Hexagony",
			Subject:   "https://github.com/cyruzin/hexagony",
			Audience:  jwt.ClaimStrings{"Clean Architecture"},
			ExpiresAt: jwt.NewNumericDate(expiration),
		},
		claimValue.UUID,
		claimValue.Name,
		claimValue.Email,
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	payload, err := token.SignedString(signingKey)
	if err != nil {
		return "", authDomain.ErrSign
	}

	return payload, nil
}
