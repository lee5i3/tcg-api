package catalog

// Single-table DynamoDB layout. One table holds every entity; items are
// addressed by composite (PK, SK) and reachable through three GSIs:
//
//	Entity       PK                     SK           GSI1PK               GSI1SK                GSI2PK        GSI3PK
//	game         GAME#{id}              META         GAME                 {createdAt}#{id}      —             —
//	game guard   GAMEKEY#{key}          UNIQ         —                    —                     —             —
//	set          GAME#{gameId}          SET#{id}     —                    —                     —             —
//	set guard    SETKEY#{gameId}/{key}  UNIQ         —                    —                     —             —
//	card         SET#{setId}            CARD#{id}    GAMECARDS#{gameId}   {nameLower}#{id}      CARD#{id}     TCGP#{gameId}#{tcgplayerId}
//
// Guard items make catalog keys unique (DynamoDB can't enforce uniqueness on
// a GSI): creates put the entity and its guard in one transaction, each
// conditioned on attribute_not_exists. The guard carries refId, so it doubles
// as the key→GUID lookup. GSI3 is sparse — only cards listed on TCGplayer
// carry it. Card variants are embedded on the card item (they were only ever
// read as an aggregate). See docs/decisions/0002-dynamodb-single-table.md.

import (
	"time"

	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	ddbtypes "github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

const (
	skMeta  = "META"
	skUniq  = "UNIQ"
	gsi1    = "GSI1"
	gsi2    = "GSI2"
	gsi3    = "GSI3"
	gamesPK = "GAME" // GSI1 partition holding every game
)

func gamePK(id string) string            { return "GAME#" + id }
func gameKeyPK(key string) string        { return "GAMEKEY#" + key }
func setSK(id string) string             { return "SET#" + id }
func setKeyPK(gameID, key string) string { return "SETKEY#" + gameID + "/" + key }
func cardPK(setID string) string         { return "SET#" + setID }
func cardSK(id string) string            { return "CARD#" + id }
func cardGSI1PK(gameID string) string    { return "GAMECARDS#" + gameID }
func cardGSI2PK(id string) string        { return "CARD#" + id }
func tcgpGSI3PK(gameID string, tcgplayerID int) string {
	return "TCGP#" + gameID + "#" + itoa(tcgplayerID)
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}

type gameItem struct {
	PK        string    `dynamodbav:"PK"`
	SK        string    `dynamodbav:"SK"`
	GSI1PK    string    `dynamodbav:"GSI1PK"`
	GSI1SK    string    `dynamodbav:"GSI1SK"`
	Entity    string    `dynamodbav:"entity"`
	ID        string    `dynamodbav:"id"`
	Key       string    `dynamodbav:"key"`
	Language  string    `dynamodbav:"language"`
	Label     string    `dynamodbav:"label"`
	CreatedAt time.Time `dynamodbav:"createdAt"`
	UpdatedAt time.Time `dynamodbav:"updatedAt"`
}

// guardItem reserves a catalog key and points back at the entity owning it.
type guardItem struct {
	PK    string `dynamodbav:"PK"`
	SK    string `dynamodbav:"SK"`
	RefID string `dynamodbav:"refId"`
}

type setItem struct {
	PK          string    `dynamodbav:"PK"`
	SK          string    `dynamodbav:"SK"`
	Entity      string    `dynamodbav:"entity"`
	ID          string    `dynamodbav:"id"`
	Key         string    `dynamodbav:"key"`
	Language    string    `dynamodbav:"language"`
	GameID      string    `dynamodbav:"gameId"`
	Name        string    `dynamodbav:"name"`
	CardCount   int       `dynamodbav:"cardCount"`
	ReleaseDate *string   `dynamodbav:"releaseDate"`
	CardTotal   *int      `dynamodbav:"cardTotal"`
	LogoURL     *string   `dynamodbav:"logoUrl"`
	CreatedAt   time.Time `dynamodbav:"createdAt"`
	UpdatedAt   time.Time `dynamodbav:"updatedAt"`
}

type cardItem struct {
	PK          string        `dynamodbav:"PK"`
	SK          string        `dynamodbav:"SK"`
	GSI1PK      string        `dynamodbav:"GSI1PK"`
	GSI1SK      string        `dynamodbav:"GSI1SK"`
	GSI2PK      string        `dynamodbav:"GSI2PK"`
	GSI3PK      string        `dynamodbav:"GSI3PK,omitempty"` // sparse: only with a TCGplayer id
	Entity      string        `dynamodbav:"entity"`
	ID          string        `dynamodbav:"id"`
	TCGPlayerID *int          `dynamodbav:"tcgplayerId,omitempty"` // absent (not NULL) when unlisted, so attribute_exists filters work
	Language    string        `dynamodbav:"language"`
	GameID      string        `dynamodbav:"gameId"`
	SetID       string        `dynamodbav:"setId"`
	Name        string        `dynamodbav:"name"`
	NameLower   string        `dynamodbav:"nameLower"` // lowercased for case-insensitive search
	Number      *string       `dynamodbav:"number"`
	Rarity      *string       `dynamodbav:"rarity"`
	ImageSmall  *string       `dynamodbav:"imageSmall"`
	ImageLarge  *string       `dynamodbav:"imageLarge"`
	Variants    []CardVariant `dynamodbav:"variants"`
	CreatedAt   time.Time     `dynamodbav:"createdAt"`
	UpdatedAt   time.Time     `dynamodbav:"updatedAt"`
}

func (g gameItem) toGame() Game {
	return Game{ID: g.ID, Key: g.Key, Language: g.Language, Label: g.Label, UpdatedAt: g.UpdatedAt}
}

func (s setItem) toSummary() SetSummary {
	return SetSummary{
		ID: s.ID, Key: s.Key, Language: s.Language, GameID: s.GameID, Name: s.Name,
		CardCount: s.CardCount, ReleaseDate: s.ReleaseDate, CardTotal: s.CardTotal,
		LogoURL: s.LogoURL, CreatedAt: s.CreatedAt, UpdatedAt: s.UpdatedAt,
	}
}

func (c cardItem) toSummary() CardSummary {
	number := ""
	if c.Number != nil {
		number = *c.Number
	}
	return CardSummary{
		ID: c.ID, TCGPlayerID: c.TCGPlayerID, Language: c.Language, Name: c.Name,
		Number: number, Rarity: c.Rarity, SetID: c.SetID,
		Image: c.ImageSmall, ImageLarge: c.ImageLarge,
		Variants: variantsOrEmpty(c.Variants), CreatedAt: c.CreatedAt, UpdatedAt: c.UpdatedAt,
	}
}

func (c cardItem) toDetail() CardDetail {
	return CardDetail{
		ID: c.ID, TCGPlayerID: c.TCGPlayerID, Language: c.Language, GameID: c.GameID,
		Name: c.Name, Number: c.Number, Rarity: c.Rarity, Set: c.SetID,
		Images:   CardImages{Small: c.ImageSmall, Large: c.ImageLarge},
		Variants: variantsOrEmpty(c.Variants), CreatedAt: c.CreatedAt, UpdatedAt: c.UpdatedAt,
	}
}

func variantsOrEmpty(v []CardVariant) []CardVariant {
	if v == nil {
		return []CardVariant{}
	}
	return v
}

// key builds a DynamoDB primary key attribute map.
func key(pk, sk string) map[string]ddbtypes.AttributeValue {
	return map[string]ddbtypes.AttributeValue{
		"PK": &ddbtypes.AttributeValueMemberS{Value: pk},
		"SK": &ddbtypes.AttributeValueMemberS{Value: sk},
	}
}

func marshal(v any) (map[string]ddbtypes.AttributeValue, error) {
	return attributevalue.MarshalMap(v)
}
