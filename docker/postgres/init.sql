-- Docker PostgreSQL initialization script
-- This file is executed when the PostgreSQL container starts for the first time.

CREATE DATABASE omnitest;

\c omnitest;

-- Apply the initial migration
\i /docker-entrypoint-initdb.d/migrations/001_init.sql
