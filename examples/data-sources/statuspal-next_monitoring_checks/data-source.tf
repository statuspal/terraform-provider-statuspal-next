# Lists every monitoring check in the organization.
data "statuspal-next_monitoring_checks" "all" {}

output "check_names" {
  value = [for c in data.statuspal-next_monitoring_checks.all.monitoring_checks : c.name]
}
