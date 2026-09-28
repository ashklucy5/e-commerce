package analytics

// V0Mode documents the intentional Analytics v0 architecture.
//
// Analytics v0 is read-only and computes bounded aggregate read models
// directly from PostgreSQL. It does not consume storefront events or add
// analytics write traffic to customer request paths.
const V0Mode = "read_only_postgres_aggregates"
