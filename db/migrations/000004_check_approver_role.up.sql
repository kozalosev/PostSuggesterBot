CREATE OR REPLACE FUNCTION check_approver_rights()
    RETURNS TRIGGER
    LANGUAGE PLPGSQL
AS $$
BEGIN
    IF (SELECT role FROM Users WHERE uid = NEW.approved_by) NOT IN ('author', 'admin') THEN
        RAISE EXCEPTION 'Attempt to approve a post by a non-authorized user!';
    END IF;
    RETURN NEW;
END;
$$;

CREATE OR REPLACE TRIGGER trg_check_approver_rights BEFORE INSERT OR UPDATE ON Approvals
    FOR EACH ROW EXECUTE FUNCTION check_approver_rights();
