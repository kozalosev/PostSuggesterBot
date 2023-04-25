CREATE OR REPLACE FUNCTION check_revoked_published()
    RETURNS TRIGGER
    LANGUAGE PLPGSQL
AS $$
BEGIN
    IF NEW.revoked AND NEW.published THEN
        RAISE EXCEPTION 'The suggestion cannot be revoked and published simultaneously!';
    END IF;
    RETURN NEW;
END;
$$;

CREATE OR REPLACE TRIGGER trg_check_revoked_published BEFORE UPDATE ON Suggestions
    FOR EACH ROW EXECUTE FUNCTION check_revoked_published();
