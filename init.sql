CREATE TABLE orders (
	id VARCHAR(64) PRIMARY KEY,
	status VARCHAR(32) NOT NULL DEFAULT 'pending',
	price NUMERIC(10, 2) NOT NULL DEFAULT 0.00,
	created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE TABLE order_items (
	id VARCHAR(64) PRIMARY KEY,
	order_id VARCHAR(64) NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
	sku VARCHAR(64) NOT NULL,
	price NUMERIC(10, 2) NOT NULL,
	status VARCHAR(32) NOT NULL DEFAULT 'pending',
	code VARCHAR(64),
	created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE TABLE payment_events (
	event_id VARCHAR(64) PRIMARY KEY,
	order_id VARCHAR(64) NOT NULL REFERENCES orders(id),
	status VARCHAR(32) NOT NULL
);

CREATE TABLE keys (
	id SERIAL PRIMARY KEY,
	sku VARCHAR(64) NOT NULL,
	code VARCHAR(64) NOT NULL UNIQUE,
	status VARCHAR(32) NOT NULL DEFAULT 'available',
	request_id VARCHAR(128)
);

CREATE INDEX IF NOT EXISTS idx_keys_sku_status ON keys (sku) WHERE status = 'available';
CREATE INDEX IF NOT EXISTS idx_order_items_pending ON order_items (order_id) WHERE status = 'pending';

INSERT INTO keys (sku, code, status) VALUES
('STEAM-TOPUP-500', 'LFXC-TNCS-BPCD', 'available'),
('STEAM-TOPUP-500', 'P3EI-W8UO-9B4K', 'available'),
('STEAM-TOPUP-500', 'FEL3-GUXN-TCCH', 'available'),
('STEAM-TOPUP-500', 'YPLV-QK2Z-IUS5', 'available'),
('STEAM-TOPUP-500', '0K9E-P1FR-BY1U', 'available'),
('KEY-CS2-PRIME',   '5LZV-UQ48-RXCZ', 'available'),
('KEY-CS2-PRIME',   'X93K-NYAQ-GEC1', 'available'),
('KEY-CS2-PRIME',   'EIO5-CQT5-35KO', 'available'),
('KEY-GTA5',        'M58F-GIIR-VJAP', 'available'),
('KEY-GTA5',        'NU8Y-SWYB-6252', 'available')
ON CONFLICT DO NOTHING;
