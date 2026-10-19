-- Migration: Add indexes for transaction pagination performance

-- Index for filtering by user_id and sorting by created_at
-- This makes paginated queries much faster
-- 
-- How it works:
-- - user_id: Find only this user's transactions (equality filter)
-- - created_at DESC: Already sorted newest first (ORDER BY)
-- - Combined: One index lookup does both filter AND sort
CREATE INDEX IF NOT EXISTS idx_transactions_user_created 
ON transactions(user_id, created_at DESC);