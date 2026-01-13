-- Add title column to pods table for terminal title from OSC 0/2 escape sequences
ALTER TABLE pods ADD COLUMN IF NOT EXISTS title VARCHAR(255);
