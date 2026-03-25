package server

import "time"

type UserRecord struct {
	ID          int64     `json:"id"`
	OIDCSubject string    `json:"oidc_subject"`
	Email       string    `json:"email"`
	DisplayName string    `json:"display_name"`
	Role        string    `json:"role"`
	Groups      []string  `json:"groups"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	LastLoginAt time.Time `json:"last_login_at"`
}

type SessionRecord struct {
	ID               int64
	UserID           int64
	SessionTokenHash string
	ExpiresAt        time.Time
	CreatedAt        time.Time
	LastSeenAt       time.Time
}

type LoginStateRecord struct {
	ID           int64
	State        string
	CodeVerifier string
	RedirectTo   string
	ExpiresAt    time.Time
	CreatedAt    time.Time
}

type OIDCClaims struct {
	Subject     string
	Email       string
	DisplayName string
	Groups      []string
}

type DeviceAuthorization struct {
	DeviceCode              string `json:"device_code"`
	UserCode                string `json:"user_code"`
	VerificationURI         string `json:"verification_uri"`
	VerificationURIComplete string `json:"verification_uri_complete,omitempty"`
	ExpiresIn               int    `json:"expires_in"`
	Interval                int    `json:"interval"`
}

type AgentRecord struct {
	ID           int64     `json:"id"`
	Name         string    `json:"name"`
	OwnerUserID  int64     `json:"owner_user_id"`
	Status       string    `json:"status"`
	LastSeenAt   time.Time `json:"last_seen_at"`
	CreatedAt    time.Time `json:"created_at"`
	RevokedAt    time.Time `json:"revoked_at"`
	MetadataJSON string    `json:"metadata_json"`
}
