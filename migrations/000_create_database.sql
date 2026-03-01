-- Migration 000: Create the weather_station database
-- This file must be run against the 'postgres' database, not the target database
-- Usage: psql -U postgres -f migrations/000_create_database.sql

CREATE DATABASE weather_station;