-- Core entities
CREATE TABLE IF NOT EXISTS apps (
  id          INTEGER PRIMARY KEY,
  name        TEXT NOT NULL UNIQUE,
  description TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS screens (
  id          INTEGER PRIMARY KEY,
  app_id      INTEGER NOT NULL REFERENCES apps(id),
  name        TEXT NOT NULL UNIQUE,
  description TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS regions (
  id          INTEGER PRIMARY KEY,
  app_id      INTEGER NOT NULL REFERENCES apps(id),
  parent_type TEXT NOT NULL CHECK(parent_type IN ('app','screen','region')),
  parent_id   INTEGER NOT NULL,
  name        TEXT NOT NULL UNIQUE,
  description TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS tags (
  id          INTEGER PRIMARY KEY,
  entity_type TEXT NOT NULL CHECK(entity_type IN ('screen','region')),
  entity_id   INTEGER NOT NULL,
  tag         TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS events (
  id        INTEGER PRIMARY KEY,
  region_id INTEGER NOT NULL REFERENCES regions(id),
  name      TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS transitions (
  id         INTEGER PRIMARY KEY,
  owner_type TEXT NOT NULL CHECK(owner_type IN ('app','screen','region')),
  owner_id   INTEGER NOT NULL,
  on_event   TEXT NOT NULL,
  from_state TEXT,
  to_state   TEXT,
  action     TEXT
);

CREATE TABLE IF NOT EXISTS flows (
  id          INTEGER PRIMARY KEY,
  app_id      INTEGER NOT NULL REFERENCES apps(id),
  name        TEXT NOT NULL UNIQUE,
  description TEXT,
  on_event    TEXT,
  sequence    TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS flow_steps (
  id       INTEGER PRIMARY KEY,
  flow_id  INTEGER NOT NULL REFERENCES flows(id),
  position INTEGER NOT NULL,
  raw      TEXT NOT NULL,
  type     TEXT NOT NULL CHECK(type IN ('screen','region','event','back','action','activate')),
  name     TEXT NOT NULL,
  history  INTEGER NOT NULL DEFAULT 0,
  data     TEXT
);

CREATE TABLE IF NOT EXISTS components (
  id          INTEGER PRIMARY KEY,
  entity_type TEXT NOT NULL CHECK(entity_type IN ('app','screen','region')),
  entity_id   INTEGER NOT NULL,
  component   TEXT NOT NULL,
  props       TEXT NOT NULL DEFAULT '{}',
  on_actions  TEXT,
  visible     TEXT,
  UNIQUE(entity_type, entity_id)
);

CREATE TABLE IF NOT EXISTS attachments (
  id      INTEGER PRIMARY KEY,
  entity  TEXT NOT NULL,
  name    TEXT NOT NULL,
  content BLOB NOT NULL,
  UNIQUE(entity, name)
);

-- Cross-cutting views
CREATE VIEW IF NOT EXISTS event_index AS
SELECT
  e.name AS event,
  r.name AS emitted_by,
  r.parent_type,
  r.parent_id,
  t.owner_type AS handled_at,
  t.owner_id AS handled_by_id,
  t.from_state,
  t.to_state,
  t.action
FROM events e
JOIN regions r ON r.id = e.region_id
LEFT JOIN transitions t ON t.on_event = e.name;

CREATE VIEW IF NOT EXISTS state_machines AS
SELECT
  t.owner_type,
  t.owner_id,
  CASE t.owner_type
    WHEN 'screen' THEN (SELECT s.name FROM screens s WHERE s.id = t.owner_id)
    WHEN 'region' THEN (SELECT r.name FROM regions r WHERE r.id = t.owner_id)
    WHEN 'app'    THEN (SELECT a.name FROM apps a WHERE a.id = t.owner_id)
  END AS owner_name,
  t.on_event,
  t.from_state,
  t.to_state,
  t.action
FROM transitions t
ORDER BY t.owner_type, t.owner_id;

CREATE VIEW IF NOT EXISTS tag_index AS
SELECT
  tg.tag,
  tg.entity_type,
  CASE tg.entity_type
    WHEN 'screen' THEN (SELECT s.name FROM screens s WHERE s.id = tg.entity_id)
    WHEN 'region' THEN (SELECT r.name FROM regions r WHERE r.id = tg.entity_id)
  END AS entity_name
FROM tags tg
ORDER BY tg.tag;

CREATE VIEW IF NOT EXISTS region_tree AS
SELECT
  r.id,
  r.name,
  r.description,
  r.parent_type,
  CASE r.parent_type
    WHEN 'app'    THEN (SELECT a.name FROM apps a WHERE a.id = r.parent_id)
    WHEN 'screen' THEN (SELECT s.name FROM screens s WHERE s.id = r.parent_id)
    WHEN 'region' THEN (SELECT r2.name FROM regions r2 WHERE r2.id = r.parent_id)
  END AS parent_name,
  (SELECT COUNT(*) FROM events e WHERE e.region_id = r.id) AS event_count,
  (SELECT COUNT(*) FROM transitions t WHERE t.owner_type = 'region' AND t.owner_id = r.id) > 0 AS has_states
FROM regions r;
