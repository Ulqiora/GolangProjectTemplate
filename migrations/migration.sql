CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE TABLE IF NOT EXISTS public.client_registration_info(
    id uuid DEFAULT uuid_generate_v4(),
    login TEXT NOT NULL,
    email TEXT NOT NULL,
    hashed_password TEXT NOT NULL,
    otp_secret TEXT NOT NULL,
    otp_secret_url TEXT NOT NULL,
    nonce TEXT NOT NULL,
    is_verified BOOLEAN NOT NULL DEFAULT FALSE
);

CREATE TABLE IF NOT EXISTS public.user(
    id uuid DEFAULT uuid_generate_v4(),
    email TEXT NOT NULL,
    hashed_password TEXT NOT NULL
);
