-- name: AcquireSnowflakeNode :one
WITH candidate AS (
    SELECT node_id
    FROM generate_series(0, 1023) AS candidates(node_id)
    WHERE NOT EXISTS (
        SELECT 1
        FROM snowflake_node_leases leases
        WHERE leases.node_id = candidates.node_id
          AND leases.lease_expires_at > sqlc.arg(now_at)
    )
    ORDER BY node_id
    LIMIT 1
)
INSERT INTO snowflake_node_leases (node_id, holder_id, lease_expires_at, updated_at)
SELECT node_id, sqlc.arg(holder_id), sqlc.arg(lease_expires_at), sqlc.arg(now_at)
FROM candidate
ON CONFLICT (node_id) DO UPDATE
SET holder_id = EXCLUDED.holder_id,
    lease_expires_at = EXCLUDED.lease_expires_at,
    updated_at = EXCLUDED.updated_at
WHERE snowflake_node_leases.lease_expires_at <= EXCLUDED.updated_at
RETURNING node_id, holder_id, lease_expires_at;

-- name: RenewSnowflakeNode :one
UPDATE snowflake_node_leases
SET lease_expires_at = sqlc.arg(lease_expires_at),
    updated_at = sqlc.arg(now_at)
WHERE node_id = sqlc.arg(node_id)
  AND holder_id = sqlc.arg(holder_id)
  AND lease_expires_at > sqlc.arg(now_at)
RETURNING node_id, holder_id, lease_expires_at;

-- name: ListObjectReferences :many
SELECT id, object_key, sha256, size_bytes, media_type, retention_state, created_at
FROM object_references
WHERE retention_state <> 'purged'
ORDER BY id;

-- name: CreateEvidenceVector :one
INSERT INTO evidence_vectors (id, object_reference_id, embedding, model)
VALUES (sqlc.arg(id), sqlc.narg(object_reference_id), sqlc.arg(embedding), sqlc.arg(model))
RETURNING id, object_reference_id, embedding, model, created_at;

-- name: GetEvidenceVector :one
SELECT id, object_reference_id, embedding, model, created_at
FROM evidence_vectors
WHERE id = sqlc.arg(id);
