// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package statuspalnext

// The structs below mirror the schemas in spage's docs/openapi.yaml. Optional
// request fields use pointers so the caller controls exactly what is sent in a
// PATCH body (a nil pointer is omitted; a non-nil pointer is always sent, even
// for the zero value such as `false`). Read-only response fields are plain
// values with `omitempty` so they never appear in request bodies.

// StatusPage is a status page belonging to the authenticated organization.
type StatusPage struct {
	ID                    string                `json:"id,omitempty"`
	Name                  string                `json:"name,omitempty"`
	Subdomain             string                `json:"subdomain,omitempty"`
	Timezone              string                `json:"timezone,omitempty"`
	WebsiteURL            string                `json:"website_url,omitempty"`
	RequireAuthentication *bool                 `json:"require_authentication,omitempty"`
	CurrentStatus         string                `json:"current_status,omitempty"`
	NotificationSettings  *NotificationSettings `json:"notification_settings,omitempty"`
	CreatedAt             string                `json:"created_at,omitempty"`
	UpdatedAt             string                `json:"updated_at,omitempty"`
}

// NotificationSettings is the nested notification configuration of a status page.
type NotificationSettings struct {
	EmailEnabled   *bool `json:"email_enabled,omitempty"`
	SlackEnabled   *bool `json:"slack_enabled,omitempty"`
	RSSFeedEnabled *bool `json:"rss_feed_enabled,omitempty"`
}

// Service is a service displayed on a status page.
type Service struct {
	ID          string  `json:"id,omitempty"`
	Slug        string  `json:"slug,omitempty"`
	Name        string  `json:"name,omitempty"`
	Description *string `json:"description,omitempty"`
	Order       *int64  `json:"order,omitempty"`
	// Status is the computed worst-case status across containers (read-only).
	Status string `json:"status,omitempty"`
	// ContainerStatuses is read-only on responses. On create it may be supplied
	// to set per-container initial statuses; see ServiceCreateRequest.
	ContainerStatuses []ContainerStatusEntry `json:"container_statuses,omitempty"`
	CreatedAt         string                 `json:"created_at,omitempty"`
	UpdatedAt         string                 `json:"updated_at,omitempty"`
}

// ContainerStatusEntry is a per-container status, used in responses and in the
// optional create-time container_statuses input.
type ContainerStatusEntry struct {
	ContainerSlug string `json:"container_slug"`
	Status        string `json:"status"`
}

// Container groups services on a status page (e.g. by region).
type Container struct {
	ID        string                    `json:"id,omitempty"`
	Slug      string                    `json:"slug,omitempty"`
	Name      string                    `json:"name,omitempty"`
	Services  []ContainerServiceSummary `json:"services,omitempty"`
	CreatedAt string                    `json:"created_at,omitempty"`
	UpdatedAt string                    `json:"updated_at,omitempty"`
}

// ContainerServiceSummary is a service entry within a container response.
type ContainerServiceSummary struct {
	Slug   string `json:"slug"`
	Name   string `json:"name"`
	Status string `json:"status"`
}

// OutgoingWebhook is an organization-level webhook subscription.
type OutgoingWebhook struct {
	ID                   string   `json:"id,omitempty"`
	Name                 string   `json:"name,omitempty"`
	URL                  string   `json:"url,omitempty"`
	Events               []string `json:"events,omitempty"`
	Enabled              *bool    `json:"enabled,omitempty"`
	AllStatusPages       *bool    `json:"all_status_pages,omitempty"`
	StatusPageSubdomains []string `json:"status_page_subdomains,omitempty"`
	LastTriggeredAt      *string  `json:"last_triggered_at,omitempty"`
	LastAttemptedAt      *string  `json:"last_attempted_at,omitempty"`
	// Secret is the plaintext signing secret, returned only on create and
	// regenerate_secret responses.
	Secret    string `json:"secret,omitempty"`
	CreatedAt string `json:"created_at,omitempty"`
	UpdatedAt string `json:"updated_at,omitempty"`
}

// MonitoringCheck is an uptime monitoring check belonging to the organization.
// It optionally drives status-page incident automation via the Automation field.
type MonitoringCheck struct {
	ID   string `json:"id,omitempty"`
	Name string `json:"name,omitempty"`
	// URL is the full target; its scheme selects the check type (tcp:// → TCP, http(s):// → HTTP).
	URL string `json:"url,omitempty"`
	// CheckType is derived server-side from URL (read-only).
	CheckType                string   `json:"check_type,omitempty"`
	HTTPMethod               *string  `json:"http_method,omitempty"`
	RecvTimeoutSecs          *int64   `json:"recv_timeout_secs,omitempty"`
	GeoAreas                 []string `json:"geo_areas,omitempty"`
	RecipientEmails          []string `json:"recipient_emails,omitempty"`
	DisplayResponseTimeChart *bool    `json:"display_response_time_chart,omitempty"`
	// Status is the last status reported by the monitoring service (read-only).
	Status *string `json:"status,omitempty"`
	// Automation links the check to a status-page service. It is intentionally
	// sent without omitempty so an explicit null disables automation on update.
	Automation *MonitoringCheckAutomation `json:"automation"`
	CreatedAt  string                     `json:"created_at,omitempty"`
	UpdatedAt  string                     `json:"updated_at,omitempty"`
}

// MonitoringCheckAutomation identifies the status-page service a check drives.
type MonitoringCheckAutomation struct {
	StatusPageSubdomain string `json:"status_page_subdomain"`
	ContainerSlug       string `json:"container_slug"`
	ServiceSlug         string `json:"service_slug"`
}

// Automation is a webhook-driven incident automation on a status page.
type Automation struct {
	ID              string `json:"id,omitempty"`
	ServiceSlug     string `json:"service_slug,omitempty"`
	ContainerSlug   string `json:"container_slug,omitempty"`
	ManageIncidents *bool  `json:"manage_incidents,omitempty"`
	// Secret is write-only; the API never returns it.
	Secret *string `json:"secret,omitempty"`
	// HasSecret reports whether a secret is configured (read-only).
	HasSecret        bool              `json:"has_secret,omitempty"`
	AutomationFormat *AutomationFormat `json:"automation_format,omitempty"`
	// TriggerURL is the URL external monitors POST to (read-only).
	TriggerURL string `json:"trigger_url,omitempty"`
	CreatedAt  string `json:"created_at,omitempty"`
	UpdatedAt  string `json:"updated_at,omitempty"`
}

// AutomationFormat is the JSONPath rule used to interpret a trigger payload.
// Name and Custom are read-only response fields.
type AutomationFormat struct {
	Name               string  `json:"name,omitempty"`
	ExpectedResultPath string  `json:"expected_result_path,omitempty"`
	ExpectedResult     string  `json:"expected_result,omitempty"`
	SecretPath         *string `json:"secret_path,omitempty"`
	Custom             bool    `json:"custom,omitempty"`
}

// BoolPtr is a small helper for building pointer-valued request fields.
func BoolPtr(b bool) *bool { return &b }

// StringPtr is a small helper for building pointer-valued request fields.
func StringPtr(s string) *string { return &s }

// Int64Ptr is a small helper for building pointer-valued request fields.
func Int64Ptr(i int64) *int64 { return &i }
