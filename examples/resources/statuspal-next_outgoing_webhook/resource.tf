resource "statuspal-next_outgoing_webhook" "ops_alerts" {
  name = "Ops alerts"
  url  = "https://hooks.example.com/statuspal"

  events = [
    "notice.created",
    "notice.updated",
    "service.status_changed",
  ]

  enabled = true

  # Scope to specific status pages. Omit (or set all_status_pages = true) to fire
  # for every status page in the organization.
  all_status_pages       = false
  status_page_subdomains = ["acme-status"]
}

# The signing secret is returned only on create and stored in state.
output "webhook_signing_secret" {
  value     = statuspal-next_outgoing_webhook.ops_alerts.secret
  sensitive = true
}
