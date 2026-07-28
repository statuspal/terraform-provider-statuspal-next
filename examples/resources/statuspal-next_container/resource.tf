resource "statuspal-next_status_page" "acme" {
  name        = "Acme Status"
  subdomain   = "acme-status"
  timezone    = "America/New_York"
  website_url = "https://acme.com"
}

resource "statuspal-next_container" "eu" {
  status_page_subdomain = statuspal-next_status_page.acme.subdomain
  name                  = "EU Region"
}

# Every status page is created with a "Default" container automatically. Import
# it instead of declaring a second container with the same name:
#   terraform import statuspal-next_container.default acme-status/default
