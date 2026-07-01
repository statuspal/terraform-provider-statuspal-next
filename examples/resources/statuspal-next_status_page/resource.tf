resource "statuspal-next_status_page" "acme" {
  name        = "Acme Status"
  subdomain   = "acme-status"
  timezone    = "America/New_York"
  website_url = "https://acme.com"

  require_authentication = false

  notification_settings = {
    email_enabled    = true
    slack_enabled    = false
    rss_feed_enabled = true
  }
}
