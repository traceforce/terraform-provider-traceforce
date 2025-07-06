output "dataset_id" {
  description = "The dataset ID"
  value       = google_bigquery_dataset.dataset.dataset_id
}

output "self_link" {
  description = "The URI of the created resource"
  value       = google_bigquery_dataset.dataset.self_link
}
