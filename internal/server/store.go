package server

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/netip"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"time"

	_ "modernc.org/sqlite"
)

const (
	defaultRequestLogLimit = 10000
	defaultRequestLogTTL   = 7 * 24 * time.Hour
)

type TunnelRecord struct {
	ID              int64     `json:"id"`
	Name            string    `json:"name"`
	Hostname        string    `json:"hostname"`
	LocalHint       string    `json:"local_target_hint"`
	OwnerUserID     int64     `json:"owner_user_id"`
	OwnerEmail      string    `json:"owner_email"`
	AssignedAgentID int64     `json:"assigned_agent_id"`
	AssignedAgent   string    `json:"assigned_agent"`
	DesiredState    string    `json:"desired_state"`
	ActualState     string    `json:"actual_state"`
	LastError       string    `json:"last_error"`
	LastSeenAt      time.Time `json:"last_seen_at"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

type TunnelAccessRuleRecord struct {
	ID                     int64     `json:"id"`
	TunnelID               int64     `json:"tunnel_id"`
	AuthMode               string    `json:"auth_mode"`
	BasicAuthUsername      string    `json:"basic_auth_username"`
	BasicAuthPasswordHash  string    `json:"basic_auth_password_hash"`
	SharedSecretHeaderName string    `json:"shared_secret_header_name"`
	SharedSecretHash       string    `json:"shared_secret_hash"`
	AllowedIPs             []string  `json:"allowed_ips"`
	CreatedAt              time.Time `json:"created_at"`
	UpdatedAt              time.Time `json:"updated_at"`
}

type TunnelEventRecord struct {
	ID        int64     `json:"id"`
	TunnelID  int64     `json:"tunnel_id"`
	EventType string    `json:"event_type"`
	Message   string    `json:"message"`
	CreatedAt time.Time `json:"created_at"`
}

type AccessRuleInput struct {
	AuthMode               string   `json:"auth_mode"`
	BasicAuthUsername      string   `json:"basic_auth_username"`
	BasicAuthPassword      string   `json:"basic_auth_password"`
	SharedSecretHeaderName string   `json:"shared_secret_header_name"`
	SharedSecretValue      string   `json:"shared_secret_value"`
	AllowedIPs             []string `json:"allowed_ips"`
}

type RequestLogRecord struct {
	ID        int64     `json:"id"`
	TunnelID  int64     `json:"tunnel_id"`
	Hostname  string    `json:"hostname"`
	Timestamp time.Time `json:"timestamp"`
	Method    string    `json:"method"`
	Host      string    `json:"host"`
	Path      string    `json:"path"`
	Status    int       `json:"status"`
	LatencyMS int64     `json:"latency_ms"`
	BytesIn   int64     `json:"bytes_in"`
	BytesOut  int64     `json:"bytes_out"`
	ClientIP  string    `json:"client_ip"`
	ErrorText string    `json:"error_text"`
	WSUpgrade bool      `json:"ws_upgrade"`
}

type Store struct {
	db            *sql.DB
	retentionHits atomic.Uint64
}

func NewStore(dataDir string) (*Store, error) {
	if dataDir == "" {
		return nil, fmt.Errorf("data-dir required")
	}
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return nil, err
	}
	dbPath := filepath.Join(dataDir, "fwdx.db")
	dsn := dbPath + "?_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)&_pragma=foreign_keys(ON)"
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	s := &Store{db: db}
	if err := s.migrate(context.Background()); err != nil {
		_ = db.Close()
		return nil, err
	}
	return s, nil
}

func (s *Store) Close() error {
	if s == nil || s.db == nil {
		return nil
	}
	return s.db.Close()
}

func (s *Store) migrate(ctx context.Context) error {
	const schema = `
CREATE TABLE IF NOT EXISTS tunnels (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  name TEXT NOT NULL,
  hostname TEXT NOT NULL UNIQUE,
  local_target_hint TEXT NOT NULL DEFAULT '',
  owner_user_id INTEGER NOT NULL DEFAULT 0,
  assigned_agent_id INTEGER NOT NULL DEFAULT 0,
  desired_state TEXT NOT NULL DEFAULT 'running',
  actual_state TEXT NOT NULL DEFAULT 'offline',
  last_error TEXT NOT NULL DEFAULT '',
  last_seen_at TEXT NOT NULL DEFAULT '',
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS agents (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  name TEXT NOT NULL UNIQUE,
  credential_hash TEXT NOT NULL UNIQUE,
  owner_user_id INTEGER NOT NULL,
  status TEXT NOT NULL DEFAULT 'authorized',
  last_seen_at TEXT NOT NULL DEFAULT '',
  created_at TEXT NOT NULL,
  revoked_at TEXT NOT NULL DEFAULT '',
  metadata_json TEXT NOT NULL DEFAULT '{}',
  FOREIGN KEY(owner_user_id) REFERENCES users(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS request_logs (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  tunnel_id INTEGER NOT NULL,
  hostname TEXT NOT NULL,
  timestamp TEXT NOT NULL,
  method TEXT NOT NULL,
  host TEXT NOT NULL,
  path TEXT NOT NULL,
  status INTEGER NOT NULL,
  latency_ms INTEGER NOT NULL,
  bytes_in INTEGER NOT NULL,
  bytes_out INTEGER NOT NULL,
  client_ip TEXT NOT NULL,
  error_text TEXT NOT NULL DEFAULT '',
  ws_upgrade INTEGER NOT NULL DEFAULT 0,
  FOREIGN KEY(tunnel_id) REFERENCES tunnels(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS tunnel_access_rules (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  tunnel_id INTEGER NOT NULL UNIQUE,
  auth_mode TEXT NOT NULL DEFAULT 'public',
  basic_auth_username TEXT NOT NULL DEFAULT '',
  basic_auth_password_hash TEXT NOT NULL DEFAULT '',
  shared_secret_header_name TEXT NOT NULL DEFAULT '',
  shared_secret_hash TEXT NOT NULL DEFAULT '',
  allowed_ips_json TEXT NOT NULL DEFAULT '[]',
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL,
  FOREIGN KEY(tunnel_id) REFERENCES tunnels(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS tunnel_events (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  tunnel_id INTEGER NOT NULL,
  event_type TEXT NOT NULL,
  message TEXT NOT NULL,
  created_at TEXT NOT NULL,
  FOREIGN KEY(tunnel_id) REFERENCES tunnels(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS users (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  oidc_subject TEXT NOT NULL UNIQUE,
  email TEXT NOT NULL DEFAULT '',
  display_name TEXT NOT NULL DEFAULT '',
  roles TEXT NOT NULL DEFAULT 'member',
  groups_snapshot TEXT NOT NULL DEFAULT '[]',
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL,
  last_login_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS oidc_sessions (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  user_id INTEGER NOT NULL,
  session_token_hash TEXT NOT NULL UNIQUE,
  expires_at TEXT NOT NULL,
  created_at TEXT NOT NULL,
  last_seen_at TEXT NOT NULL,
  FOREIGN KEY(user_id) REFERENCES users(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS oidc_login_states (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  state TEXT NOT NULL UNIQUE,
  code_verifier TEXT NOT NULL,
  redirect_to TEXT NOT NULL DEFAULT '/admin/ui',
  expires_at TEXT NOT NULL,
  created_at TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_tunnels_name ON tunnels(name);
CREATE UNIQUE INDEX IF NOT EXISTS idx_tunnels_name_unique ON tunnels(name);
CREATE INDEX IF NOT EXISTS idx_tunnels_hostname ON tunnels(hostname);
CREATE INDEX IF NOT EXISTS idx_tunnels_owner ON tunnels(owner_user_id);
CREATE INDEX IF NOT EXISTS idx_tunnels_assigned_agent ON tunnels(assigned_agent_id);
CREATE INDEX IF NOT EXISTS idx_agents_name ON agents(name);
CREATE INDEX IF NOT EXISTS idx_agents_owner ON agents(owner_user_id);
CREATE INDEX IF NOT EXISTS idx_agents_credential_hash ON agents(credential_hash);
CREATE INDEX IF NOT EXISTS idx_request_logs_tunnel_time ON request_logs(tunnel_id, timestamp DESC);
CREATE INDEX IF NOT EXISTS idx_tunnel_access_rules_tunnel_id ON tunnel_access_rules(tunnel_id);
CREATE INDEX IF NOT EXISTS idx_tunnel_events_tunnel_time ON tunnel_events(tunnel_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_users_subject ON users(oidc_subject);
CREATE INDEX IF NOT EXISTS idx_users_email ON users(email);
CREATE INDEX IF NOT EXISTS idx_oidc_sessions_hash ON oidc_sessions(session_token_hash);
`
	if _, err := s.db.ExecContext(ctx, schema); err != nil {
		return err
	}
	legacy := []string{
		`ALTER TABLE tunnels ADD COLUMN owner_user_id INTEGER NOT NULL DEFAULT 0`,
		`ALTER TABLE tunnels ADD COLUMN assigned_agent_id INTEGER NOT NULL DEFAULT 0`,
	}
	for _, stmt := range legacy {
		_, _ = s.db.ExecContext(ctx, stmt)
	}
	return nil
}

func tunnelNameFromHostname(hostname string) string {
	hostname = strings.TrimSpace(strings.ToLower(hostname))
	if hostname == "" {
		return ""
	}
	if i := strings.IndexByte(hostname, '.'); i > 0 {
		return hostname[:i]
	}
	return hostname
}

func (s *Store) UpsertTunnelStatus(ctx context.Context, hostname, localHint, desiredState, actualState, lastError string, seenAt time.Time) error {
	now := time.Now().UTC().Format(time.RFC3339Nano)
	seen := ""
	if !seenAt.IsZero() {
		seen = seenAt.UTC().Format(time.RFC3339Nano)
	}
	name := tunnelNameFromHostname(hostname)
	if name == "" {
		name = hostname
	}
	const q = `
INSERT INTO tunnels (name, hostname, local_target_hint, desired_state, actual_state, last_error, last_seen_at, created_at, updated_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(hostname) DO UPDATE SET
  local_target_hint=excluded.local_target_hint,
  desired_state=excluded.desired_state,
  actual_state=excluded.actual_state,
  last_error=excluded.last_error,
  last_seen_at=excluded.last_seen_at,
  updated_at=excluded.updated_at
`
	_, err := s.db.ExecContext(ctx, q, name, hostname, localHint, desiredState, actualState, lastError, seen, now, now)
	return err
}

func (s *Store) MarkTunnelOffline(ctx context.Context, hostname, lastError string) error {
	const q = `
UPDATE tunnels
SET actual_state = 'offline',
    last_error = ?,
    updated_at = ?
WHERE hostname = ?
`
	_, err := s.db.ExecContext(ctx, q, lastError, time.Now().UTC().Format(time.RFC3339Nano), hostname)
	return err
}

func (s *Store) ListTunnels(ctx context.Context) ([]TunnelRecord, error) {
	rows, err := s.db.QueryContext(ctx, `
SELECT t.id, t.name, t.hostname, t.local_target_hint, t.owner_user_id, COALESCE(u.email, ''), t.assigned_agent_id, COALESCE(a.name, ''), t.desired_state, t.actual_state, t.last_error, t.last_seen_at, t.created_at, t.updated_at
FROM tunnels t
LEFT JOIN users u ON u.id = t.owner_user_id
LEFT JOIN agents a ON a.id = t.assigned_agent_id
ORDER BY hostname ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []TunnelRecord
	for rows.Next() {
		var rec TunnelRecord
		var lastSeen, created, updated string
		if err := rows.Scan(&rec.ID, &rec.Name, &rec.Hostname, &rec.LocalHint, &rec.OwnerUserID, &rec.OwnerEmail, &rec.AssignedAgentID, &rec.AssignedAgent, &rec.DesiredState, &rec.ActualState, &rec.LastError, &lastSeen, &created, &updated); err != nil {
			return nil, err
		}
		rec.LastSeenAt = parseRFC3339(lastSeen)
		rec.CreatedAt = parseRFC3339(created)
		rec.UpdatedAt = parseRFC3339(updated)
		out = append(out, rec)
	}
	return out, rows.Err()
}

func (s *Store) ListTunnelsForUser(ctx context.Context, userID int64, isAdmin bool) ([]TunnelRecord, error) {
	if isAdmin {
		return s.ListTunnels(ctx)
	}
	rows, err := s.db.QueryContext(ctx, `
SELECT t.id, t.name, t.hostname, t.local_target_hint, t.owner_user_id, COALESCE(u.email, ''), t.assigned_agent_id, COALESCE(a.name, ''), t.desired_state, t.actual_state, t.last_error, t.last_seen_at, t.created_at, t.updated_at
FROM tunnels t
LEFT JOIN users u ON u.id = t.owner_user_id
LEFT JOIN agents a ON a.id = t.assigned_agent_id
WHERE t.owner_user_id = ?
ORDER BY t.hostname ASC`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []TunnelRecord
	for rows.Next() {
		var rec TunnelRecord
		var lastSeen, created, updated string
		if err := rows.Scan(&rec.ID, &rec.Name, &rec.Hostname, &rec.LocalHint, &rec.OwnerUserID, &rec.OwnerEmail, &rec.AssignedAgentID, &rec.AssignedAgent, &rec.DesiredState, &rec.ActualState, &rec.LastError, &lastSeen, &created, &updated); err != nil {
			return nil, err
		}
		rec.LastSeenAt = parseRFC3339(lastSeen)
		rec.CreatedAt = parseRFC3339(created)
		rec.UpdatedAt = parseRFC3339(updated)
		out = append(out, rec)
	}
	return out, rows.Err()
}

func (s *Store) CreateTunnel(ctx context.Context, ownerUserID int64, name, hostname, localHint string, assignedAgentID int64) (TunnelRecord, error) {
	now := time.Now().UTC().Format(time.RFC3339Nano)
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return TunnelRecord{}, err
	}
	defer tx.Rollback()
	_, err = tx.ExecContext(ctx, `
INSERT INTO tunnels (name, hostname, local_target_hint, owner_user_id, assigned_agent_id, desired_state, actual_state, last_error, last_seen_at, created_at, updated_at)
VALUES (?, ?, ?, ?, ?, 'stopped', 'offline', '', '', ?, ?)`,
		name, hostname, localHint, ownerUserID, assignedAgentID, now, now)
	if err != nil {
		return TunnelRecord{}, err
	}
	if err := upsertTunnelAccessRuleTx(ctx, tx, 0, name, hostname, AccessRuleInput{AuthMode: "public"}, true, nil); err != nil {
		return TunnelRecord{}, err
	}
	if err := tx.Commit(); err != nil {
		return TunnelRecord{}, err
	}
	return s.GetTunnelByName(ctx, name)
}

func (s *Store) GetTunnelByName(ctx context.Context, name string) (TunnelRecord, error) {
	row := s.db.QueryRowContext(ctx, `
SELECT t.id, t.name, t.hostname, t.local_target_hint, t.owner_user_id, COALESCE(u.email, ''), t.assigned_agent_id, COALESCE(a.name, ''), t.desired_state, t.actual_state, t.last_error, t.last_seen_at, t.created_at, t.updated_at
FROM tunnels t
LEFT JOIN users u ON u.id = t.owner_user_id
LEFT JOIN agents a ON a.id = t.assigned_agent_id
WHERE t.name = ?`, name)
	var rec TunnelRecord
	var lastSeen, created, updated string
	if err := row.Scan(&rec.ID, &rec.Name, &rec.Hostname, &rec.LocalHint, &rec.OwnerUserID, &rec.OwnerEmail, &rec.AssignedAgentID, &rec.AssignedAgent, &rec.DesiredState, &rec.ActualState, &rec.LastError, &lastSeen, &created, &updated); err != nil {
		return TunnelRecord{}, err
	}
	rec.LastSeenAt = parseRFC3339(lastSeen)
	rec.CreatedAt = parseRFC3339(created)
	rec.UpdatedAt = parseRFC3339(updated)
	return rec, nil
}

func (s *Store) DeleteTunnelByName(ctx context.Context, name string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM tunnels WHERE name = ?`, name)
	return err
}

func (s *Store) AssignTunnelToAgent(ctx context.Context, name string, agentID int64) error {
	_, err := s.db.ExecContext(ctx, `UPDATE tunnels SET assigned_agent_id = ?, updated_at = ? WHERE name = ?`, agentID, time.Now().UTC().Format(time.RFC3339Nano), name)
	return err
}

func (s *Store) SetTunnelDesiredState(ctx context.Context, name, desiredState string) error {
	_, err := s.db.ExecContext(ctx, `UPDATE tunnels SET desired_state = ?, updated_at = ? WHERE name = ?`, desiredState, time.Now().UTC().Format(time.RFC3339Nano), name)
	return err
}

func (s *Store) UpdateTunnelStateByName(ctx context.Context, name, localHint, actualState, lastError string, seenAt time.Time) error {
	seen := ""
	if !seenAt.IsZero() {
		seen = seenAt.UTC().Format(time.RFC3339Nano)
	}
	_, err := s.db.ExecContext(ctx, `
UPDATE tunnels
SET local_target_hint = CASE WHEN ? <> '' THEN ? ELSE local_target_hint END,
    actual_state = ?,
    last_error = ?,
    last_seen_at = ?,
    updated_at = ?
WHERE name = ?`,
		localHint, localHint, actualState, lastError, seen, time.Now().UTC().Format(time.RFC3339Nano), name)
	return err
}

func (s *Store) GetTunnelForAgent(ctx context.Context, name string, agentID int64) (TunnelRecord, error) {
	row := s.db.QueryRowContext(ctx, `
SELECT t.id, t.name, t.hostname, t.local_target_hint, t.owner_user_id, COALESCE(u.email, ''), t.assigned_agent_id, COALESCE(a.name, ''), t.desired_state, t.actual_state, t.last_error, t.last_seen_at, t.created_at, t.updated_at
FROM tunnels t
LEFT JOIN users u ON u.id = t.owner_user_id
LEFT JOIN agents a ON a.id = t.assigned_agent_id
WHERE t.name = ? AND t.assigned_agent_id = ?`, name, agentID)
	var rec TunnelRecord
	var lastSeen, created, updated string
	if err := row.Scan(&rec.ID, &rec.Name, &rec.Hostname, &rec.LocalHint, &rec.OwnerUserID, &rec.OwnerEmail, &rec.AssignedAgentID, &rec.AssignedAgent, &rec.DesiredState, &rec.ActualState, &rec.LastError, &lastSeen, &created, &updated); err != nil {
		return TunnelRecord{}, err
	}
	rec.LastSeenAt = parseRFC3339(lastSeen)
	rec.CreatedAt = parseRFC3339(created)
	rec.UpdatedAt = parseRFC3339(updated)
	return rec, nil
}

func (s *Store) InsertRequestLog(ctx context.Context, rec RequestLogRecord) error {
	tunnelID := rec.TunnelID
	var err error
	if tunnelID == 0 {
		tunnelID, err = s.tunnelIDByHostname(ctx, rec.Hostname)
		if err != nil {
			return err
		}
	}
	if tunnelID == 0 {
		return nil
	}
	const q = `
INSERT INTO request_logs (tunnel_id, hostname, timestamp, method, host, path, status, latency_ms, bytes_in, bytes_out, client_ip, error_text, ws_upgrade)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
`
	_, err = s.db.ExecContext(ctx, q,
		tunnelID,
		rec.Hostname,
		rec.Timestamp.UTC().Format(time.RFC3339Nano),
		rec.Method,
		rec.Host,
		rec.Path,
		rec.Status,
		rec.LatencyMS,
		rec.BytesIn,
		rec.BytesOut,
		rec.ClientIP,
		rec.ErrorText,
		boolToInt(rec.WSUpgrade),
	)
	if err != nil {
		return err
	}
	if s.retentionHits.Add(1)%100 == 0 {
		_ = s.enforceLogRetention(ctx, tunnelID)
	}
	return nil
}

func (s *Store) ListRequestLogs(ctx context.Context, hostname string, limit int) ([]RequestLogRecord, error) {
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	rows, err := s.db.QueryContext(ctx, `
SELECT id, tunnel_id, hostname, timestamp, method, host, path, status, latency_ms, bytes_in, bytes_out, client_ip, error_text, ws_upgrade
FROM request_logs
WHERE hostname = ?
ORDER BY timestamp DESC
LIMIT ?`, hostname, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []RequestLogRecord
	for rows.Next() {
		var rec RequestLogRecord
		var ts string
		var ws int
		if err := rows.Scan(&rec.ID, &rec.TunnelID, &rec.Hostname, &ts, &rec.Method, &rec.Host, &rec.Path, &rec.Status, &rec.LatencyMS, &rec.BytesIn, &rec.BytesOut, &rec.ClientIP, &rec.ErrorText, &ws); err != nil {
			return nil, err
		}
		rec.Timestamp = parseRFC3339(ts)
		rec.WSUpgrade = ws == 1
		out = append(out, rec)
	}
	return out, rows.Err()
}

func (s *Store) ListRequestLogsByTunnel(ctx context.Context, tunnelID int64, limit int) ([]RequestLogRecord, error) {
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	rows, err := s.db.QueryContext(ctx, `
SELECT id, tunnel_id, hostname, timestamp, method, host, path, status, latency_ms, bytes_in, bytes_out, client_ip, error_text, ws_upgrade
FROM request_logs
WHERE tunnel_id = ?
ORDER BY timestamp DESC
LIMIT ?`, tunnelID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []RequestLogRecord
	for rows.Next() {
		var rec RequestLogRecord
		var ts string
		var ws int
		if err := rows.Scan(&rec.ID, &rec.TunnelID, &rec.Hostname, &ts, &rec.Method, &rec.Host, &rec.Path, &rec.Status, &rec.LatencyMS, &rec.BytesIn, &rec.BytesOut, &rec.ClientIP, &rec.ErrorText, &ws); err != nil {
			return nil, err
		}
		rec.Timestamp = parseRFC3339(ts)
		rec.WSUpgrade = ws == 1
		out = append(out, rec)
	}
	return out, rows.Err()
}

func (s *Store) UpsertUserFromOIDC(ctx context.Context, subject, email, displayName string, groups []string, role string) (UserRecord, error) {
	now := time.Now().UTC().Format(time.RFC3339Nano)
	groupsJSON := "[]"
	if len(groups) > 0 {
		data, _ := jsonMarshal(groups)
		groupsJSON = string(data)
	}
	if role == "" {
		role = "member"
	}
	const q = `
INSERT INTO users (oidc_subject, email, display_name, roles, groups_snapshot, created_at, updated_at, last_login_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(oidc_subject) DO UPDATE SET
  email=excluded.email,
  display_name=excluded.display_name,
  roles=excluded.roles,
  groups_snapshot=excluded.groups_snapshot,
  updated_at=excluded.updated_at,
  last_login_at=excluded.last_login_at
`
	if _, err := s.db.ExecContext(ctx, q, subject, email, displayName, role, groupsJSON, now, now, now); err != nil {
		return UserRecord{}, err
	}
	row := s.db.QueryRowContext(ctx, `
SELECT id, oidc_subject, email, display_name, roles, groups_snapshot, created_at, updated_at, last_login_at
FROM users WHERE oidc_subject = ?`, subject)
	return scanUserRecord(row)
}

func (s *Store) CreateSession(ctx context.Context, userID int64, tokenHash string, expiresAt time.Time) error {
	now := time.Now().UTC().Format(time.RFC3339Nano)
	_, err := s.db.ExecContext(ctx, `
INSERT INTO oidc_sessions (user_id, session_token_hash, expires_at, created_at, last_seen_at)
VALUES (?, ?, ?, ?, ?)`,
		userID, tokenHash, expiresAt.UTC().Format(time.RFC3339Nano), now, now)
	return err
}

func (s *Store) GetSessionByToken(ctx context.Context, tokenHash string) (SessionRecord, UserRecord, error) {
	row := s.db.QueryRowContext(ctx, `
SELECT s.id, s.user_id, s.session_token_hash, s.expires_at, s.created_at, s.last_seen_at,
       u.id, u.oidc_subject, u.email, u.display_name, u.roles, u.groups_snapshot, u.created_at, u.updated_at, u.last_login_at
FROM oidc_sessions s
JOIN users u ON u.id = s.user_id
WHERE s.session_token_hash = ?`, tokenHash)
	var sess SessionRecord
	var user UserRecord
	var sessExp, sessCreated, sessLastSeen string
	var groupsJSON, userCreated, userUpdated, userLastLogin string
	if err := row.Scan(
		&sess.ID, &sess.UserID, &sess.SessionTokenHash, &sessExp, &sessCreated, &sessLastSeen,
		&user.ID, &user.OIDCSubject, &user.Email, &user.DisplayName, &user.Role, &groupsJSON, &userCreated, &userUpdated, &userLastLogin,
	); err != nil {
		return SessionRecord{}, UserRecord{}, err
	}
	sess.ExpiresAt = parseRFC3339(sessExp)
	sess.CreatedAt = parseRFC3339(sessCreated)
	sess.LastSeenAt = parseRFC3339(sessLastSeen)
	if !sess.ExpiresAt.IsZero() && time.Now().After(sess.ExpiresAt) {
		_ = s.DeleteSession(ctx, tokenHash)
		return SessionRecord{}, UserRecord{}, sql.ErrNoRows
	}
	user.CreatedAt = parseRFC3339(userCreated)
	user.UpdatedAt = parseRFC3339(userUpdated)
	user.LastLoginAt = parseRFC3339(userLastLogin)
	user.Groups = parseJSONStrings(groupsJSON)
	_, _ = s.db.ExecContext(ctx, `UPDATE oidc_sessions SET last_seen_at = ? WHERE id = ?`, time.Now().UTC().Format(time.RFC3339Nano), sess.ID)
	return sess, user, nil
}

func (s *Store) DeleteSession(ctx context.Context, tokenHash string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM oidc_sessions WHERE session_token_hash = ?`, tokenHash)
	return err
}

func (s *Store) CreateLoginState(ctx context.Context, state, verifier, redirectTo string, expiresAt time.Time) error {
	if redirectTo == "" {
		redirectTo = "/admin/ui"
	}
	_, err := s.db.ExecContext(ctx, `
INSERT INTO oidc_login_states (state, code_verifier, redirect_to, expires_at, created_at)
VALUES (?, ?, ?, ?, ?)`,
		state, verifier, redirectTo, expiresAt.UTC().Format(time.RFC3339Nano), time.Now().UTC().Format(time.RFC3339Nano))
	return err
}

func (s *Store) ConsumeLoginState(ctx context.Context, state string) (LoginStateRecord, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return LoginStateRecord{}, err
	}
	defer tx.Rollback()
	row := tx.QueryRowContext(ctx, `
SELECT id, state, code_verifier, redirect_to, expires_at, created_at
FROM oidc_login_states
WHERE state = ?`, state)
	var rec LoginStateRecord
	var expiresAt, createdAt string
	if err := row.Scan(&rec.ID, &rec.State, &rec.CodeVerifier, &rec.RedirectTo, &expiresAt, &createdAt); err != nil {
		return LoginStateRecord{}, err
	}
	rec.ExpiresAt = parseRFC3339(expiresAt)
	rec.CreatedAt = parseRFC3339(createdAt)
	if _, err := tx.ExecContext(ctx, `DELETE FROM oidc_login_states WHERE id = ?`, rec.ID); err != nil {
		return LoginStateRecord{}, err
	}
	if err := tx.Commit(); err != nil {
		return LoginStateRecord{}, err
	}
	if !rec.ExpiresAt.IsZero() && time.Now().After(rec.ExpiresAt) {
		return LoginStateRecord{}, sql.ErrNoRows
	}
	return rec, nil
}

func (s *Store) AddTunnelEvent(ctx context.Context, hostname, eventType, message string) error {
	tunnelID, err := s.tunnelIDByHostname(ctx, hostname)
	if err != nil || tunnelID == 0 {
		return err
	}
	_, err = s.db.ExecContext(ctx, `
INSERT INTO tunnel_events (tunnel_id, event_type, message, created_at)
VALUES (?, ?, ?, ?)`, tunnelID, eventType, message, time.Now().UTC().Format(time.RFC3339Nano))
	return err
}

func (s *Store) ListTunnelEventsByTunnel(ctx context.Context, tunnelID int64, limit int) ([]TunnelEventRecord, error) {
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	rows, err := s.db.QueryContext(ctx, `
SELECT id, tunnel_id, event_type, message, created_at
FROM tunnel_events
WHERE tunnel_id = ?
ORDER BY created_at DESC
LIMIT ?`, tunnelID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []TunnelEventRecord
	for rows.Next() {
		var rec TunnelEventRecord
		var created string
		if err := rows.Scan(&rec.ID, &rec.TunnelID, &rec.EventType, &rec.Message, &created); err != nil {
			return nil, err
		}
		rec.CreatedAt = parseRFC3339(created)
		out = append(out, rec)
	}
	return out, rows.Err()
}

func (s *Store) GetTunnelByHostname(ctx context.Context, hostname string) (TunnelRecord, error) {
	row := s.db.QueryRowContext(ctx, `
SELECT t.id, t.name, t.hostname, t.local_target_hint, t.owner_user_id, COALESCE(u.email, ''), t.assigned_agent_id, COALESCE(a.name, ''), t.desired_state, t.actual_state, t.last_error, t.last_seen_at, t.created_at, t.updated_at
FROM tunnels t
LEFT JOIN users u ON u.id = t.owner_user_id
LEFT JOIN agents a ON a.id = t.assigned_agent_id
WHERE t.hostname = ?`, hostname)
	var rec TunnelRecord
	var lastSeen, created, updated string
	if err := row.Scan(&rec.ID, &rec.Name, &rec.Hostname, &rec.LocalHint, &rec.OwnerUserID, &rec.OwnerEmail, &rec.AssignedAgentID, &rec.AssignedAgent, &rec.DesiredState, &rec.ActualState, &rec.LastError, &lastSeen, &created, &updated); err != nil {
		return TunnelRecord{}, err
	}
	rec.LastSeenAt = parseRFC3339(lastSeen)
	rec.CreatedAt = parseRFC3339(created)
	rec.UpdatedAt = parseRFC3339(updated)
	return rec, nil
}

func (s *Store) GetTunnelAccessRule(ctx context.Context, tunnelID int64) (TunnelAccessRuleRecord, error) {
	row := s.db.QueryRowContext(ctx, `
SELECT id, tunnel_id, auth_mode, basic_auth_username, basic_auth_password_hash, shared_secret_header_name, shared_secret_hash, allowed_ips_json, created_at, updated_at
FROM tunnel_access_rules
WHERE tunnel_id = ?`, tunnelID)
	var rec TunnelAccessRuleRecord
	var allowedIPsJSON, created, updated string
	if err := row.Scan(&rec.ID, &rec.TunnelID, &rec.AuthMode, &rec.BasicAuthUsername, &rec.BasicAuthPasswordHash, &rec.SharedSecretHeaderName, &rec.SharedSecretHash, &allowedIPsJSON, &created, &updated); err != nil {
		return TunnelAccessRuleRecord{}, err
	}
	rec.AllowedIPs = parseJSONStrings(allowedIPsJSON)
	rec.CreatedAt = parseRFC3339(created)
	rec.UpdatedAt = parseRFC3339(updated)
	return rec, nil
}

func (s *Store) UpsertTunnelAccessRule(ctx context.Context, tunnelID int64, input AccessRuleInput) error {
	existing, err := s.GetTunnelAccessRule(ctx, tunnelID)
	var existingRule *TunnelAccessRuleRecord
	if err == nil {
		existingRule = &existing
	} else if err != sql.ErrNoRows {
		return err
	}
	return upsertTunnelAccessRuleTx(ctx, s.db, tunnelID, "", "", input, false, existingRule)
}

func (s *Store) AccessRuleByTunnelName(ctx context.Context, name string) (TunnelAccessRuleRecord, error) {
	tun, err := s.GetTunnelByName(ctx, name)
	if err != nil {
		return TunnelAccessRuleRecord{}, err
	}
	return s.GetTunnelAccessRule(ctx, tun.ID)
}

func (s *Store) CreateAgent(ctx context.Context, ownerUserID int64, name, credentialHash string) (AgentRecord, error) {
	now := time.Now().UTC().Format(time.RFC3339Nano)
	_, err := s.db.ExecContext(ctx, `
INSERT INTO agents (name, credential_hash, owner_user_id, status, last_seen_at, created_at, revoked_at, metadata_json)
VALUES (?, ?, ?, 'authorized', '', ?, '', '{}')`, name, credentialHash, ownerUserID, now)
	if err != nil {
		return AgentRecord{}, err
	}
	return s.GetAgentByName(ctx, name)
}

func (s *Store) GetAgentByName(ctx context.Context, name string) (AgentRecord, error) {
	row := s.db.QueryRowContext(ctx, `
SELECT id, name, owner_user_id, status, last_seen_at, created_at, revoked_at, metadata_json
FROM agents WHERE name = ?`, name)
	return scanAgentRecord(row)
}

func (s *Store) GetAgentByCredentialHash(ctx context.Context, credentialHash string) (AgentRecord, error) {
	row := s.db.QueryRowContext(ctx, `
SELECT id, name, owner_user_id, status, last_seen_at, created_at, revoked_at, metadata_json
FROM agents WHERE credential_hash = ?`, credentialHash)
	return scanAgentRecord(row)
}

func (s *Store) ListAgentsForUser(ctx context.Context, userID int64, isAdmin bool) ([]AgentRecord, error) {
	query := `
SELECT id, name, owner_user_id, status, last_seen_at, created_at, revoked_at, metadata_json
FROM agents`
	var rows *sql.Rows
	var err error
	if isAdmin {
		rows, err = s.db.QueryContext(ctx, query+` ORDER BY name ASC`)
	} else {
		rows, err = s.db.QueryContext(ctx, query+` WHERE owner_user_id = ? ORDER BY name ASC`, userID)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []AgentRecord
	for rows.Next() {
		rec, err := scanAgentRecord(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, rec)
	}
	return out, rows.Err()
}

func (s *Store) RevokeAgentByName(ctx context.Context, name string) error {
	_, err := s.db.ExecContext(ctx, `UPDATE agents SET status = 'revoked', revoked_at = ?, last_seen_at = ? WHERE name = ?`, time.Now().UTC().Format(time.RFC3339Nano), "", name)
	return err
}

func (s *Store) TouchAgent(ctx context.Context, agentID int64, status string) error {
	_, err := s.db.ExecContext(ctx, `UPDATE agents SET status = ?, last_seen_at = ? WHERE id = ?`, status, time.Now().UTC().Format(time.RFC3339Nano), agentID)
	return err
}

func (s *Store) tunnelIDByHostname(ctx context.Context, hostname string) (int64, error) {
	var id int64
	err := s.db.QueryRowContext(ctx, `SELECT id FROM tunnels WHERE hostname = ?`, hostname).Scan(&id)
	if err == sql.ErrNoRows {
		return 0, nil
	}
	return id, err
}

func (s *Store) enforceLogRetention(ctx context.Context, tunnelID int64) error {
	cutoff := time.Now().Add(-defaultRequestLogTTL).UTC().Format(time.RFC3339Nano)
	_, _ = s.db.ExecContext(ctx, `DELETE FROM request_logs WHERE tunnel_id = ? AND timestamp < ?`, tunnelID, cutoff)

	_, err := s.db.ExecContext(ctx, `
DELETE FROM request_logs
WHERE tunnel_id = ?
  AND id NOT IN (
    SELECT id FROM request_logs WHERE tunnel_id = ? ORDER BY timestamp DESC LIMIT ?
  )
`, tunnelID, tunnelID, defaultRequestLogLimit)
	return err
}

func parseRFC3339(s string) time.Time {
	if s == "" {
		return time.Time{}
	}
	t, _ := time.Parse(time.RFC3339Nano, s)
	return t
}

func parseJSONStrings(s string) []string {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	var out []string
	_ = json.Unmarshal([]byte(s), &out)
	return out
}

func jsonMarshal(v any) ([]byte, error) {
	return json.Marshal(v)
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanUserRecord(row rowScanner) (UserRecord, error) {
	var rec UserRecord
	var groupsJSON, created, updated, lastLogin string
	if err := row.Scan(&rec.ID, &rec.OIDCSubject, &rec.Email, &rec.DisplayName, &rec.Role, &groupsJSON, &created, &updated, &lastLogin); err != nil {
		return UserRecord{}, err
	}
	rec.Groups = parseJSONStrings(groupsJSON)
	rec.CreatedAt = parseRFC3339(created)
	rec.UpdatedAt = parseRFC3339(updated)
	rec.LastLoginAt = parseRFC3339(lastLogin)
	return rec, nil
}

func scanAgentRecord(row rowScanner) (AgentRecord, error) {
	var rec AgentRecord
	var lastSeen, created, revoked string
	if err := row.Scan(&rec.ID, &rec.Name, &rec.OwnerUserID, &rec.Status, &lastSeen, &created, &revoked, &rec.MetadataJSON); err != nil {
		return AgentRecord{}, err
	}
	rec.LastSeenAt = parseRFC3339(lastSeen)
	rec.CreatedAt = parseRFC3339(created)
	rec.RevokedAt = parseRFC3339(revoked)
	return rec, nil
}

func boolToInt(v bool) int {
	if v {
		return 1
	}
	return 0
}

type execContexter interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
	QueryRowContext(context.Context, string, ...any) *sql.Row
}

func upsertTunnelAccessRuleTx(ctx context.Context, db execContexter, tunnelID int64, tunnelName, hostname string, input AccessRuleInput, ensureTunnelLookup bool, existing *TunnelAccessRuleRecord) error {
	if ensureTunnelLookup && tunnelID == 0 {
		row := db.QueryRowContext(ctx, `SELECT id FROM tunnels WHERE name = ? OR hostname = ? LIMIT 1`, tunnelName, hostname)
		if err := row.Scan(&tunnelID); err != nil {
			return err
		}
	}
	mode, username, passwordHash, headerName, secretHash, allowed, err := validateAccessRuleInput(input, existing)
	if err != nil {
		return err
	}
	allowedJSON := "[]"
	if len(allowed) > 0 {
		data, _ := json.Marshal(allowed)
		allowedJSON = string(data)
	}
	now := time.Now().UTC().Format(time.RFC3339Nano)
	_, err = db.ExecContext(ctx, `
INSERT INTO tunnel_access_rules (tunnel_id, auth_mode, basic_auth_username, basic_auth_password_hash, shared_secret_header_name, shared_secret_hash, allowed_ips_json, created_at, updated_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(tunnel_id) DO UPDATE SET
  auth_mode=excluded.auth_mode,
  basic_auth_username=excluded.basic_auth_username,
  basic_auth_password_hash=excluded.basic_auth_password_hash,
  shared_secret_header_name=excluded.shared_secret_header_name,
  shared_secret_hash=excluded.shared_secret_hash,
  allowed_ips_json=excluded.allowed_ips_json,
  updated_at=excluded.updated_at
`, tunnelID, mode, username, passwordHash, headerName, secretHash, allowedJSON, now, now)
	return err
}

func validateAccessRuleInput(input AccessRuleInput, existing *TunnelAccessRuleRecord) (mode, username, passwordHash, headerName, secretHash string, allowed []string, err error) {
	mode = strings.TrimSpace(strings.ToLower(input.AuthMode))
	if mode == "" {
		if existing != nil && existing.AuthMode != "" {
			mode = existing.AuthMode
		} else {
			mode = "public"
		}
	}
	switch mode {
	case "public", "basic_auth", "shared_secret_header":
	default:
		return "", "", "", "", "", nil, fmt.Errorf("invalid auth mode")
	}
	allowed = make([]string, 0, len(input.AllowedIPs))
	for _, cidr := range input.AllowedIPs {
		cidr = strings.TrimSpace(cidr)
		if cidr == "" {
			continue
		}
		if _, err := netip.ParsePrefix(cidr); err != nil {
			return "", "", "", "", "", nil, fmt.Errorf("invalid cidr: %s", cidr)
		}
		allowed = append(allowed, cidr)
	}
	switch mode {
	case "public":
		return mode, "", "", "", "", allowed, nil
	case "basic_auth":
		username = strings.TrimSpace(input.BasicAuthUsername)
		if username == "" && existing != nil {
			username = existing.BasicAuthUsername
		}
		if username == "" {
			return "", "", "", "", "", nil, fmt.Errorf("basic auth username required")
		}
		if strings.TrimSpace(input.BasicAuthPassword) != "" {
			passwordHash = hashSecret(input.BasicAuthPassword)
		} else if existing != nil && existing.BasicAuthPasswordHash != "" {
			passwordHash = existing.BasicAuthPasswordHash
		}
		if passwordHash == "" {
			return "", "", "", "", "", nil, fmt.Errorf("basic auth password required")
		}
		return mode, username, passwordHash, "", "", allowed, nil
	case "shared_secret_header":
		headerName = httpCanonicalHeaderKey(strings.TrimSpace(input.SharedSecretHeaderName))
		if headerName == "" && existing != nil {
			headerName = existing.SharedSecretHeaderName
		}
		if headerName == "" {
			return "", "", "", "", "", nil, fmt.Errorf("shared secret header name required")
		}
		if strings.TrimSpace(input.SharedSecretValue) != "" {
			secretHash = hashSecret(input.SharedSecretValue)
		} else if existing != nil && existing.SharedSecretHash != "" {
			secretHash = existing.SharedSecretHash
		}
		if secretHash == "" {
			return "", "", "", "", "", nil, fmt.Errorf("shared secret value required")
		}
		return mode, "", "", headerName, secretHash, allowed, nil
	}
	return "", "", "", "", "", nil, fmt.Errorf("invalid auth mode")
}

func hashSecret(raw string) string {
	sum := sha256.Sum256([]byte(strings.TrimSpace(raw)))
	return hex.EncodeToString(sum[:])
}

func verifyAccessRuleBasicAuth(rule TunnelAccessRuleRecord, username, password string) bool {
	if rule.AuthMode != "basic_auth" {
		return true
	}
	return strings.TrimSpace(rule.BasicAuthUsername) == strings.TrimSpace(username) && rule.BasicAuthPasswordHash == hashSecret(password)
}

func verifyAccessRuleSharedSecret(rule TunnelAccessRuleRecord, value string) bool {
	if rule.AuthMode != "shared_secret_header" {
		return true
	}
	return rule.SharedSecretHash == hashSecret(value)
}

func httpCanonicalHeaderKey(v string) string {
	parts := strings.Split(strings.ToLower(strings.TrimSpace(v)), "-")
	for i, p := range parts {
		if p == "" {
			continue
		}
		parts[i] = strings.ToUpper(p[:1]) + p[1:]
	}
	return strings.Join(parts, "-")
}
