// Package graphql is the public read surface of the catalog. It exposes
// queries only — every write goes through the trusted gRPC API
// (internal/rpc), so this endpoint can be exposed openly.
package graphql

// Schema is the SDL contract for public consumers.
const Schema = `
	schema {
		query: Query
	}

	type Query {
		"All games this instance tracks"
		games: [Game!]!

		"Sets for a game with live card counts, newest release first; optional name filter"
		sets(game: String!, query: String): [Set!]!

		"Every card in a set (GUID or key), in collector-number order"
		setCards(game: String!, setId: String!): [Card!]!

		"Search a game's cards by name, newest set first (max 48); min 2 characters"
		searchCards(game: String!, query: String!): [Card!]!

		"One card by GUID or TCGplayer id, with current prices; null when not found"
		card(game: String!, id: String!): Card
	}

	type Game {
		"GUID"
		id: ID!
		"Immutable routing key, unique (e.g. \"pokemon\")"
		key: String!
		"ISO 639 alpha-3 (\"eng\", \"jpn\")"
		language: String!
		label: String!
		"RFC 3339"
		updatedAt: String!
	}

	type Set {
		"GUID"
		id: ID!
		"Catalog set id (e.g. \"sv3pt5\"), unique per game"
		key: String!
		"ISO 639 alpha-3 (\"eng\", \"jpn\")"
		language: String!
		gameId: ID!
		name: String!
		"Live count of cards in the catalog — never stored"
		cardCount: Int!
		"YYYY-MM-DD"
		releaseDate: String
		"Official printed set size"
		cardTotal: Int
		logoUrl: String
		"RFC 3339"
		createdAt: String!
		"RFC 3339"
		updatedAt: String!
	}

	type Card {
		"GUID"
		id: ID!
		"TCGplayer product id; null when not listed there"
		tcgplayerId: Int
		"ISO 639 alpha-3 (\"eng\", \"jpn\")"
		language: String!
		name: String!
		number: String!
		rarity: String
		"Set GUID — full set details come from the sets query"
		setId: ID!
		image: String
		imageLarge: String
		"The printings this card exists in, with their current prices"
		variants: [CardVariant!]!
		"RFC 3339"
		createdAt: String!
		"When the card itself last changed (RFC 3339)"
		updatedAt: String!
	}

	"One printing a card exists in (\"Normal\", \"Holofoil\", \"1st Edition\", ...)"
	type CardVariant {
		"GUID"
		id: ID!
		name: String!
		price: Float
	}
`
