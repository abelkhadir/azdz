CREATE TRIGGER groups_unique_title_insert
BEFORE INSERT ON groups
FOR EACH ROW
WHEN EXISTS (
    SELECT 1
    FROM groups
    WHERE lower(trim(title)) = lower(trim(NEW.title))
)
BEGIN
    SELECT RAISE(ABORT, 'group title already exists');
END;

CREATE TRIGGER groups_unique_title_update
BEFORE UPDATE OF title ON groups
FOR EACH ROW
WHEN EXISTS (
    SELECT 1
    FROM groups
    WHERE id != NEW.id
      AND lower(trim(title)) = lower(trim(NEW.title))
)
BEGIN
    SELECT RAISE(ABORT, 'group title already exists');
END;
