INSERT INTO users (first_name, last_name, email, phone, password_hash, role, status, email_verified)
VALUES
('Admin', 'User', 'admin@vein.dev', '+10000000001', 'dev-hash', 'admin', 'active', true),
('Manager', 'User', 'manager@vein.dev', '+10000000002', 'dev-hash', 'manager', 'active', true),
('Regular', 'User', 'user@vein.dev', '+10000000003', 'dev-hash', 'user', 'active', false)
ON CONFLICT (email) DO NOTHING;
