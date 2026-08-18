-- ============================================
-- SpeedMap Database Schema (PostgreSQL)
-- ============================================

-- -------------------------------------------
-- Users (optional registration)
-- -------------------------------------------
CREATE TABLE IF NOT EXISTS users (
    id         SERIAL          PRIMARY KEY,
    usuario    VARCHAR(50)     NOT NULL UNIQUE,
    correo     VARCHAR(100)    NOT NULL UNIQUE,
    password   VARCHAR(255)    NOT NULL,
    created_at TIMESTAMP       DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP       DEFAULT CURRENT_TIMESTAMP
);

-- -------------------------------------------
-- ISP Providers
-- -------------------------------------------
CREATE TABLE IF NOT EXISTS providers (
    id         SERIAL          PRIMARY KEY,
    nombre     VARCHAR(100)    NOT NULL UNIQUE,
    created_at TIMESTAMP       DEFAULT CURRENT_TIMESTAMP
);

-- -------------------------------------------
-- Geographic Zones
-- -------------------------------------------
CREATE TABLE IF NOT EXISTS zones (
    id         SERIAL          PRIMARY KEY,
    nombre     VARCHAR(100)    NOT NULL,
    ciudad     VARCHAR(100)    NOT NULL,
    estado     VARCHAR(100)    NOT NULL,
    created_at TIMESTAMP       DEFAULT CURRENT_TIMESTAMP
);

-- -------------------------------------------
-- Speed Test Results
-- -------------------------------------------
CREATE TABLE IF NOT EXISTS speedtests (
    id            BIGSERIAL        PRIMARY KEY,
    provider_id   INT              NOT NULL REFERENCES providers(id),
    zone_id       INT              NULL REFERENCES zones(id),
    download_mbps DECIMAL(10,2)    NOT NULL,
    upload_mbps   DECIMAL(10,2)    NOT NULL,
    ping_ms       DECIMAL(10,2)    NOT NULL,
    latitude      DECIMAL(10,6)    NOT NULL,
    longitude     DECIMAL(10,6)    NOT NULL,
    ip_hash       VARCHAR(64)      NULL,
    visitor_id    VARCHAR(64)      NULL,
    created_at    TIMESTAMP        DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_provider ON speedtests(provider_id);
CREATE INDEX IF NOT EXISTS idx_zone ON speedtests(zone_id);
CREATE INDEX IF NOT EXISTS idx_created ON speedtests(created_at);
CREATE INDEX IF NOT EXISTS idx_location ON speedtests(latitude, longitude);

-- -------------------------------------------
-- Connectivity Issue Reports
-- -------------------------------------------
-- Note: Create the enum type if it does not exist.
-- In standard PostgreSQL, IF NOT EXISTS is not supported directly for CREATE TYPE.
-- So we can use a DO block or just try to create it, or use check constraint.
-- Let's define the column using a CHECK constraint for simplicity and safety,
-- which avoids errors if the script is run multiple times:
CREATE TABLE IF NOT EXISTS reports (
    id          BIGSERIAL        PRIMARY KEY,
    provider_id INT              NOT NULL REFERENCES providers(id),
    zone_id     INT              NULL REFERENCES zones(id),
    latitude    DECIMAL(10,6)    NOT NULL,
    longitude   DECIMAL(10,6)    NOT NULL,
    issue_type  VARCHAR(50)      NOT NULL CHECK (issue_type IN ('sin_internet', 'lento', 'intermitente')),
    descripcion TEXT             NULL,
    ip_hash     VARCHAR(64)      NULL,
    created_at  TIMESTAMP        DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_report_provider ON reports(provider_id);
CREATE INDEX IF NOT EXISTS idx_report_zone ON reports(zone_id);
CREATE INDEX IF NOT EXISTS idx_report_created ON reports(created_at);
