# A simple HTTP uptime check.
resource "statuspal-next_monitoring_check" "api" {
  name              = "API Gateway health"
  url               = "https://api.acme.com/health"
  http_method       = "get"
  recv_timeout_secs = 5
  geo_areas         = ["US", "EU"]
  recipient_emails  = ["ops@acme.com"]
}

# A TCP check that also drives status-page incident automation: when it goes
# down, an incident is opened on the "api" service in the "eu" container, and
# resolved when it recovers.
resource "statuspal-next_monitoring_check" "db" {
  name             = "Primary database"
  url              = "tcp://db.acme.com:5432"
  recipient_emails = ["ops@acme.com"]

  automation = {
    status_page_subdomain = "acme-status"
    container_slug        = "eu"
    service_slug          = "api"
  }
}
