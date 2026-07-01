data "statuspal-next_outgoing_webhooks" "all" {}

output "webhook_urls" {
  value = [for w in data.statuspal-next_outgoing_webhooks.all.outgoing_webhooks : w.url]
}
