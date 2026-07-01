data "statuspal-next_services" "acme" {
  status_page_subdomain = "acme-status"
}

output "service_slugs" {
  value = [for s in data.statuspal-next_services.acme.services : s.slug]
}
