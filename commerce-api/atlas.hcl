env "local" {
  url = getenv("ATLAS_DB_URL")

  dev = "docker://postgres/18/dev?search_path=public"

  schema {
    src = "file://db/schema.pg.hcl"
  }

  migration {
    dir = "file://db/migrations"
  }
}