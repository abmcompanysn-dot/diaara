ALTER TABLE sales ADD COLUMN IF NOT EXISTS checkout_token TEXT;
CREATE INDEX IF NOT EXISTS idx_sales_checkout_token ON sales(checkout_token);
