terraform {
  required_providers {
    statuspal-next = {
      source = "statuspal/statuspal-next"
    }
  }
}

provider "statuspal-next" {
  # API key may also be supplied via the STATUSPAL_NEXT_API_KEY environment variable.
  api_key = var.statuspal_next_api_key

  # Optional. Defaults to https://next.statuspal.io/api/v1. May also be supplied
  # via the STATUSPAL_NEXT_ENDPOINT environment variable. Point this at a local
  # instance for development, e.g. "http://spage.test:7070/api/v1".
  # endpoint = "http://spage.test:7070/api/v1"
}

variable "statuspal_next_api_key" {
  type      = string
  sensitive = true
}
