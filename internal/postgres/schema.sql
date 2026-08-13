-- Card-catalog schema. Applied idempotently at boot.
--
-- Design rules carried over from the original data model:
--   * prices live on card_variants (per printing) and sealed products —
--     nothing is stored on the card row itself
--   * set sizes are never stored — always a live count of the cards table
--   * every entity has a GUID primary key (first column); games and sets keep
--     the human catalog id ("pokemon", "sv3pt5") as `key`, immutable, and
--     lookups accept either; cards carry an optional tcgplayer_id instead,
--     and card lookups accept the GUID or that id
--   * catalog entities carry a language (ISO 639 alpha-3: "eng", "jpn");
--     English and Japanese catalogs are entirely separate rows, never
--     translations of one another

CREATE TABLE IF NOT EXISTS games (
    id         uuid PRIMARY KEY,
    key        text NOT NULL UNIQUE, -- immutable routing key (e.g. "pokemon")
    language   text NOT NULL DEFAULT 'eng', -- ISO 639 alpha-3
    label      text NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);

-- Upgrade pre-GUID databases where key was the primary key. Postgres can't
-- reorder columns, so the table is rebuilt with id as the FIRST column; the
-- child FKs depend on games' indexes and are dropped and re-added around the
-- swap. Both steps are no-ops on databases already in shape.
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns
                   WHERE table_name = 'games' AND column_name = 'id') THEN
        ALTER TABLE games ADD COLUMN id uuid;
        UPDATE games SET id = gen_random_uuid();
    END IF;

    IF NOT EXISTS (SELECT 1 FROM information_schema.columns
                   WHERE table_name = 'games' AND column_name = 'id'
                     AND ordinal_position = 1) THEN
        CREATE TABLE games_rebuilt (
            id         uuid PRIMARY KEY,
            key        text NOT NULL UNIQUE,
            label      text NOT NULL,
            created_at timestamptz NOT NULL DEFAULT now(),
            updated_at timestamptz NOT NULL DEFAULT now()
        );
        INSERT INTO games_rebuilt (id, key, label, created_at)
        SELECT id, key, label, created_at FROM games;

        ALTER TABLE card_sets DROP CONSTRAINT card_sets_game_fkey;
        ALTER TABLE cards DROP CONSTRAINT cards_game_fkey;
        DROP TABLE games;
        ALTER TABLE games_rebuilt RENAME TO games;
        ALTER INDEX games_rebuilt_pkey RENAME TO games_pkey;
        ALTER TABLE games RENAME CONSTRAINT games_rebuilt_key_key TO games_key_key;
        ALTER TABLE card_sets ADD CONSTRAINT card_sets_game_fkey
            FOREIGN KEY (game) REFERENCES games (key) ON DELETE CASCADE;
        ALTER TABLE cards ADD CONSTRAINT cards_game_fkey
            FOREIGN KEY (game) REFERENCES games (key) ON DELETE CASCADE;
    END IF;
END
$$;

-- Bring pre-existing games tables to the canonical column order. New columns
-- can only be appended, and Postgres can't reorder columns, so when the shape
-- doesn't match, the table is rebuilt with child FKs dropped and re-added
-- around the swap. No-op once in shape.
ALTER TABLE games ADD COLUMN IF NOT EXISTS updated_at timestamptz NOT NULL DEFAULT now();
ALTER TABLE games ADD COLUMN IF NOT EXISTS language text NOT NULL DEFAULT 'eng';

DO $$
BEGIN
    IF (SELECT array_agg(column_name::text ORDER BY ordinal_position)
        FROM information_schema.columns WHERE table_name = 'games')
       IS DISTINCT FROM
       ARRAY['id', 'key', 'language', 'label', 'created_at', 'updated_at'] THEN
        CREATE TABLE games_rebuilt (
            id         uuid PRIMARY KEY,
            key        text NOT NULL UNIQUE,
            language   text NOT NULL DEFAULT 'eng',
            label      text NOT NULL,
            created_at timestamptz NOT NULL DEFAULT now(),
            updated_at timestamptz NOT NULL DEFAULT now()
        );
        INSERT INTO games_rebuilt (id, key, language, label, created_at, updated_at)
        SELECT id, key, language, label, created_at, updated_at FROM games;

        IF to_regclass('sets') IS NOT NULL THEN
            ALTER TABLE sets DROP CONSTRAINT IF EXISTS sets_game_id_fkey;
        END IF;
        IF to_regclass('cards') IS NOT NULL THEN
            ALTER TABLE cards DROP CONSTRAINT IF EXISTS cards_game_id_fkey;
            -- legacy key-based FK; the cards rebuild further down re-shapes it
            ALTER TABLE cards DROP CONSTRAINT IF EXISTS cards_game_fkey;
        END IF;
        IF to_regclass('card_sets') IS NOT NULL THEN
            ALTER TABLE card_sets DROP CONSTRAINT IF EXISTS card_sets_game_fkey;
        END IF;
        IF to_regclass('sealed') IS NOT NULL THEN
            ALTER TABLE sealed DROP CONSTRAINT IF EXISTS sealed_game_id_fkey;
        END IF;
        DROP TABLE games;
        ALTER TABLE games_rebuilt RENAME TO games;
        ALTER INDEX games_rebuilt_pkey RENAME TO games_pkey;
        ALTER TABLE games RENAME CONSTRAINT games_rebuilt_key_key TO games_key_key;
        IF to_regclass('sets') IS NOT NULL THEN
            ALTER TABLE sets ADD CONSTRAINT sets_game_id_fkey
                FOREIGN KEY (game_id) REFERENCES games (id) ON DELETE CASCADE;
        END IF;
        IF EXISTS (SELECT 1 FROM information_schema.columns
                   WHERE table_name = 'cards' AND column_name = 'game_id') THEN
            ALTER TABLE cards ADD CONSTRAINT cards_game_id_fkey
                FOREIGN KEY (game_id) REFERENCES games (id) ON DELETE CASCADE;
        END IF;
        IF to_regclass('sealed') IS NOT NULL THEN
            ALTER TABLE sealed ADD CONSTRAINT sealed_game_id_fkey
                FOREIGN KEY (game_id) REFERENCES games (id) ON DELETE CASCADE;
        END IF;
    END IF;
END
$$;

CREATE TABLE IF NOT EXISTS sets (
    id           uuid PRIMARY KEY,
    key          text NOT NULL, -- catalog set id (e.g. "sv3pt5"), immutable
    language     text NOT NULL DEFAULT 'eng', -- ISO 639 alpha-3
    game_id      uuid NOT NULL REFERENCES games (id) ON DELETE CASCADE,
    name         text NOT NULL,
    release_date date,
    card_total   integer, -- official printed set size; cardCount stays a live count
    logo_url     text,
    created_at   timestamptz NOT NULL DEFAULT now(),
    updated_at   timestamptz NOT NULL DEFAULT now(),
    UNIQUE (game_id, key)
);

CREATE INDEX IF NOT EXISTS sets_game_idx ON sets (game_id);

-- The set-symbol columns were retired; drop them from databases that still
-- carry them (under any of their historical names).
ALTER TABLE sets
    DROP COLUMN IF EXISTS symbol_text,
    DROP COLUMN IF EXISTS symbol_image,
    DROP COLUMN IF EXISTS symbol,
    DROP COLUMN IF EXISTS symbol_url;

-- The series grouping was retired (too Pokémon-specific for a multi-game
-- catalog); drop the sets pointer and the table itself.
ALTER TABLE sets DROP COLUMN IF EXISTS series_id;
DROP TABLE IF EXISTS series;

-- Bring pre-existing sets tables to the canonical column order. New columns
-- can only be appended, and Postgres can't reorder columns, so when the shape
-- doesn't match, the table is rebuilt with child FKs dropped and re-added
-- around the swap. No-op once in shape.
ALTER TABLE sets ADD COLUMN IF NOT EXISTS card_total integer;
ALTER TABLE sets ADD COLUMN IF NOT EXISTS language text NOT NULL DEFAULT 'eng';

DO $$
BEGIN
    IF (SELECT array_agg(column_name::text ORDER BY ordinal_position)
        FROM information_schema.columns WHERE table_name = 'sets')
       IS DISTINCT FROM
       ARRAY['id', 'key', 'language', 'game_id', 'name', 'release_date',
             'card_total', 'logo_url', 'created_at', 'updated_at'] THEN
        CREATE TABLE sets_rebuilt (
            id           uuid PRIMARY KEY,
            key          text NOT NULL,
            language     text NOT NULL DEFAULT 'eng',
            game_id      uuid NOT NULL REFERENCES games (id) ON DELETE CASCADE,
            name         text NOT NULL,
            release_date date,
            card_total   integer,
            logo_url     text,
            created_at   timestamptz NOT NULL DEFAULT now(),
            updated_at   timestamptz NOT NULL DEFAULT now(),
            UNIQUE (game_id, key)
        );
        INSERT INTO sets_rebuilt (id, key, language, game_id, name,
                                  release_date, card_total, logo_url,
                                  created_at, updated_at)
        SELECT id, key, language, game_id, name,
               release_date, card_total, logo_url,
               created_at, updated_at
        FROM sets;

        IF to_regclass('cards') IS NOT NULL THEN
            ALTER TABLE cards DROP CONSTRAINT IF EXISTS cards_set_id_fkey;
        END IF;
        IF to_regclass('sealed') IS NOT NULL THEN
            ALTER TABLE sealed DROP CONSTRAINT IF EXISTS sealed_set_id_fkey;
        END IF;
        DROP TABLE sets;
        ALTER TABLE sets_rebuilt RENAME TO sets;
        ALTER INDEX sets_rebuilt_pkey RENAME TO sets_pkey;
        ALTER TABLE sets RENAME CONSTRAINT sets_rebuilt_game_id_key_key TO sets_game_id_key_key;
        ALTER TABLE sets RENAME CONSTRAINT sets_rebuilt_game_id_fkey TO sets_game_id_fkey;
        IF EXISTS (SELECT 1 FROM information_schema.columns
                   WHERE table_name = 'cards' AND column_name = 'set_id'
                     AND data_type = 'uuid') THEN
            ALTER TABLE cards ADD CONSTRAINT cards_set_id_fkey
                FOREIGN KEY (set_id) REFERENCES sets (id) ON DELETE CASCADE;
        END IF;
        IF to_regclass('sealed') IS NOT NULL THEN
            ALTER TABLE sealed ADD CONSTRAINT sealed_set_id_fkey
                FOREIGN KEY (set_id) REFERENCES sets (id) ON DELETE SET NULL;
        END IF;
        CREATE INDEX sets_game_idx ON sets (game_id);
    END IF;
END
$$;

-- Sealed products: booster boxes, bundles, packs, tins, blisters, binder
-- sets, and the like. set_id is null for products not tied to one set;
-- deleting a set detaches its products rather than removing them.
CREATE TABLE IF NOT EXISTS sealed (
    id           uuid PRIMARY KEY,
    set_id       uuid REFERENCES sets (id) ON DELETE SET NULL,
    game_id      uuid NOT NULL REFERENCES games (id) ON DELETE CASCADE,
    language     text NOT NULL DEFAULT 'eng', -- ISO 639 alpha-3
    name         text NOT NULL,
    image_url    text,
    price        numeric,
    release_date date,
    created_at   timestamptz NOT NULL DEFAULT now(),
    updated_at   timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS sealed_game_idx ON sealed (game_id);
CREATE INDEX IF NOT EXISTS sealed_set_idx ON sealed (set_id);

-- Bring pre-existing sealed tables to the canonical column order. New columns
-- can only be appended, and Postgres can't reorder columns, so when the shape
-- doesn't match, the table is rebuilt. Nothing references sealed, so the swap
-- is self-contained. No-op once in shape.
ALTER TABLE sealed ADD COLUMN IF NOT EXISTS image_url text;
ALTER TABLE sealed ADD COLUMN IF NOT EXISTS language text NOT NULL DEFAULT 'eng';
ALTER TABLE sealed ADD COLUMN IF NOT EXISTS price numeric;

DO $$
BEGIN
    IF (SELECT array_agg(column_name::text ORDER BY ordinal_position)
        FROM information_schema.columns WHERE table_name = 'sealed')
       IS DISTINCT FROM
       ARRAY['id', 'set_id', 'game_id', 'language', 'name', 'image_url',
             'price', 'release_date', 'created_at', 'updated_at'] THEN
        CREATE TABLE sealed_rebuilt (
            id           uuid PRIMARY KEY,
            set_id       uuid REFERENCES sets (id) ON DELETE SET NULL,
            game_id      uuid NOT NULL REFERENCES games (id) ON DELETE CASCADE,
            language     text NOT NULL DEFAULT 'eng',
            name         text NOT NULL,
            image_url    text,
            price        numeric,
            release_date date,
            created_at   timestamptz NOT NULL DEFAULT now(),
            updated_at   timestamptz NOT NULL DEFAULT now()
        );
        INSERT INTO sealed_rebuilt (id, set_id, game_id, language, name, image_url,
                                    price, release_date, created_at, updated_at)
        SELECT id, set_id, game_id, language, name, image_url,
               price, release_date, created_at, updated_at
        FROM sealed;

        DROP TABLE sealed;
        ALTER TABLE sealed_rebuilt RENAME TO sealed;
        ALTER INDEX sealed_rebuilt_pkey RENAME TO sealed_pkey;
        ALTER TABLE sealed RENAME CONSTRAINT sealed_rebuilt_set_id_fkey TO sealed_set_id_fkey;
        ALTER TABLE sealed RENAME CONSTRAINT sealed_rebuilt_game_id_fkey TO sealed_game_id_fkey;
        CREATE INDEX sealed_game_idx ON sealed (game_id);
        CREATE INDEX sealed_set_idx ON sealed (set_id);
    END IF;
END
$$;

-- Migrate the retired card_sets table: its text catalog id becomes `key`
-- with a fresh GUID id, `game` (key) becomes game_id (GUID), and release_date
-- becomes a real date (the series text is retired and not carried over).
-- cards.set_id follows along, re-pointed from the catalog id to the GUID.
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM information_schema.tables
               WHERE table_name = 'card_sets') THEN
        INSERT INTO sets (id, key, game_id, name, release_date,
                          logo_url, updated_at)
        SELECT gen_random_uuid(), cs.id, g.id, cs.name,
               to_date(cs.release_date, 'YYYY/MM/DD'),
               cs.logo_url, cs.updated_at
        FROM card_sets cs
        JOIN games g ON g.key = cs.game
        ON CONFLICT (game_id, key) DO NOTHING;

        ALTER TABLE cards DROP CONSTRAINT cards_set_id_fkey;
        ALTER TABLE cards ADD COLUMN set_uuid uuid;
        UPDATE cards SET set_uuid = s.id
        FROM sets s
        JOIN games g ON g.id = s.game_id
        WHERE s.key = cards.set_id AND g.key = cards.game;
        ALTER TABLE cards DROP COLUMN set_id;
        ALTER TABLE cards RENAME COLUMN set_uuid TO set_id;
        ALTER TABLE cards ALTER COLUMN set_id SET NOT NULL;
        ALTER TABLE cards ADD CONSTRAINT cards_set_id_fkey
            FOREIGN KEY (set_id) REFERENCES sets (id) ON DELETE CASCADE;

        DROP TABLE card_sets;
    END IF;
END
$$;

CREATE TABLE IF NOT EXISTS cards (
    id           uuid PRIMARY KEY,
    tcgplayer_id integer, -- TCGplayer product id; null when not listed there
    language     text NOT NULL DEFAULT 'eng', -- ISO 639 alpha-3
    game_id      uuid NOT NULL REFERENCES games (id) ON DELETE CASCADE,
    set_id       uuid NOT NULL REFERENCES sets (id) ON DELETE CASCADE,
    name         text NOT NULL,
    number       text,
    rarity       text,
    image_small  text,
    image_large  text,
    created_at   timestamptz NOT NULL DEFAULT now(),
    updated_at   timestamptz NOT NULL DEFAULT now()
);

-- Rebuild pre-game_id cards tables: `game` (games.key) becomes game_id
-- (GUID) placed before set_id, created_at is added, and the price_id
-- pointer is dropped (the latest snapshot is found by captured_at instead).
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM information_schema.columns
               WHERE table_name = 'cards' AND column_name = 'game') THEN
        CREATE TABLE cards_rebuilt (
            id          uuid PRIMARY KEY,
            key         text NOT NULL,
            game_id     uuid NOT NULL REFERENCES games (id) ON DELETE CASCADE,
            set_id      uuid NOT NULL REFERENCES sets (id) ON DELETE CASCADE,
            name        text NOT NULL,
            number      text,
            rarity      text,
            image_small text,
            image_large text,
            created_at  timestamptz NOT NULL DEFAULT now(),
            updated_at  timestamptz NOT NULL DEFAULT now(),
            UNIQUE (game_id, key)
        );
        -- created_at didn't exist before this shape; the row's updated_at is
        -- the closest available approximation.
        INSERT INTO cards_rebuilt (id, key, game_id, set_id, name, number, rarity,
                                   image_small, image_large, created_at, updated_at)
        SELECT c.id, c.key, g.id, c.set_id, c.name, c.number, c.rarity,
               c.image_small, c.image_large, c.updated_at, c.updated_at
        FROM cards c
        JOIN games g ON g.key = c.game;

        ALTER TABLE card_prices DROP CONSTRAINT card_prices_card_id_fkey;
        DROP TABLE cards;
        ALTER TABLE cards_rebuilt RENAME TO cards;
        ALTER INDEX cards_rebuilt_pkey RENAME TO cards_pkey;
        ALTER TABLE cards RENAME CONSTRAINT cards_rebuilt_game_id_key_key TO cards_game_id_key_key;
        ALTER TABLE cards RENAME CONSTRAINT cards_rebuilt_game_id_fkey TO cards_game_id_fkey;
        ALTER TABLE cards RENAME CONSTRAINT cards_rebuilt_set_id_fkey TO cards_set_id_fkey;
        ALTER TABLE card_prices ADD CONSTRAINT card_prices_card_id_fkey
            FOREIGN KEY (card_id) REFERENCES cards (id) ON DELETE CASCADE;
    END IF;
END
$$;

-- Price history was retired: prices now live directly on card_variants and
-- sealed, overwritten in place.
DROP TABLE IF EXISTS card_prices;

-- Bring pre-existing cards tables to the canonical column order (the retired
-- catalog `key` is dropped along the way). New columns can only be appended,
-- and Postgres can't reorder columns, so when the shape doesn't match, the
-- table is rebuilt with the referencing FKs dropped and re-added around the
-- swap. No-op once in shape.
ALTER TABLE cards ADD COLUMN IF NOT EXISTS language text NOT NULL DEFAULT 'eng';
ALTER TABLE cards ADD COLUMN IF NOT EXISTS tcgplayer_id integer;

DO $$
BEGIN
    IF (SELECT array_agg(column_name::text ORDER BY ordinal_position)
        FROM information_schema.columns WHERE table_name = 'cards')
       IS DISTINCT FROM
       ARRAY['id', 'tcgplayer_id', 'language', 'game_id', 'set_id', 'name', 'number',
             'rarity', 'image_small', 'image_large', 'created_at', 'updated_at'] THEN
        CREATE TABLE cards_rebuilt (
            id           uuid PRIMARY KEY,
            tcgplayer_id integer,
            language     text NOT NULL DEFAULT 'eng',
            game_id      uuid NOT NULL REFERENCES games (id) ON DELETE CASCADE,
            set_id       uuid NOT NULL REFERENCES sets (id) ON DELETE CASCADE,
            name         text NOT NULL,
            number       text,
            rarity       text,
            image_small  text,
            image_large  text,
            created_at   timestamptz NOT NULL DEFAULT now(),
            updated_at   timestamptz NOT NULL DEFAULT now()
        );
        INSERT INTO cards_rebuilt (id, tcgplayer_id, language, game_id, set_id, name, number,
                                   rarity, image_small, image_large, created_at, updated_at)
        SELECT id, tcgplayer_id, language, game_id, set_id, name, number,
               rarity, image_small, image_large, created_at, updated_at
        FROM cards;

        IF to_regclass('card_variants') IS NOT NULL THEN
            ALTER TABLE card_variants DROP CONSTRAINT IF EXISTS card_variants_card_id_fkey;
        END IF;
        IF to_regclass('variants') IS NOT NULL THEN
            ALTER TABLE variants DROP CONSTRAINT IF EXISTS variants_card_id_fkey;
        END IF;
        DROP TABLE cards;
        ALTER TABLE cards_rebuilt RENAME TO cards;
        ALTER INDEX cards_rebuilt_pkey RENAME TO cards_pkey;
        ALTER TABLE cards RENAME CONSTRAINT cards_rebuilt_game_id_fkey TO cards_game_id_fkey;
        ALTER TABLE cards RENAME CONSTRAINT cards_rebuilt_set_id_fkey TO cards_set_id_fkey;
        IF to_regclass('card_variants') IS NOT NULL THEN
            ALTER TABLE card_variants ADD CONSTRAINT card_variants_card_id_fkey
                FOREIGN KEY (card_id) REFERENCES cards (id) ON DELETE CASCADE;
        END IF;
        IF to_regclass('variants') IS NOT NULL THEN
            ALTER TABLE variants ADD CONSTRAINT variants_card_id_fkey
                FOREIGN KEY (card_id) REFERENCES cards (id) ON DELETE CASCADE;
        END IF;
    END IF;
END
$$;

CREATE INDEX IF NOT EXISTS cards_set_idx ON cards (set_id);
CREATE INDEX IF NOT EXISTS cards_game_name_idx ON cards (game_id, name);
CREATE INDEX IF NOT EXISTS cards_tcgplayer_idx ON cards (tcgplayer_id);

-- Rename pass for databases created while card_variants was called variants;
-- the constraint names follow the table.
DO $$
BEGIN
    IF to_regclass('variants') IS NOT NULL AND to_regclass('card_variants') IS NULL THEN
        ALTER TABLE variants RENAME TO card_variants;
        ALTER INDEX variants_pkey RENAME TO card_variants_pkey;
        ALTER TABLE card_variants RENAME CONSTRAINT variants_card_id_fkey TO card_variants_card_id_fkey;
        ALTER TABLE card_variants RENAME CONSTRAINT variants_card_id_name_key TO card_variants_card_id_name_key;
    END IF;
END
$$;

-- Card variants ("Normal", "Holofoil", "Reverse Holofoil", "Cold Foil",
-- "Unlimited", "1st Edition", ...). One row per printing a card exists in,
-- carrying that printing's current price.
CREATE TABLE IF NOT EXISTS card_variants (
    id      uuid PRIMARY KEY,
    card_id uuid NOT NULL REFERENCES cards (id) ON DELETE CASCADE,
    name    text NOT NULL,
    price   numeric,
    UNIQUE (card_id, name)
);

ALTER TABLE card_variants ADD COLUMN IF NOT EXISTS price numeric;

-- updated_at upkeep: bumped by trigger on every row update so application
-- code never has to remember it. card_prices is exempt — snapshots are
-- immutable history.
CREATE OR REPLACE FUNCTION set_updated_at() RETURNS trigger AS $$
BEGIN
    NEW.updated_at = now();
    RETURN NEW;
END
$$ LANGUAGE plpgsql;

CREATE OR REPLACE TRIGGER games_updated_at
    BEFORE UPDATE ON games
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();
CREATE OR REPLACE TRIGGER sets_updated_at
    BEFORE UPDATE ON sets
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();
CREATE OR REPLACE TRIGGER sealed_updated_at
    BEFORE UPDATE ON sealed
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();
CREATE OR REPLACE TRIGGER cards_updated_at
    BEFORE UPDATE ON cards
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();
