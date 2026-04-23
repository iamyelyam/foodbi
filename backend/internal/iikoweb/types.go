package iikoweb

// AuthLoginRequest is the JSON body for POST /api/auth/login.
type AuthLoginRequest struct {
	Login    string `json:"login"`
	Password string `json:"password"`
}

// AuthStatusResponse is returned by GET /api/auth (status check, not login).
// Useful as a sanity probe and to extract appversion / clientName.
type AuthStatusResponse struct {
	Domain     string      `json:"domain"`
	ClientName string      `json:"clientName"`
	Authorized bool        `json:"authorized"`
	User       interface{} `json:"user"`
	AppVersion string      `json:"appversion"`
}

// AuthLoginResponse is returned by POST /api/auth/login.
// On success: error=false and user is populated.
// On failure: error=true with message (e.g. "Неверные авторизационные данные").
type AuthLoginResponse struct {
	Error               bool        `json:"error"`
	Warning             bool        `json:"warning"`
	Message             string      `json:"message"`
	ErrorMessage        string      `json:"errorMessage"`
	User                interface{} `json:"user,omitempty"`
	FormValidationError bool        `json:"formValidationError"`
}

// Store entry from /api/stores/list.
// NOTE: Field shape is best-effort and pending live-session verification on a real iikoWeb
// tenant. Keep the type tolerant — extra fields go into Extra via a custom UnmarshalJSON if
// needed later.
type Store struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Code string `json:"code,omitempty"`
}

// StoresListResponse wraps /api/stores/list.
type StoresListResponse struct {
	Stores []Store `json:"stores"`
}

// KpiMetricStoresResponse is the shape of /api/kpi-metric/stores. Exact schema TBD —
// stored as a raw map so callers can probe before the schema is firmed up.
type KpiMetricStoresResponse map[string]interface{}
