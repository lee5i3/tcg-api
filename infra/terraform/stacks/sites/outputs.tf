output "www_bucket" {
  value = module.www_site.bucket
}

output "www_distribution_id" {
  value = module.www_site.distribution_id
}

output "www_url" {
  description = "Marketing site URL — the product's front door"
  value       = module.www_site.url
}

output "app_bucket" {
  value = module.app_site.bucket
}

output "app_distribution_id" {
  value = module.app_site.distribution_id
}

output "app_url" {
  description = "Public app URL — serves the SPA and proxies /v1/* to the API"
  value       = module.app_site.url
}

output "admin_bucket" {
  value = module.admin_site.bucket
}

output "admin_distribution_id" {
  value = module.admin_site.distribution_id
}

output "admin_url" {
  description = "Admin site URL (separate distribution from the public app)"
  value       = module.admin_site.url
}
