CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TABLE IF NOT EXISTS users (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),

    first_name VARCHAR(100),
    last_name VARCHAR(100),

    email VARCHAR(255) NOT NULL UNIQUE,
    phone VARCHAR(30) UNIQUE,

    password_hash TEXT NOT NULL,

    role VARCHAR(50) DEFAULT 'user',
    status VARCHAR(20) DEFAULT 'active',

    email_verified BOOLEAN DEFAULT FALSE,

    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX idx_users_email ON users(email);
CREATE INDEX idx_users_phone ON users(phone);
CREATE INDEX idx_users_status ON users(status);