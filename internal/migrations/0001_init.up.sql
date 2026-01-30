-- ===== USERS =====
CREATE TABLE users (
    id SERIAL PRIMARY KEY,
    username TEXT UNIQUE NOT NULL,
    pass_hash TEXT NOT NULL,
    role TEXT NOT NULL CHECK (
        role IN ('admin', 'manager', 'viewer')
    ),
    created_at TIMESTAMP NOT NULL DEFAULT now()
);

-- ===== ITEMS =====
CREATE TABLE items (
    id SERIAL PRIMARY KEY,
    title TEXT NOT NULL,
    description TEXT,
    price BIGINT NOT NULL,
    visible BOOLEAN NOT NULL DEFAULT true,
    available_amount INT NOT NULL DEFAULT 0,
    created_at TIMESTAMP NOT NULL DEFAULT now(),
    updated_at TIMESTAMP NOT NULL DEFAULT now(),
    deleted_at TIMESTAMP NULL,
    updated_by TEXT
);

-- ===== ITEMS HISTORY =====
CREATE TABLE items_history (
    id SERIAL PRIMARY KEY,
    item_id INT NOT NULL REFERENCES items (id) ON DELETE CASCADE,
    version INT NOT NULL,
    action TEXT NOT NULL CHECK (
        action IN ('INSERT', 'UPDATE', 'DELETE')
    ),
    changed_at TIMESTAMP NOT NULL DEFAULT now(),
    changed_by TEXT,
    old_data JSONB,
    new_data JSONB
);

CREATE INDEX idx_items_history_item_id ON items_history (item_id);

CREATE INDEX idx_items_history_changed_at ON items_history (changed_at);

CREATE INDEX idx_items_history_changed_by ON items_history (changed_by);

-- ===== TRIGGER FUNCTION =====
CREATE OR REPLACE FUNCTION log_item_changes()
RETURNS TRIGGER AS $$
DECLARE
    next_version INT;
BEGIN
    SELECT COALESCE(MAX(version), 0) + 1
    INTO next_version
    FROM items_history
    WHERE item_id = COALESCE(NEW.id, OLD.id);

    IF TG_OP = 'INSERT' THEN
        INSERT INTO items_history(item_id, version, action, old_data, new_data, changed_by)
        VALUES (NEW.id, next_version, 'INSERT', NULL, to_jsonb(NEW), NEW.updated_by);
        RETURN NEW;

    ELSIF TG_OP = 'UPDATE' THEN
        INSERT INTO items_history(item_id, version, action, old_data, new_data, changed_by)
        VALUES (NEW.id, next_version, 'UPDATE', to_jsonb(OLD), to_jsonb(NEW), NEW.updated_by);
        RETURN NEW;

    ELSIF TG_OP = 'DELETE' THEN
        INSERT INTO items_history(item_id, version, action, old_data, new_data, changed_by)
        VALUES (OLD.id, next_version, 'DELETE', to_jsonb(OLD), NULL, OLD.updated_by);
        RETURN OLD;
    END IF;
END;
$$ LANGUAGE plpgsql;

-- ===== TRIGGER =====
CREATE TRIGGER items_audit_trigger
AFTER INSERT OR UPDATE OR DELETE ON items
FOR EACH ROW EXECUTE FUNCTION log_item_changes();