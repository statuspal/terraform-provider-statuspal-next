resource "statuspal-next_status_page" "acme" {
  name        = "Acme Status"
  subdomain   = "acme-status"
  timezone    = "America/New_York"
  website_url = "https://acme.com"
}

resource "statuspal-next_service" "api" {
  status_page_subdomain = statuspal-next_status_page.acme.subdomain
  name                  = "API Gateway"
  description           = "Public REST API"
  # slug is auto-generated from name when omitted.
}
