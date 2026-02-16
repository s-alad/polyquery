-- name: UpsertMarket :exec
INSERT INTO markets (id, question, condition_id, end_date, resolution_src, status)
VALUES (?, ?, ?, ?, ?, 'watching')
ON CONFLICT(id) DO UPDATE SET
    question = excluded.question,
    end_date = excluded.end_date,
    resolution_src = excluded.resolution_src,
    updated_at = datetime('now');

-- name: GetMarketsByStatus :many
SELECT * FROM markets WHERE status = ? ORDER BY created_at DESC;

-- name: GetMarketByID :one
SELECT * FROM markets WHERE id = ?;

-- name: UpdateMarketResolution :exec
UPDATE markets SET
    status = ?,
    resolver_name = ?,
    outcome = ?,
    confidence = ?,
    reasoning = ?,
    sources = ?,
    updated_at = datetime('now')
WHERE id = ?;

-- name: UpdateMarketStatus :exec
UPDATE markets SET status = ?, updated_at = datetime('now') WHERE id = ?;

-- name: SetMarketProposed :exec
UPDATE markets SET status = 'proposed', tx_hash = ?, proposed_at = datetime('now'), updated_at = datetime('now') WHERE id = ?;

-- name: SetMarketSettled :exec
UPDATE markets SET status = 'settled', settled_at = datetime('now'), updated_at = datetime('now') WHERE id = ?;

-- name: CreateApproval :exec
INSERT INTO approvals (market_id, outcome, confidence, reasoning, sources)
VALUES (?, ?, ?, ?, ?);

-- name: GetPendingApprovals :many
SELECT a.*, m.question, m.end_date, m.resolver_name
FROM approvals a
JOIN markets m ON m.id = a.market_id
WHERE a.status = 'pending'
ORDER BY a.created_at DESC;

-- name: GetApprovalByID :one
SELECT * FROM approvals WHERE id = ?;

-- name: ApproveApproval :exec
UPDATE approvals SET status = 'approved', acted_at = datetime('now') WHERE id = ?;

-- name: RejectApproval :exec
UPDATE approvals SET status = 'rejected', acted_at = datetime('now') WHERE id = ?;

-- name: RecordLedger :exec
INSERT INTO ledger (market_id, type, amount_usdc, tx_hash)
VALUES (?, ?, ?, ?);

-- name: GetLedger :many
SELECT l.*, m.question FROM ledger l
JOIN markets m ON m.id = l.market_id
ORDER BY l.created_at DESC;
