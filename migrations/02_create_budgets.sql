-- Create the budgets table
CREATE TABLE budgets (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id TEXT NOT NULL,
    category TEXT NOT NULL,
    limit_amount DECIMAL(10,2) NOT NULL CHECK (limit_amount > 0),
    spent DECIMAL(10,2) NOT NULL DEFAULT 0.00 CHECK (spent >= 0),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(user_id, category)
);

-- Index for GetByUserAndCategory queries
CREATE INDEX idx_budgets_user_category ON budgets(user_id, category);