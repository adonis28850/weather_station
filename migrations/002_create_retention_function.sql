-- Migration 002: Create retention cleanup function
-- This function deletes readings older than the specified number of days
-- It's called periodically by a scheduled job in the Go application
CREATE OR REPLACE FUNCTION delete_old_readings(retention_days INTEGER)
RETURNS void AS $$
BEGIN
    -- Delete all readings where the timestamp is older than the specified retention period
    -- $1 is the retention_days parameter (e.g., 30 for 30 days)
    -- NOW() - INTERVAL '1 day' * retention_days calculates the cutoff date
    DELETE FROM readings
    WHERE timestamp < NOW() - INTERVAL '1 day' * retention_days;
END;
$$ LANGUAGE plpgsql;

-- Note: This function is called by the Go application's retention cleaner goroutine