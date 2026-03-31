-- Add callback_token to orders for secure payment callback verification
ALTER TABLE orders ADD COLUMN callback_token VARCHAR(64);
CREATE UNIQUE INDEX idx_orders_callback_token ON orders(callback_token) WHERE callback_token IS NOT NULL;
