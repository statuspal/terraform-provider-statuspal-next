resource "statuspal-next_automation" "api" {
  status_page_subdomain = "acme-status"
  service_slug          = "api"
  container_slug        = "eu"

  # When true, a failing trigger opens an incident (and a healthy one resolves it).
  # When false, only the service status is flipped.
  manage_incidents = true

  # Optional shared secret validated against the request (see secret_path below).
  secret = var.automation_secret

  automation_format = {
    expected_result_path = "$.status"
    expected_result      = "ok"
    secret_path          = "$.token" # or "h:X-Webhook-Secret" to read a header
  }
}

# Point your external monitor at this URL.
output "api_automation_trigger_url" {
  value = statuspal-next_automation.api.trigger_url
}
