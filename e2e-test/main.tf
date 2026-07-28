terraform {
  required_providers {
    statuspal-next = {
      source = "statuspal/statuspal-next"
    }
  }
}

# api_key/endpoint come from STATUSPAL_NEXT_API_KEY / STATUSPAL_NEXT_ENDPOINT,
# or set them explicitly here via the variables below.
provider "statuspal-next" {
  api_key  = var.statuspal_next_api_key
  endpoint = var.statuspal_next_endpoint
}

variable "statuspal_next_api_key" {
  description = "Organization API key (sk_...). Prefer STATUSPAL_NEXT_API_KEY env var."
  type        = string
  sensitive   = true
  default     = null
}

variable "statuspal_next_endpoint" {
  description = "Management API base URL. Prefer STATUSPAL_NEXT_ENDPOINT env var."
  type        = string
  default     = null
}

variable "subdomain" {
  description = "Status page subdomain to create (must be unique in the org)."
  type        = string
  default     = "tf-e2e-test"
}

variable "monitoring_recipient_email" {
  description = "Email of an existing organization user to notify on check status changes."
  type        = string
  default     = "jorge@statuspal.io"
}

# --- Exercise the full resource set against a real backend ---------------

resource "statuspal-next_status_page" "test" {
  name        = "Terraform E2E Test"
  subdomain   = var.subdomain
  timezone    = "Etc/UTC"
  website_url = "https://example.com"

  require_authentication = false

  notification_settings = {
    email_enabled    = true
    slack_enabled    = false
    rss_feed_enabled = true
  }
}

resource "statuspal-next_container" "eu" {
  status_page_subdomain = statuspal-next_status_page.test.subdomain
  name                  = "EU Region"
}

resource "statuspal-next_container" "us" {
  status_page_subdomain = statuspal-next_status_page.test.subdomain
  name                  = "US Region"
}

# Services depend on the containers so they are created after the full set of
# containers exists. spage builds the container×service grid at create time from
# whatever already exists, so creating services and containers concurrently can
# leave grid cells missing (see the container-grid race). Serializing avoids it.
resource "statuspal-next_service" "api" {
  status_page_subdomain = statuspal-next_status_page.test.subdomain
  name                  = "API Gateway"
  description           = "Public REST API"

  depends_on = [statuspal-next_container.eu, statuspal-next_container.us]
}

resource "statuspal-next_service" "web" {
  status_page_subdomain = statuspal-next_status_page.test.subdomain
  name                  = "Web Dashboard"
  description           = "Customer-facing web app"

  depends_on = [statuspal-next_container.eu, statuspal-next_container.us]
}

resource "statuspal-next_outgoing_webhook" "ops" {
  name = "Ops alerts (e2e)"
  url  = "https://hooks.example.com/statuspal"

  events = [
    "notice.created",
    "service.status_changed",
  ]

  enabled                = true
  all_status_pages       = false
  status_page_subdomains = [statuspal-next_status_page.test.subdomain]
}

# A monitoring check that drives status-page incident automation: when it goes
# down, an incident opens on the API Gateway service in the EU container.
resource "statuspal-next_monitoring_check" "api_health" {
  name              = "API Gateway health (e2e)"
  url               = "https://api.example.com/health"
  http_method       = "get"
  recv_timeout_secs = 5
  geo_areas         = ["US", "EU"]
  recipient_emails  = [var.monitoring_recipient_email]

  automation = {
    status_page_subdomain = statuspal-next_status_page.test.subdomain
    container_slug        = statuspal-next_container.eu.slug
    service_slug          = statuspal-next_service.api.slug
  }
}

# A webhook-driven incident automation with an inline custom JSONPath format.
resource "statuspal-next_automation" "api" {
  status_page_subdomain = statuspal-next_status_page.test.subdomain
  service_slug          = statuspal-next_service.api.slug
  container_slug        = statuspal-next_container.eu.slug
  manage_incidents      = true

  automation_format = {
    expected_result_path = "$.status"
    expected_result      = "ok"
    secret_path          = "$.token"
  }
}

# --- Read back via data sources to confirm round-trip --------------------

data "statuspal-next_services" "all" {
  status_page_subdomain = statuspal-next_status_page.test.subdomain

  depends_on = [statuspal-next_service.api, statuspal-next_service.web]
}

data "statuspal-next_monitoring_checks" "all" {
  depends_on = [statuspal-next_monitoring_check.api_health]
}

data "statuspal-next_automations" "all" {
  status_page_subdomain = statuspal-next_status_page.test.subdomain

  depends_on = [statuspal-next_automation.api]
}

output "status_page_id" {
  value = statuspal-next_status_page.test.id
}

output "service_slug" {
  value = statuspal-next_service.api.slug
}

output "service_count" {
  value = length(data.statuspal-next_services.all.services)
}

output "webhook_signing_secret" {
  value     = statuspal-next_outgoing_webhook.ops.secret
  sensitive = true
}

output "monitoring_check_count" {
  value = length(data.statuspal-next_monitoring_checks.all.monitoring_checks)
}

output "automation_trigger_url" {
  value = statuspal-next_automation.api.trigger_url
}
