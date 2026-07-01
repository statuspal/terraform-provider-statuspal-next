data "statuspal-next_containers" "acme" {
  status_page_subdomain = "acme-status"
}

output "container_names" {
  value = [for c in data.statuspal-next_containers.acme.containers : c.name]
}
