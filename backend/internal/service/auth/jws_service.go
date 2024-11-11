package auth

import (
	"crypto/ecdsa"
	"fmt"

	"github.com/lestrrat-go/jwx/jwa"
	"github.com/lestrrat-go/jwx/jwk"
	"github.com/lestrrat-go/jwx/jws"
	"github.com/lestrrat-go/jwx/jwt"
	"github.com/oapi-codegen/oapi-codegen/v2/pkg/ecdsafile"
	"github.com/olga-larina/otus-highload/backend/internal/model"
)

const (
	keyID            = `fake-key-id`
	fakeIssuer       = "fake-issuer"
	fakeAudience     = "example-users"
	permissionsClaim = "perm"
	userIdClaim      = "user-id"
)

type FakeAuthenticator struct {
	privateKey *ecdsa.PrivateKey
	keySet     jwk.Set
}

// сервис по созданию и проверке токенов, использует ECDSA key из конфигов
func NewFakeAuthenticator(privateKey string) (*FakeAuthenticator, error) {
	privKey, err := ecdsafile.LoadEcdsaPrivateKey([]byte(privateKey))
	if err != nil {
		return nil, fmt.Errorf("loading PEM private key: %w", err)
	}

	set := jwk.NewSet()
	pubKey := jwk.NewECDSAPublicKey()

	err = pubKey.FromRaw(&privKey.PublicKey)
	if err != nil {
		return nil, fmt.Errorf("parsing jwk key: %w", err)
	}

	err = pubKey.Set(jwk.AlgorithmKey, jwa.ES256)
	if err != nil {
		return nil, fmt.Errorf("setting key algorithm: %w", err)
	}

	err = pubKey.Set(jwk.KeyIDKey, keyID)
	if err != nil {
		return nil, fmt.Errorf("setting key ID: %w", err)
	}

	set.Add(pubKey)

	return &FakeAuthenticator{privateKey: privKey, keySet: set}, nil
}

// проверка корректности токена по критичным claims
func (f *FakeAuthenticator) ValidateJWS(jwsString string) (jwt.Token, error) {
	return jwt.Parse([]byte(jwsString), jwt.WithKeySet(f.keySet),
		jwt.WithAudience(fakeAudience), jwt.WithIssuer(fakeIssuer))
}

// подпись JWT приватным ключом, возвращает JWS.
func (f *FakeAuthenticator) SignToken(t jwt.Token) ([]byte, error) {
	hdr := jws.NewHeaders()
	if err := hdr.Set(jws.AlgorithmKey, jwa.ES256); err != nil {
		return nil, fmt.Errorf("setting algorithm: %w", err)
	}
	if err := hdr.Set(jws.TypeKey, "JWT"); err != nil {
		return nil, fmt.Errorf("setting type: %w", err)
	}
	if err := hdr.Set(jws.KeyIDKey, keyID); err != nil {
		return nil, fmt.Errorf("setting Key ID: %w", err)
	}
	return jwt.Sign(t, jwa.ES256, f.privateKey, jwt.WithHeaders(hdr))
}

// создание токена с userId
func (f *FakeAuthenticator) CreateJWS(userId string) (string, error) {
	return f.CreateJWSWithClaims(userId, nil)
}

// создание токена с permissionsClaims и userId
func (f *FakeAuthenticator) CreateJWSWithClaims(userId string, claims []string) (string, error) {
	t := jwt.New()
	var err error
	if err = t.Set(jwt.IssuerKey, fakeIssuer); err != nil {
		return "", fmt.Errorf("setting issuer: %w", err)
	}
	if err = t.Set(jwt.AudienceKey, fakeAudience); err != nil {
		return "", fmt.Errorf("setting audience: %w", err)
	}
	if claims != nil {
		if err = t.Set(permissionsClaim, claims); err != nil {
			return "", fmt.Errorf("setting permissions: %w", err)
		}
	}
	if err = t.Set(userIdClaim, userId); err != nil {
		return "", fmt.Errorf("setting userId: %w", err)
	}

	token, err := f.SignToken(t)
	if err != nil {
		return "", err
	}
	return string(token), nil
}

// список claims из токена
func (f *FakeAuthenticator) GetClaimsFromToken(t jwt.Token) ([]string, error) {
	rawPerms, found := t.Get(permissionsClaim)
	if !found {
		// If the perms aren't found, it means that the token has none, but it has
		// passed signature validation by now, so it's a valid token, so we return
		// the empty list.
		return make([]string, 0), nil
	}

	// rawPerms will be an untyped JSON list, so we need to convert it to
	// a string list.
	rawList, ok := rawPerms.([]interface{})
	if !ok {
		return nil, fmt.Errorf("'%s' claim is unexpected type'", permissionsClaim)
	}

	claims := make([]string, len(rawList))

	for i, rawClaim := range rawList {
		var ok bool
		claims[i], ok = rawClaim.(string)
		if !ok {
			return nil, fmt.Errorf("%s[%d] is not a string", permissionsClaim, i)
		}
	}
	return claims, nil
}

func (f *FakeAuthenticator) CheckTokenClaims(expectedClaims []string, t jwt.Token) error {
	claims, err := f.GetClaimsFromToken(t)
	if err != nil {
		return fmt.Errorf("getting claims from token: %w", err)
	}
	claimsMap := make(map[string]bool, len(claims))
	for _, c := range claims {
		claimsMap[c] = true
	}

	for _, e := range expectedClaims {
		if !claimsMap[e] {
			return model.ErrClaimsInvalid
		}
	}
	return nil
}

func (f *FakeAuthenticator) ExtractUserId(t jwt.Token) (string, error) {
	rawUserId, found := t.Get(userIdClaim)
	if !found {
		return "", fmt.Errorf("userId not found")
	}
	return rawUserId.(string), nil
}
