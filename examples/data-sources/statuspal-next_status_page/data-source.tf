data "statuspal-next_status_page" "acme" {
  subdomain = "acme-status"
}

output "acme_current_status" {
  value = data.statuspal-next_status_page.acme.current_status
}
