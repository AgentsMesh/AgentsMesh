-- Remove the description (summary) column from tickets table
-- This field is redundant with the content field
ALTER TABLE tickets DROP COLUMN IF EXISTS description;
