data "statuspal-next_status_pages" "all" {}

output "status_page_subdomains" {
  value = [for p in data.statuspal-next_status_pages.all.status_pages : p.subdomain]
}
