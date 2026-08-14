// Package useraccounts is end-user account storage for the app: register,
// authenticate (bcrypt), and fetch. Accounts live in the same DynamoDB
// single table as the catalog, under their own key prefixes:
//
//	Entity          PK                             SK    Notes
//	user            USER#{id}                      META  email lowercased; passwordHash bcrypt (absent for social-only accounts)
//	email guard     USEREMAIL#{email}              UNIQ  refId → user id; transactional uniqueness + login lookup
//	identity guard  USERIDENT#{provider}#{subject} UNIQ  refId → user id; one per linked OAuth identity
//
// This package is intentionally separate from the card-catalog store —
// accounts are a different function, not catalog data.
package useraccounts

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	ddbtypes "github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

// ErrInvalid is returned for malformed input (bad email, short password).
var ErrInvalid = errors.New("invalid input")

// ErrEmailTaken is returned when registering an email that already exists.
var ErrEmailTaken = errors.New("email already registered")

// ErrBadCredentials is returned for wrong email/password combinations and
// deliberately does not distinguish which half was wrong.
var ErrBadCredentials = errors.New("invalid email or password")

// ErrNotFound is returned when a user id does not exist.
var ErrNotFound = errors.New("user not found")

// User is the account shape exposed to the API. The password hash never
// leaves this package.
type User struct {
	ID        string    `json:"id"`
	Email     string    `json:"email"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"createdAt"`
}

// DynamoAPI is the slice of the DynamoDB client this store uses.
type DynamoAPI interface {
	GetItem(ctx context.Context, in *dynamodb.GetItemInput, opts ...func(*dynamodb.Options)) (*dynamodb.GetItemOutput, error)
	TransactWriteItems(ctx context.Context, in *dynamodb.TransactWriteItemsInput, opts ...func(*dynamodb.Options)) (*dynamodb.TransactWriteItemsOutput, error)
}

type Store struct {
	DB    DynamoAPI
	Table string

	now        func() time.Time
	newID      func() string
	bcryptCost int
}

func New(db DynamoAPI, table string) *Store {
	return &Store{DB: db, Table: table, now: time.Now, newID: uuid.NewString, bcryptCost: bcrypt.DefaultCost}
}

func userPK(id string) string     { return "USER#" + id }
func emailPK(email string) string { return "USEREMAIL#" + email }

func identPK(provider, subject string) string { return "USERIDENT#" + provider + "#" + subject }

type userItem struct {
	PK           string    `dynamodbav:"PK"`
	SK           string    `dynamodbav:"SK"`
	Entity       string    `dynamodbav:"entity"`
	ID           string    `dynamodbav:"id"`
	Email        string    `dynamodbav:"email"`
	Name         string    `dynamodbav:"name"`
	PasswordHash []byte    `dynamodbav:"passwordHash"`
	CreatedAt    time.Time `dynamodbav:"createdAt"`
	UpdatedAt    time.Time `dynamodbav:"updatedAt"`
}

type guardItem struct {
	PK    string `dynamodbav:"PK"`
	SK    string `dynamodbav:"SK"`
	RefID string `dynamodbav:"refId"`
}

func (u userItem) toUser() User {
	return User{ID: u.ID, Email: u.Email, Name: u.Name, CreatedAt: u.CreatedAt}
}

// emailRe is deliberately loose — real validation is the verification email
// we don't send yet. It only rejects obvious garbage.
var emailRe = regexp.MustCompile(`^[^@\s]+@[^@\s]+\.[^@\s]+$`)

const minPasswordLen = 8

func normEmail(raw string) (string, error) {
	email := strings.ToLower(strings.TrimSpace(raw))
	if !emailRe.MatchString(email) {
		return "", fmt.Errorf("%w: enter a valid email address", ErrInvalid)
	}
	return email, nil
}

// Register creates an account. The email must be unused; the password is
// stored as a bcrypt hash only.
func (s *Store) Register(ctx context.Context, email, password, name string) (User, error) {
	email, err := normEmail(email)
	if err != nil {
		return User{}, err
	}
	if len(password) < minPasswordLen {
		return User{}, fmt.Errorf("%w: password must be at least %d characters", ErrInvalid, minPasswordLen)
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), s.bcryptCost)
	if err != nil {
		return User{}, err
	}
	now := s.now().UTC()
	item := userItem{
		PK: userPK(s.newID()), SK: "META",
		Entity: "user", Email: email, Name: strings.TrimSpace(name),
		PasswordHash: hash, CreatedAt: now, UpdatedAt: now,
	}
	item.ID = strings.TrimPrefix(item.PK, "USER#")

	userAV, err := attributevalue.MarshalMap(item)
	if err != nil {
		return User{}, err
	}
	guardAV, err := attributevalue.MarshalMap(guardItem{PK: emailPK(email), SK: "UNIQ", RefID: item.ID})
	if err != nil {
		return User{}, err
	}
	_, err = s.DB.TransactWriteItems(ctx, &dynamodb.TransactWriteItemsInput{
		TransactItems: []ddbtypes.TransactWriteItem{
			{Put: &ddbtypes.Put{
				TableName: aws.String(s.Table), Item: guardAV,
				ConditionExpression: aws.String("attribute_not_exists(PK)"),
			}},
			{Put: &ddbtypes.Put{TableName: aws.String(s.Table), Item: userAV}},
		},
	})
	if conditionFailed(err) {
		return User{}, ErrEmailTaken
	}
	if err != nil {
		return User{}, err
	}
	return item.toUser(), nil
}

// Authenticate verifies an email/password pair and returns the account.
func (s *Store) Authenticate(ctx context.Context, email, password string) (User, error) {
	normalized, err := normEmail(email)
	if err != nil {
		return User{}, ErrBadCredentials
	}
	var guard guardItem
	found, err := s.getItem(ctx, emailPK(normalized), "UNIQ", &guard)
	if err != nil {
		return User{}, err
	}
	if !found {
		// Burn comparable time so unknown emails aren't distinguishable by
		// response latency.
		_ = bcrypt.CompareHashAndPassword(dummyHash, []byte(password))
		return User{}, ErrBadCredentials
	}
	var item userItem
	found, err = s.getItem(ctx, userPK(guard.RefID), "META", &item)
	if err != nil {
		return User{}, err
	}
	if !found {
		return User{}, ErrBadCredentials
	}
	if bcrypt.CompareHashAndPassword(item.PasswordHash, []byte(password)) != nil {
		return User{}, ErrBadCredentials
	}
	return item.toUser(), nil
}

// GetUser fetches an account by id (token subject).
func (s *Store) GetUser(ctx context.Context, id string) (User, error) {
	var item userItem
	found, err := s.getItem(ctx, userPK(id), "META", &item)
	if err != nil {
		return User{}, err
	}
	if !found {
		return User{}, fmt.Errorf("%w: %q", ErrNotFound, id)
	}
	return item.toUser(), nil
}

// FindOrCreateOAuthUser resolves an OAuth identity (provider + stable
// subject id) to an account:
//
//  1. a known identity returns its user;
//  2. an unknown identity whose (verified) email matches an existing account
//     is linked to that account;
//  3. otherwise a new social-only account is created (no password —
//     password sign-in stays impossible for it until one is set).
func (s *Store) FindOrCreateOAuthUser(ctx context.Context, provider, subject, email, name string) (User, error) {
	return s.findOrCreateOAuthUser(ctx, provider, subject, email, name, true)
}

func (s *Store) findOrCreateOAuthUser(ctx context.Context, provider, subject, email, name string, retry bool) (User, error) {
	if provider == "" || subject == "" {
		return User{}, fmt.Errorf("%w: missing provider identity", ErrInvalid)
	}
	email, err := normEmail(email)
	if err != nil {
		return User{}, err
	}

	// 1. Known identity.
	var ident guardItem
	found, err := s.getItem(ctx, identPK(provider, subject), "UNIQ", &ident)
	if err != nil {
		return User{}, err
	}
	if found {
		return s.GetUser(ctx, ident.RefID)
	}

	// 2. Same email → link the identity to the existing account.
	var emailGuard guardItem
	found, err = s.getItem(ctx, emailPK(email), "UNIQ", &emailGuard)
	if err != nil {
		return User{}, err
	}
	if found {
		identAV, err := attributevalue.MarshalMap(guardItem{PK: identPK(provider, subject), SK: "UNIQ", RefID: emailGuard.RefID})
		if err != nil {
			return User{}, err
		}
		_, err = s.DB.TransactWriteItems(ctx, &dynamodb.TransactWriteItemsInput{
			TransactItems: []ddbtypes.TransactWriteItem{
				{Put: &ddbtypes.Put{
					TableName: aws.String(s.Table), Item: identAV,
					ConditionExpression: aws.String("attribute_not_exists(PK)"),
				}},
			},
		})
		if conditionFailed(err) && retry {
			// Concurrent link of the same identity — re-read it.
			return s.findOrCreateOAuthUser(ctx, provider, subject, email, name, false)
		}
		if err != nil {
			return User{}, err
		}
		return s.GetUser(ctx, emailGuard.RefID)
	}

	// 3. Brand-new social-only account: user + email guard + identity guard
	// in one transaction.
	now := s.now().UTC()
	item := userItem{
		PK: userPK(s.newID()), SK: "META",
		Entity: "user", Email: email, Name: strings.TrimSpace(name),
		CreatedAt: now, UpdatedAt: now,
	}
	item.ID = strings.TrimPrefix(item.PK, "USER#")

	userAV, err := attributevalue.MarshalMap(item)
	if err != nil {
		return User{}, err
	}
	emailAV, err := attributevalue.MarshalMap(guardItem{PK: emailPK(email), SK: "UNIQ", RefID: item.ID})
	if err != nil {
		return User{}, err
	}
	identAV, err := attributevalue.MarshalMap(guardItem{PK: identPK(provider, subject), SK: "UNIQ", RefID: item.ID})
	if err != nil {
		return User{}, err
	}
	_, err = s.DB.TransactWriteItems(ctx, &dynamodb.TransactWriteItemsInput{
		TransactItems: []ddbtypes.TransactWriteItem{
			{Put: &ddbtypes.Put{
				TableName: aws.String(s.Table), Item: emailAV,
				ConditionExpression: aws.String("attribute_not_exists(PK)"),
			}},
			{Put: &ddbtypes.Put{
				TableName: aws.String(s.Table), Item: identAV,
				ConditionExpression: aws.String("attribute_not_exists(PK)"),
			}},
			{Put: &ddbtypes.Put{TableName: aws.String(s.Table), Item: userAV}},
		},
	})
	if conditionFailed(err) && retry {
		// Lost a race with a concurrent register/link — resolve again.
		return s.findOrCreateOAuthUser(ctx, provider, subject, email, name, false)
	}
	if err != nil {
		return User{}, err
	}
	return item.toUser(), nil
}

// dummyHash is a valid bcrypt hash of an unguessable value, used to equalize
// timing for unknown emails.
var dummyHash, _ = bcrypt.GenerateFromPassword([]byte(uuid.NewString()), bcrypt.DefaultCost)

func (s *Store) getItem(ctx context.Context, pk, sk string, out any) (bool, error) {
	res, err := s.DB.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: aws.String(s.Table),
		Key: map[string]ddbtypes.AttributeValue{
			"PK": &ddbtypes.AttributeValueMemberS{Value: pk},
			"SK": &ddbtypes.AttributeValueMemberS{Value: sk},
		},
	})
	if err != nil {
		return false, err
	}
	if len(res.Item) == 0 {
		return false, nil
	}
	return true, attributevalue.UnmarshalMap(res.Item, out)
}

func conditionFailed(err error) bool {
	var tc *ddbtypes.TransactionCanceledException
	if errors.As(err, &tc) {
		for _, r := range tc.CancellationReasons {
			if r.Code != nil && *r.Code == "ConditionalCheckFailed" {
				return true
			}
		}
	}
	var ccf *ddbtypes.ConditionalCheckFailedException
	return errors.As(err, &ccf)
}
