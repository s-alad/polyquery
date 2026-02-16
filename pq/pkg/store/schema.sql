CREATE TABLE IF NOT EXISTS markets (
    id              TEXT PRIMARY KEY,
    question        TEXT NOT NULL,
    condition_id    TEXT NOT NULL,
    end_date        TEXT,
    resolution_src  TEXT,
    status          TEXT NOT NULL DEFAULT 'watching',
    resolver_name   TEXT,
    outcome         INTEGER,
    confidence      REAL,
    reasoning       TEXT,
    sources         TEXT, -- JSON array
    proposed_at     TEXT,
    settled_at      TEXT,
    tx_hash         TEXT,
    created_at      TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at      TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE TABLE IF NOT EXISTS approvals (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    market_id   TEXT NOT NULL REFERENCES markets(id),
    outcome     INTEGER NOT NULL,
    confidence  REAL NOT NULL,
    reasoning   TEXT,
    sources     TEXT,
    status      TEXT NOT NULL DEFAULT 'pending',
    created_at  TEXT NOT NULL DEFAULT (datetime('now')),
    acted_at    TEXT
);

CREATE TABLE IF NOT EXISTS ledger (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    market_id   TEXT NOT NULL REFERENCES markets(id),
    type        TEXT NOT NULL,
    amount_usdc REAL NOT NULL,
    tx_hash     TEXT,
    created_at  TEXT NOT NULL DEFAULT (datetime('now'))
);
