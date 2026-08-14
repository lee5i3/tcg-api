package catalog

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	ddbtypes "github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/google/uuid"
)

// searchLimit caps name-search results, matching the old SQL `LIMIT 48`.
const searchLimit = 48

// DynamoAPI is the slice of the DynamoDB client the store uses; tests
// substitute a fake.
type DynamoAPI interface {
	GetItem(ctx context.Context, in *dynamodb.GetItemInput, opts ...func(*dynamodb.Options)) (*dynamodb.GetItemOutput, error)
	PutItem(ctx context.Context, in *dynamodb.PutItemInput, opts ...func(*dynamodb.Options)) (*dynamodb.PutItemOutput, error)
	DeleteItem(ctx context.Context, in *dynamodb.DeleteItemInput, opts ...func(*dynamodb.Options)) (*dynamodb.DeleteItemOutput, error)
	UpdateItem(ctx context.Context, in *dynamodb.UpdateItemInput, opts ...func(*dynamodb.Options)) (*dynamodb.UpdateItemOutput, error)
	Query(ctx context.Context, in *dynamodb.QueryInput, opts ...func(*dynamodb.Options)) (*dynamodb.QueryOutput, error)
	BatchWriteItem(ctx context.Context, in *dynamodb.BatchWriteItemInput, opts ...func(*dynamodb.Options)) (*dynamodb.BatchWriteItemOutput, error)
	TransactWriteItems(ctx context.Context, in *dynamodb.TransactWriteItemsInput, opts ...func(*dynamodb.Options)) (*dynamodb.TransactWriteItemsOutput, error)
}

// Catalog is the storage facade the API is built on.
type Catalog struct {
	DB    DynamoAPI
	Table string

	now   func() time.Time
	newID func() string
}

func New(db DynamoAPI, table string) *Catalog {
	return &Catalog{DB: db, Table: table, now: time.Now, newID: uuid.NewString}
}

// conditionFailed reports whether err is a failed conditional write —
// either a plain ConditionalCheckFailedException or a cancelled transaction
// whose cancellation reasons include one.
func conditionFailed(err error) bool {
	var ccf *ddbtypes.ConditionalCheckFailedException
	if errors.As(err, &ccf) {
		return true
	}
	var tc *ddbtypes.TransactionCanceledException
	if errors.As(err, &tc) {
		for _, r := range tc.CancellationReasons {
			if r.Code != nil && *r.Code == "ConditionalCheckFailed" {
				return true
			}
		}
	}
	return false
}

// getItem loads one item into out; found is false when the item is absent.
func (c *Catalog) getItem(ctx context.Context, pk, sk string, out any) (bool, error) {
	res, err := c.DB.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: aws.String(c.Table),
		Key:       key(pk, sk),
	})
	if err != nil {
		return false, err
	}
	if len(res.Item) == 0 {
		return false, nil
	}
	return true, attributevalue.UnmarshalMap(res.Item, out)
}

// queryAll runs a query to exhaustion, appending raw items.
func (c *Catalog) queryAll(ctx context.Context, in *dynamodb.QueryInput) ([]map[string]ddbtypes.AttributeValue, error) {
	var items []map[string]ddbtypes.AttributeValue
	for {
		res, err := c.DB.Query(ctx, in)
		if err != nil {
			return nil, err
		}
		items = append(items, res.Items...)
		if len(res.LastEvaluatedKey) == 0 {
			return items, nil
		}
		in.ExclusiveStartKey = res.LastEvaluatedKey
	}
}

// ---------------------------------------------------------------------------
// Games
// ---------------------------------------------------------------------------

var defaultGames = []Game{
	{Key: "pokemon", Label: "Pokémon"},
	{Key: "magic", Label: "Magic: The Gathering"},
	{Key: "lorcana", Label: "Disney Lorcana"},
	{Key: "sports", Label: "Sports cards"},
}

// Seed inserts the default games when the catalog has none.
func (c *Catalog) Seed(ctx context.Context) error {
	res, err := c.DB.Query(ctx, &dynamodb.QueryInput{
		TableName:                 aws.String(c.Table),
		IndexName:                 aws.String(gsi1),
		KeyConditionExpression:    aws.String("GSI1PK = :pk"),
		ExpressionAttributeValues: map[string]ddbtypes.AttributeValue{":pk": &ddbtypes.AttributeValueMemberS{Value: gamesPK}},
		Limit:                     aws.Int32(1),
	})
	if err != nil {
		return err
	}
	if len(res.Items) > 0 {
		return nil
	}
	for _, g := range defaultGames {
		if _, err := c.AddGame(ctx, g.Key, g.Label, nil); err != nil && !errors.Is(err, ErrConflict) {
			return err
		}
	}
	return nil
}

// Games lists every game, oldest first (creation order, like the old
// `ORDER BY created_at`).
func (c *Catalog) Games(ctx context.Context) ([]Game, error) {
	items, err := c.queryAll(ctx, &dynamodb.QueryInput{
		TableName:                 aws.String(c.Table),
		IndexName:                 aws.String(gsi1),
		KeyConditionExpression:    aws.String("GSI1PK = :pk"),
		ExpressionAttributeValues: map[string]ddbtypes.AttributeValue{":pk": &ddbtypes.AttributeValueMemberS{Value: gamesPK}},
	})
	if err != nil {
		return nil, err
	}
	games := []Game{}
	for _, raw := range items {
		var item gameItem
		if err := attributevalue.UnmarshalMap(raw, &item); err != nil {
			return nil, err
		}
		games = append(games, item.toGame())
	}
	return games, nil
}

func (c *Catalog) AddGame(ctx context.Context, gameKey, label string, language *string) (Game, error) {
	if !gameKeyRe.MatchString(gameKey) {
		return Game{}, fmt.Errorf("%w: key must be 2–32 characters: lowercase letters, numbers, dashes", ErrInvalid)
	}
	if strings.TrimSpace(label) == "" {
		return Game{}, fmt.Errorf("%w: give the game a name", ErrInvalid)
	}
	lang, err := normLanguage(language)
	if err != nil {
		return Game{}, err
	}
	now := c.now().UTC()
	item := gameItem{
		PK: gamePK(c.newID()), SK: skMeta,
		GSI1PK: gamesPK,
		Entity: "game", Key: gameKey, Language: lang, Label: strings.TrimSpace(label),
		CreatedAt: now, UpdatedAt: now,
	}
	item.ID = strings.TrimPrefix(item.PK, "GAME#")
	item.GSI1SK = now.Format(time.RFC3339Nano) + "#" + item.ID

	gameAV, err := marshal(item)
	if err != nil {
		return Game{}, err
	}
	guardAV, err := marshal(guardItem{PK: gameKeyPK(gameKey), SK: skUniq, RefID: item.ID})
	if err != nil {
		return Game{}, err
	}
	_, err = c.DB.TransactWriteItems(ctx, &dynamodb.TransactWriteItemsInput{
		TransactItems: []ddbtypes.TransactWriteItem{
			{Put: &ddbtypes.Put{
				TableName: aws.String(c.Table), Item: guardAV,
				ConditionExpression: aws.String("attribute_not_exists(PK)"),
			}},
			{Put: &ddbtypes.Put{TableName: aws.String(c.Table), Item: gameAV}},
		},
	})
	if conditionFailed(err) {
		return Game{}, fmt.Errorf("%w: game %q", ErrConflict, gameKey)
	}
	if err != nil {
		return Game{}, err
	}
	return item.toGame(), nil
}

// ResolveGame turns a game reference (GUID or key) into the full row.
func (c *Catalog) ResolveGame(ctx context.Context, idOrKey string) (Game, error) {
	var item gameItem
	found, err := c.getItem(ctx, gamePK(idOrKey), skMeta, &item)
	if err != nil {
		return Game{}, err
	}
	if !found {
		var guard guardItem
		hasGuard, err := c.getItem(ctx, gameKeyPK(idOrKey), skUniq, &guard)
		if err != nil {
			return Game{}, err
		}
		if hasGuard {
			found, err = c.getItem(ctx, gamePK(guard.RefID), skMeta, &item)
			if err != nil {
				return Game{}, err
			}
		}
	}
	if !found {
		return Game{}, fmt.Errorf("%w: game %q", ErrNotFound, idOrKey)
	}
	return item.toGame(), nil
}

// ---------------------------------------------------------------------------
// Sets
// ---------------------------------------------------------------------------

// resolveSet turns a set reference (GUID or key) into the full item.
func (c *Catalog) resolveSet(ctx context.Context, gameID, idOrKey string) (setItem, error) {
	var item setItem
	found, err := c.getItem(ctx, gamePK(gameID), setSK(idOrKey), &item)
	if err != nil {
		return setItem{}, err
	}
	if !found {
		var guard guardItem
		hasGuard, err := c.getItem(ctx, setKeyPK(gameID, idOrKey), skUniq, &guard)
		if err != nil {
			return setItem{}, err
		}
		if hasGuard {
			found, err = c.getItem(ctx, gamePK(gameID), setSK(guard.RefID), &item)
			if err != nil {
				return setItem{}, err
			}
		}
	}
	if !found {
		return setItem{}, fmt.Errorf("%w: set %q", ErrNotFound, idOrKey)
	}
	return item, nil
}

// ListSets returns a game's sets, newest release first (release-less sets
// last), optionally filtered by a case-insensitive name substring.
func (c *Catalog) ListSets(ctx context.Context, gameID, query string) ([]SetSummary, error) {
	items, err := c.queryAll(ctx, &dynamodb.QueryInput{
		TableName:              aws.String(c.Table),
		KeyConditionExpression: aws.String("PK = :pk AND begins_with(SK, :sk)"),
		ExpressionAttributeValues: map[string]ddbtypes.AttributeValue{
			":pk": &ddbtypes.AttributeValueMemberS{Value: gamePK(gameID)},
			":sk": &ddbtypes.AttributeValueMemberS{Value: "SET#"},
		},
	})
	if err != nil {
		return nil, err
	}
	q := strings.ToLower(strings.TrimSpace(query))
	sets := []SetSummary{}
	for _, raw := range items {
		var item setItem
		if err := attributevalue.UnmarshalMap(raw, &item); err != nil {
			return nil, err
		}
		if q != "" && !strings.Contains(strings.ToLower(item.Name), q) {
			continue
		}
		sets = append(sets, item.toSummary())
	}
	sort.SliceStable(sets, func(i, j int) bool {
		a, b := sets[i].ReleaseDate, sets[j].ReleaseDate
		switch {
		case a == nil && b == nil:
			return sets[i].Name < sets[j].Name
		case a == nil:
			return false // nulls last
		case b == nil:
			return true
		case *a != *b:
			return *a > *b // newest first
		default:
			return sets[i].Name < sets[j].Name
		}
	})
	return sets, nil
}

// CreateSet adds a set and returns its GUID.
func (c *Catalog) CreateSet(ctx context.Context, gameID string, in SetInput) (string, error) {
	setKey := strings.TrimSpace(in.Key)
	if setKey == "" {
		return "", fmt.Errorf("%w: set key is required", ErrInvalid)
	}
	releaseDate, err := normReleaseDate(in.ReleaseDate)
	if err != nil {
		return "", err
	}
	lang, err := normLanguage(in.Language)
	if err != nil {
		return "", err
	}
	now := c.now().UTC()
	item := setItem{
		PK: gamePK(gameID), SK: setSK(c.newID()),
		Entity: "set", Key: setKey, Language: lang, GameID: gameID, Name: in.Name,
		ReleaseDate: releaseDate, CardTotal: in.CardTotal, LogoURL: in.LogoURL,
		CreatedAt: now, UpdatedAt: now,
	}
	item.ID = strings.TrimPrefix(item.SK, "SET#")

	setAV, err := marshal(item)
	if err != nil {
		return "", err
	}
	guardAV, err := marshal(guardItem{PK: setKeyPK(gameID, setKey), SK: skUniq, RefID: item.ID})
	if err != nil {
		return "", err
	}
	_, err = c.DB.TransactWriteItems(ctx, &dynamodb.TransactWriteItemsInput{
		TransactItems: []ddbtypes.TransactWriteItem{
			{Put: &ddbtypes.Put{
				TableName: aws.String(c.Table), Item: guardAV,
				ConditionExpression: aws.String("attribute_not_exists(PK)"),
			}},
			{Put: &ddbtypes.Put{TableName: aws.String(c.Table), Item: setAV}},
		},
	})
	if conditionFailed(err) {
		return "", fmt.Errorf("%w: set %q", ErrConflict, setKey)
	}
	if err != nil {
		return "", err
	}
	return item.ID, nil
}

// UpdateSet rewrites a set's fields. The key is immutable; idOrKey accepts
// the GUID or the key.
func (c *Catalog) UpdateSet(ctx context.Context, gameID, idOrKey string, in SetInput) error {
	releaseDate, err := normReleaseDate(in.ReleaseDate)
	if err != nil {
		return err
	}
	existing, err := c.resolveSet(ctx, gameID, idOrKey)
	if err != nil {
		return err
	}
	_, err = c.DB.UpdateItem(ctx, &dynamodb.UpdateItemInput{
		TableName:                aws.String(c.Table),
		Key:                      key(existing.PK, existing.SK),
		ConditionExpression:      aws.String("attribute_exists(PK)"),
		UpdateExpression:         aws.String("SET #name = :name, releaseDate = :rd, cardTotal = :ct, logoUrl = :lu, updatedAt = :ua"),
		ExpressionAttributeNames: map[string]string{"#name": "name"},
		ExpressionAttributeValues: map[string]ddbtypes.AttributeValue{
			":name": &ddbtypes.AttributeValueMemberS{Value: in.Name},
			":rd":   optString(releaseDate),
			":ct":   optInt(in.CardTotal),
			":lu":   optString(in.LogoURL),
			":ua":   &ddbtypes.AttributeValueMemberS{Value: c.now().UTC().Format(time.RFC3339Nano)},
		},
	})
	if conditionFailed(err) {
		return fmt.Errorf("%w: set %q", ErrNotFound, idOrKey)
	}
	return err
}

// DeleteSet removes a set and all of its cards. Returns the number of cards
// removed.
func (c *Catalog) DeleteSet(ctx context.Context, gameID, idOrKey string) (int, error) {
	existing, err := c.resolveSet(ctx, gameID, idOrKey)
	if err != nil {
		return 0, err
	}
	cards, err := c.setCardItems(ctx, existing.ID)
	if err != nil {
		return 0, err
	}
	// Cards go first, in batches of 25 (the BatchWriteItem ceiling); the set
	// and its key guard go last so a failed run stays resumable.
	for start := 0; start < len(cards); start += 25 {
		end := min(start+25, len(cards))
		reqs := make([]ddbtypes.WriteRequest, 0, end-start)
		for _, card := range cards[start:end] {
			reqs = append(reqs, ddbtypes.WriteRequest{
				DeleteRequest: &ddbtypes.DeleteRequest{Key: key(card.PK, card.SK)},
			})
		}
		unprocessed := map[string][]ddbtypes.WriteRequest{c.Table: reqs}
		for len(unprocessed) > 0 {
			res, err := c.DB.BatchWriteItem(ctx, &dynamodb.BatchWriteItemInput{RequestItems: unprocessed})
			if err != nil {
				return 0, err
			}
			unprocessed = res.UnprocessedItems
		}
	}
	_, err = c.DB.TransactWriteItems(ctx, &dynamodb.TransactWriteItemsInput{
		TransactItems: []ddbtypes.TransactWriteItem{
			{Delete: &ddbtypes.Delete{TableName: aws.String(c.Table), Key: key(existing.PK, existing.SK)}},
			{Delete: &ddbtypes.Delete{TableName: aws.String(c.Table), Key: key(setKeyPK(gameID, existing.Key), skUniq)}},
		},
	})
	if err != nil {
		return 0, err
	}
	return len(cards), nil
}

// ---------------------------------------------------------------------------
// Cards (reads)
// ---------------------------------------------------------------------------

func (c *Catalog) setCardItems(ctx context.Context, setID string) ([]cardItem, error) {
	raws, err := c.queryAll(ctx, &dynamodb.QueryInput{
		TableName:              aws.String(c.Table),
		KeyConditionExpression: aws.String("PK = :pk AND begins_with(SK, :sk)"),
		ExpressionAttributeValues: map[string]ddbtypes.AttributeValue{
			":pk": &ddbtypes.AttributeValueMemberS{Value: cardPK(setID)},
			":sk": &ddbtypes.AttributeValueMemberS{Value: "CARD#"},
		},
	})
	if err != nil {
		return nil, err
	}
	cards := make([]cardItem, 0, len(raws))
	for _, raw := range raws {
		var item cardItem
		if err := attributevalue.UnmarshalMap(raw, &item); err != nil {
			return nil, err
		}
		cards = append(cards, item)
	}
	return cards, nil
}

// SearchCards searches a game's catalog by case-insensitive name substring,
// alphabetically, capped at 48 results.
func (c *Catalog) SearchCards(ctx context.Context, gameID, query string) ([]CardSummary, error) {
	q := strings.ToLower(strings.TrimSpace(query))
	if q == "" {
		return []CardSummary{}, nil
	}
	cards := []CardSummary{}
	in := &dynamodb.QueryInput{
		TableName:              aws.String(c.Table),
		IndexName:              aws.String(gsi1),
		KeyConditionExpression: aws.String("GSI1PK = :pk"),
		FilterExpression:       aws.String("contains(nameLower, :q)"),
		ExpressionAttributeValues: map[string]ddbtypes.AttributeValue{
			":pk": &ddbtypes.AttributeValueMemberS{Value: cardGSI1PK(gameID)},
			":q":  &ddbtypes.AttributeValueMemberS{Value: q},
		},
	}
scan:
	for {
		res, err := c.DB.Query(ctx, in)
		if err != nil {
			return nil, err
		}
		for _, raw := range res.Items {
			var item cardItem
			if err := attributevalue.UnmarshalMap(raw, &item); err != nil {
				return nil, err
			}
			cards = append(cards, item.toSummary())
			if len(cards) == searchLimit {
				break scan
			}
		}
		if len(res.LastEvaluatedKey) == 0 {
			break
		}
		in.ExclusiveStartKey = res.LastEvaluatedKey
	}
	// The GSI sort key ("{nameLower}#{id}") is only roughly alphabetical —
	// the separator sorts below space — so finish the ordering here.
	sort.SliceStable(cards, func(i, j int) bool {
		a, b := strings.ToLower(cards[i].Name), strings.ToLower(cards[j].Name)
		if a != b {
			return a < b
		}
		return cards[i].ID < cards[j].ID
	})
	return cards, nil
}

// SetCards returns every card of a set (GUID or key), in collector-number
// order.
func (c *Catalog) SetCards(ctx context.Context, gameID, setIDOrKey string) ([]CardSummary, error) {
	set, err := c.resolveSet(ctx, gameID, setIDOrKey)
	if err != nil {
		return nil, err
	}
	items, err := c.setCardItems(ctx, set.ID)
	if err != nil {
		return nil, err
	}
	cards := make([]CardSummary, 0, len(items))
	for _, item := range items {
		cards = append(cards, item.toSummary())
	}
	sort.SliceStable(cards, func(i, j int) bool {
		ni, si := numberSortKey(cards[i].Number)
		nj, sj := numberSortKey(cards[j].Number)
		if ni != nj {
			return ni < nj
		}
		return si < sj
	})
	return cards, nil
}

var digitsOnly = func(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

// getCardItem fetches one card by GUID or TCGplayer id, scoped to a game.
func (c *Catalog) getCardItem(ctx context.Context, gameID, idOrKey string) (cardItem, error) {
	in := &dynamodb.QueryInput{
		TableName: aws.String(c.Table),
		Limit:     aws.Int32(1),
	}
	if digitsOnly(idOrKey) {
		// A TCGplayer product id; GSI3 is already game-scoped.
		in.IndexName = aws.String(gsi3)
		in.KeyConditionExpression = aws.String("GSI3PK = :pk")
		in.ExpressionAttributeValues = map[string]ddbtypes.AttributeValue{
			":pk": &ddbtypes.AttributeValueMemberS{Value: "TCGP#" + gameID + "#" + idOrKey},
		}
	} else {
		in.IndexName = aws.String(gsi2)
		in.KeyConditionExpression = aws.String("GSI2PK = :pk")
		in.ExpressionAttributeValues = map[string]ddbtypes.AttributeValue{
			":pk": &ddbtypes.AttributeValueMemberS{Value: cardGSI2PK(idOrKey)},
		}
	}
	res, err := c.DB.Query(ctx, in)
	if err != nil {
		return cardItem{}, err
	}
	if len(res.Items) == 0 {
		return cardItem{}, fmt.Errorf("%w: card %q", ErrNotFound, idOrKey)
	}
	var item cardItem
	if err := attributevalue.UnmarshalMap(res.Items[0], &item); err != nil {
		return cardItem{}, err
	}
	if item.GameID != gameID {
		return cardItem{}, fmt.Errorf("%w: card %q", ErrNotFound, idOrKey)
	}
	return item, nil
}

// GetCard fetches one card by GUID or TCGplayer id, with its variants and
// their current prices.
func (c *Catalog) GetCard(ctx context.Context, gameID, idOrKey string) (*CardDetail, error) {
	item, err := c.getCardItem(ctx, gameID, idOrKey)
	if err != nil {
		return nil, err
	}
	detail := item.toDetail()
	return &detail, nil
}

// ---------------------------------------------------------------------------
// Cards (writes)
// ---------------------------------------------------------------------------

func (c *Catalog) buildCardItem(id string, game Game, setID, lang string, in CardInput, variants []CardVariant, createdAt, updatedAt time.Time) (cardItem, error) {
	item := cardItem{
		PK: cardPK(setID), SK: cardSK(id),
		GSI1PK: cardGSI1PK(game.ID),
		GSI1SK: strings.ToLower(in.Name) + "#" + id,
		GSI2PK: cardGSI2PK(id),
		Entity: "card", ID: id, TCGPlayerID: in.TCGPlayerID, Language: lang,
		GameID: game.ID, SetID: setID,
		Name: in.Name, NameLower: strings.ToLower(in.Name),
		Number: in.Number, Rarity: in.Rarity,
		ImageSmall: in.ImageSmall, ImageLarge: in.ImageLarge,
		Variants:  variantsOrEmpty(variants),
		CreatedAt: createdAt, UpdatedAt: updatedAt,
	}
	if in.TCGPlayerID != nil {
		item.GSI3PK = tcgpGSI3PK(game.ID, *in.TCGPlayerID)
	}
	return item, nil
}

// cardCountUpdate bumps a set's card counter by delta inside a transaction.
func (c *Catalog) cardCountUpdate(setID, gameID string, delta int) ddbtypes.TransactWriteItem {
	return ddbtypes.TransactWriteItem{
		Update: &ddbtypes.Update{
			TableName:           aws.String(c.Table),
			Key:                 key(gamePK(gameID), setSK(setID)),
			ConditionExpression: aws.String("attribute_exists(PK)"),
			UpdateExpression:    aws.String("ADD cardCount :d"),
			ExpressionAttributeValues: map[string]ddbtypes.AttributeValue{
				":d": &ddbtypes.AttributeValueMemberN{Value: itoa(delta)},
			},
		},
	}
}

// CreateCard adds a card to a set (GUID or key) and returns the card's GUID.
func (c *Catalog) CreateCard(ctx context.Context, game Game, in CardInput) (string, error) {
	set, err := c.resolveSet(ctx, game.ID, in.SetID)
	if err != nil {
		return "", fmt.Errorf("%w: set %q does not exist in game %q", ErrInvalid, in.SetID, game.Key)
	}
	lang, err := normLanguage(in.Language)
	if err != nil {
		return "", err
	}
	now := c.now().UTC()
	item, err := c.buildCardItem(c.newID(), game, set.ID, lang, in, nil, now, now)
	if err != nil {
		return "", err
	}
	av, err := marshal(item)
	if err != nil {
		return "", err
	}
	_, err = c.DB.TransactWriteItems(ctx, &dynamodb.TransactWriteItemsInput{
		TransactItems: []ddbtypes.TransactWriteItem{
			{Put: &ddbtypes.Put{TableName: aws.String(c.Table), Item: av}},
			c.cardCountUpdate(set.ID, game.ID, 1),
		},
	})
	if conditionFailed(err) {
		return "", fmt.Errorf("%w: set %q does not exist in game %q", ErrInvalid, in.SetID, game.Key)
	}
	if err != nil {
		return "", err
	}
	return item.ID, nil
}

// UpdateCard rewrites a card's fields; idOrKey accepts the GUID or the
// TCGplayer id. Language and variants are preserved.
func (c *Catalog) UpdateCard(ctx context.Context, game Game, idOrKey string, in CardInput) error {
	set, err := c.resolveSet(ctx, game.ID, in.SetID)
	if err != nil {
		return fmt.Errorf("%w: set %q does not exist in game %q", ErrInvalid, in.SetID, game.Key)
	}
	existing, err := c.getCardItem(ctx, game.ID, idOrKey)
	if err != nil {
		return err
	}
	item, err := c.buildCardItem(existing.ID, game, set.ID, existing.Language, in,
		existing.Variants, existing.CreatedAt, c.now().UTC())
	if err != nil {
		return err
	}
	av, err := marshal(item)
	if err != nil {
		return err
	}
	if set.ID == existing.SetID {
		_, err = c.DB.PutItem(ctx, &dynamodb.PutItemInput{TableName: aws.String(c.Table), Item: av})
		return err
	}
	// The card moved sets: relocate the item and shift both counters.
	_, err = c.DB.TransactWriteItems(ctx, &dynamodb.TransactWriteItemsInput{
		TransactItems: []ddbtypes.TransactWriteItem{
			{Delete: &ddbtypes.Delete{TableName: aws.String(c.Table), Key: key(existing.PK, existing.SK)}},
			{Put: &ddbtypes.Put{TableName: aws.String(c.Table), Item: av}},
			c.cardCountUpdate(existing.SetID, game.ID, -1),
			c.cardCountUpdate(set.ID, game.ID, 1),
		},
	})
	return err
}

// DeleteCard removes a card.
func (c *Catalog) DeleteCard(ctx context.Context, gameID, idOrKey string) error {
	existing, err := c.getCardItem(ctx, gameID, idOrKey)
	if err != nil {
		return err
	}
	_, err = c.DB.TransactWriteItems(ctx, &dynamodb.TransactWriteItemsInput{
		TransactItems: []ddbtypes.TransactWriteItem{
			{Delete: &ddbtypes.Delete{TableName: aws.String(c.Table), Key: key(existing.PK, existing.SK)}},
			c.cardCountUpdate(existing.SetID, gameID, -1),
		},
	})
	return err
}

// ---------------------------------------------------------------------------
// Variant prices (used by the price-updater jobs)
// ---------------------------------------------------------------------------

// PricedCard is a card that can receive external price updates.
type PricedCard struct {
	ID          string `json:"id"`
	TCGPlayerID int    `json:"tcgplayerId"`
}

// PricedCards lists a game's cards that carry a TCGplayer id, i.e. the ones
// a price feed can quote.
func (c *Catalog) PricedCards(ctx context.Context, gameID string) ([]PricedCard, error) {
	items, err := c.queryAll(ctx, &dynamodb.QueryInput{
		TableName:              aws.String(c.Table),
		IndexName:              aws.String(gsi1),
		KeyConditionExpression: aws.String("GSI1PK = :pk"),
		FilterExpression:       aws.String("attribute_exists(tcgplayerId)"),
		ExpressionAttributeValues: map[string]ddbtypes.AttributeValue{
			":pk": &ddbtypes.AttributeValueMemberS{Value: cardGSI1PK(gameID)},
		},
	})
	if err != nil {
		return nil, err
	}
	cards := []PricedCard{}
	for _, raw := range items {
		var item cardItem
		if err := attributevalue.UnmarshalMap(raw, &item); err != nil {
			return nil, err
		}
		if item.TCGPlayerID != nil {
			cards = append(cards, PricedCard{ID: item.ID, TCGPlayerID: *item.TCGPlayerID})
		}
	}
	return cards, nil
}

// SetCardPrices overwrites variant prices on one card (looked up by
// TCGplayer id). Prices for variant names the card already has are updated
// in place; unknown names become new variants. Prices are overwritten, never
// versioned — the catalog keeps no price history.
func (c *Catalog) SetCardPrices(ctx context.Context, gameID string, tcgplayerID int, prices map[string]float64) error {
	if len(prices) == 0 {
		return nil
	}
	item, err := c.getCardItem(ctx, gameID, itoa(tcgplayerID))
	if err != nil {
		return err
	}
	existing := make(map[string]int, len(item.Variants)) // variant name → index
	for i, v := range item.Variants {
		existing[strings.ToLower(v.Name)] = i
	}
	for name, price := range prices {
		p := price
		if i, ok := existing[strings.ToLower(name)]; ok {
			item.Variants[i].Price = &p
		} else {
			item.Variants = append(item.Variants, CardVariant{ID: c.newID(), Name: name, Price: &p})
		}
	}
	sort.Slice(item.Variants, func(i, j int) bool { return item.Variants[i].Name < item.Variants[j].Name })
	item.UpdatedAt = c.now().UTC()
	av, err := marshal(item)
	if err != nil {
		return err
	}
	_, err = c.DB.PutItem(ctx, &dynamodb.PutItemInput{TableName: aws.String(c.Table), Item: av})
	return err
}

func optString(v *string) ddbtypes.AttributeValue {
	if v == nil {
		return &ddbtypes.AttributeValueMemberNULL{Value: true}
	}
	return &ddbtypes.AttributeValueMemberS{Value: *v}
}

func optInt(v *int) ddbtypes.AttributeValue {
	if v == nil {
		return &ddbtypes.AttributeValueMemberNULL{Value: true}
	}
	return &ddbtypes.AttributeValueMemberN{Value: itoa(*v)}
}
