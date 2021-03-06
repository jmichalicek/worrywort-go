package worrywort

import (
	"crypto/sha512"
	"database/sql"
	"encoding/base64"
	"errors"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
	"strings"
	"time"
	// "log"
)

var ErrInvalidToken error = errors.New("Invalid token. Not found.")

var ErrBadTokenFormat = errors.New("Token should be formatted as `tokenId:secret` but was not")

// TODO: Possibly move authToken stuff to its own package so that scope stuff will be
// authToken.READ_ALL, etc.
type AuthTokenScopeType int64

const (
	TOKEN_SCOPE_ALL AuthTokenScopeType = iota
	TOKEN_SCOPE_READ_ALL
	TOKEN_SCOPE_WRITE_TEMPS
	TOKEN_SCOPE_READ_TEMPS
)

type AuthTokenType int

const (
	TOKEN_TYPE_LOGIN AuthTokenType = iota
	TOKEN_TYPE_PERSONAL_ACCESS
)

// Simplified auth tokens.  May eventually be replaced with proper OAuth 2.
type AuthToken struct {
	// Can this just be a uuid type? Tried and it did not play nicely, but should be possible
	// TODO: have UserId as just id
	// TODO: For now we store both short lived login and long lived personal access tokens. Arguably the login tokens
	// could be handled purely in memory JWT style, but couldn't easily expire them early then still to force a logout of just
	// a single device/instance.
	Id         string             `db:"id"`
	Token      string             `db:"token"`
	User       User               `db:"user"`
	ExpiresAt  pq.NullTime        `db:"expires_at"` // TODO: just make a *time.Time
	CreatedAt  time.Time          `db:"created_at"`
	UpdatedAt  time.Time          `db:"updated_at"`
	Scope      AuthTokenScopeType `db:"scope"`
	Type       AuthTokenType      `db:"type"`
	fromString string             // usually empty, the string this token was generated from
}

func (t AuthToken) ForAuthenticationHeader() string {
	// TODO: Base64 encode this?
	// "encoding/base64"
	return t.Id + ":" + t.fromString
}
func (t *AuthToken) Save(db *sqlx.DB) error {
	// TODO: May change the name of this table as it suggests a joining table.
	tokenId := new(string)
	createdAt := new(time.Time)
	updatedAt := new(time.Time)
	if t.Id != "" {
		return nil
	}
	query := db.Rebind(`INSERT INTO user_authtokens (token, expires_at, updated_at, scope, user_id, type)
		VALUES (?, ?, NOW(), ?, ?, ?) RETURNING id, created_at, updated_at`)
	err := db.QueryRow(
		query, t.Token, t.ExpiresAt, t.Scope, t.User.Id, t.Type).Scan(tokenId, createdAt, updatedAt)
	if err == nil {
		t.Id = *tokenId
		t.CreatedAt = *createdAt
		t.UpdatedAt = *updatedAt
	}
	return err
}

func (t AuthToken) Compare(token string) bool {
	tokenHash := MakeTokenHash(token)
	return tokenHash == t.Token
}

// Make a hashed token from string.  May rename
func MakeTokenHash(tokenStr string) string {
	tokenBytes := sha512.Sum512([]byte(tokenStr))
	// tokenBytes is a byte array, which cannot be directly cast to a string.  Instead make it a
	// byte slice for EncodeToString, which does cast properly.
	token := base64.URLEncoding.EncodeToString(tokenBytes[:])
	return token
}

// TODO: rename to NewAuthToken
func NewToken(token string, user User, scope AuthTokenScopeType, t AuthTokenType) AuthToken {
	tokenString := MakeTokenHash(token)
	return AuthToken{Token: tokenString, User: user, Scope: scope, fromString: token, Type: t}
}

// Returns an AuthToken with a hashed token for a given tokenId and token string
func NewLoginToken(token string, user User, scope AuthTokenScopeType) AuthToken {
	return NewToken(token, user, scope, TOKEN_TYPE_LOGIN)
}

// Generate a random auth token for a user with the given scope
func GenerateTokenForUser(user User, scope AuthTokenScopeType) (AuthToken, error) {
	// TODO: instead of taking hashCost, take a function which hashes the passwd - this could then do bcrypt at any cost,
	// pbkdf2, or for testing situations a simple md5 or just leave alone.
	// TODO: differentiate login token and personal access token to be able to generate both
	token, err := uuid.NewRandom()
	if err != nil {
		return AuthToken{}, err
	}

	// not sure there's much point to this, but it makes it nicer looking
	tokenb64 := base64.URLEncoding.EncodeToString([]byte(token.String()))
	return NewLoginToken(tokenb64, user, scope), nil
}

func AuthenticateUserByToken(tokenStr string, db *sqlx.DB) (AuthToken, error) {
	// TODO: Is there a good way to abstract this so that token data could optionally
	// be stored in redis while other data is in postgres?  If two separate lookups
	// are done even for db then it is easy.

	// TODO: Considering making this take token id and actual token as separate params
	// for explicitness that token is passed in has 2 parts
	token := AuthToken{}
	tokenParts := strings.SplitN(tokenStr, ":", 2)
	if len(tokenParts) != 2 {
		return token, ErrBadTokenFormat
	}

	tokenId := tokenParts[0]
	tokenSecret := tokenParts[1]
	// TODO: sqrl
	query := db.Rebind(
		`SELECT t.id, t.token, t.scope, t.expires_at, t.created_at, t.updated_at, u.id "user.id", u.uuid "user.uuid",
			u.full_name "user.full_name", u.username "user.username", u.email "user.email", u.created_at "user.created_at",
			u.updated_at "user.updated_at", u.password "user.password" FROM user_authtokens t
			JOIN users u ON t.user_id = u.id
			WHERE t.id = ? AND (t.expires_at IS NULL OR t.expires_at > ?)`)
	err := db.Get(&token, query, tokenId, time.Now())
	if err == sql.ErrNoRows {
		err = ErrInvalidToken
	}
	if !token.Compare(tokenSecret) {
		err = ErrInvalidToken
	}

	return token, err
}
