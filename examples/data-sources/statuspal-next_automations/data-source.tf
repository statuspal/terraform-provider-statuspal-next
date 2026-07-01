# Lists the incident automations on a status page.
data "statuspal-next_automations" "acme" {
  status_page_subdomain = "acme-status"
}

output "automation_trigger_urls" {
  value = [for a in data.statuspal-next_automations.acme.automations : a.trigger_url]
}
