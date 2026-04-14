INSERT INTO users (first_name, last_name, email, phone, password_hash, role, status, email_verified)
VALUES
('Admin', 'User', 'admin@vein.dev', '+10000000001', '$2a$10$2UMQQ2qIhcu0FQrnEz54jOgdZc5BtHw8Br8zkab8JeO8u0bgYGurm', 'admin', 'active', true),
('Manager', 'User', 'manager@vein.dev', '+10000000002', '$2a$10$2UMQQ2qIhcu0FQrnEz54jOgdZc5BtHw8Br8zkab8JeO8u0bgYGurm', 'manager', 'active', true),
('Regular', 'User', 'user@vein.dev', '+10000000003', '$2a$10$2UMQQ2qIhcu0FQrnEz54jOgdZc5BtHw8Br8zkab8JeO8u0bgYGurm', 'user', 'active', false)
ON CONFLICT (email) DO NOTHING;
