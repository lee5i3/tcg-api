package useraccounts

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	ddbtypes "github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"golang.org/x/crypto/bcrypt"
)

// fakeDynamo covers exactly what the store uses: GetItem and conditional
// transactional puts.
type fakeDynamo struct {
	mu    sync.Mutex
	items map[string]map[string]ddbtypes.AttributeValue
}

func s(av ddbtypes.AttributeValue) string {
	if v, ok := av.(*ddbtypes.AttributeValueMemberS); ok {
		return v.Value
	}
	return ""
}

func key(m map[string]ddbtypes.AttributeValue) string { return s(m["PK"]) + "|" + s(m["SK"]) }

func (f *fakeDynamo) GetItem(_ context.Context, in *dynamodb.GetItemInput, _ ...func(*dynamodb.Options)) (*dynamodb.GetItemOutput, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return &dynamodb.GetItemOutput{Item: f.items[key(in.Key)]}, nil
}

func (f *fakeDynamo) TransactWriteItems(_ context.Context, in *dynamodb.TransactWriteItemsInput, _ ...func(*dynamodb.Options)) (*dynamodb.TransactWriteItemsOutput, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, t := range in.TransactItems {
		if t.Put != nil && t.Put.ConditionExpression != nil {
			if _, exists := f.items[key(t.Put.Item)]; exists {
				return nil, &ddbtypes.TransactionCanceledException{
					CancellationReasons: []ddbtypes.CancellationReason{{Code: aws.String("ConditionalCheckFailed")}},
				}
			}
		}
	}
	for _, t := range in.TransactItems {
		if t.Put != nil {
			f.items[key(t.Put.Item)] = t.Put.Item
		}
	}
	return &dynamodb.TransactWriteItemsOutput{}, nil
}

func newTestStore() *Store {
	st := New(&fakeDynamo{items: map[string]map[string]ddbtypes.AttributeValue{}}, "test-table")
	st.bcryptCost = bcrypt.MinCost // keep tests fast
	ids := 0
	st.newID = func() string { ids++; return fmt.Sprintf("user-%d", ids) }
	st.now = func() time.Time { return time.Date(2026, 8, 14, 0, 0, 0, 0, time.UTC) }
	return st
}

func TestRegisterAndAuthenticate(t *testing.T) {
	st := newTestStore()
	ctx := context.Background()

	user, err := st.Register(ctx, " Lee@Example.COM ", "hunter2secure", "Lee")
	if err != nil {
		t.Fatal(err)
	}
	if user.Email != "lee@example.com" || user.Name != "Lee" || user.ID == "" {
		t.Errorf("user = %+v", user)
	}

	got, err := st.Authenticate(ctx, "LEE@example.com", "hunter2secure")
	if err != nil || got.ID != user.ID {
		t.Errorf("authenticate: %+v, %v", got, err)
	}
	if _, err := st.Authenticate(ctx, "lee@example.com", "wrong-password"); !errors.Is(err, ErrBadCredentials) {
		t.Errorf("wrong password err = %v", err)
	}
	if _, err := st.Authenticate(ctx, "nobody@example.com", "hunter2secure"); !errors.Is(err, ErrBadCredentials) {
		t.Errorf("unknown email err = %v", err)
	}
}

func TestRegisterValidation(t *testing.T) {
	st := newTestStore()
	ctx := context.Background()
	if _, err := st.Register(ctx, "not-an-email", "hunter2secure", ""); !errors.Is(err, ErrInvalid) {
		t.Errorf("bad email err = %v", err)
	}
	if _, err := st.Register(ctx, "lee@example.com", "short", ""); !errors.Is(err, ErrInvalid) {
		t.Errorf("short password err = %v", err)
	}
}

func TestRegisterDuplicateEmail(t *testing.T) {
	st := newTestStore()
	ctx := context.Background()
	if _, err := st.Register(ctx, "lee@example.com", "hunter2secure", ""); err != nil {
		t.Fatal(err)
	}
	if _, err := st.Register(ctx, "LEE@EXAMPLE.COM", "otherpassword", ""); !errors.Is(err, ErrEmailTaken) {
		t.Errorf("duplicate email err = %v", err)
	}
}

func TestFindOrCreateOAuthUser(t *testing.T) {
	st := newTestStore()
	ctx := context.Background()

	// New identity + new email → fresh social-only account.
	u1, err := st.FindOrCreateOAuthUser(ctx, "google", "goog-123", "Lee@Example.com", "Lee")
	if err != nil {
		t.Fatal(err)
	}
	if u1.Email != "lee@example.com" || u1.Name != "Lee" {
		t.Errorf("user = %+v", u1)
	}
	// Social-only accounts cannot password-login.
	if _, err := st.Authenticate(ctx, "lee@example.com", "whatever-pass"); !errors.Is(err, ErrBadCredentials) {
		t.Errorf("password login on social-only account err = %v", err)
	}

	// Same identity again → same account, no duplicate.
	u2, err := st.FindOrCreateOAuthUser(ctx, "google", "goog-123", "lee@example.com", "Lee")
	if err != nil || u2.ID != u1.ID {
		t.Errorf("repeat identity: %+v, %v", u2, err)
	}

	// Different provider, same email → links to the existing account.
	u3, err := st.FindOrCreateOAuthUser(ctx, "apple", "appl-9", "lee@example.com", "")
	if err != nil || u3.ID != u1.ID {
		t.Errorf("email link: %+v, %v", u3, err)
	}

	// Missing identity pieces are invalid.
	if _, err := st.FindOrCreateOAuthUser(ctx, "", "x", "a@b.co", ""); !errors.Is(err, ErrInvalid) {
		t.Errorf("empty provider err = %v", err)
	}
	if _, err := st.FindOrCreateOAuthUser(ctx, "google", "s", "not-an-email", ""); !errors.Is(err, ErrInvalid) {
		t.Errorf("bad email err = %v", err)
	}
}

func TestOAuthLinksToPasswordAccount(t *testing.T) {
	st := newTestStore()
	ctx := context.Background()
	registered, _ := st.Register(ctx, "lee@example.com", "hunter2secure", "Lee")

	social, err := st.FindOrCreateOAuthUser(ctx, "facebook", "fb-77", "LEE@example.com", "Lee H")
	if err != nil || social.ID != registered.ID {
		t.Errorf("link to password account: %+v, %v", social, err)
	}
	// Password login still works after linking.
	if _, err := st.Authenticate(ctx, "lee@example.com", "hunter2secure"); err != nil {
		t.Errorf("password login after link: %v", err)
	}
}

func TestGetUser(t *testing.T) {
	st := newTestStore()
	ctx := context.Background()
	user, _ := st.Register(ctx, "lee@example.com", "hunter2secure", "Lee")

	got, err := st.GetUser(ctx, user.ID)
	if err != nil || got.Email != "lee@example.com" {
		t.Errorf("get: %+v, %v", got, err)
	}
	if _, err := st.GetUser(ctx, "missing"); !errors.Is(err, ErrNotFound) {
		t.Errorf("missing user err = %v", err)
	}
}
