data "external_schema" "gorm" {
  program = [
    "go",
    "run",
    "-mod=mod",
    "ariga.io/atlas-provider-gorm",
    "load",
    "--path", "./src/models",
    "--dialect", "postgres", // | postgres | sqlite | sqlserver
  ]
}
data "composite_schema" "app" {
  # Load the GORM model first
  schema "public" {
    url = data.external_schema.gorm.url
  }
  # Next, load the RLS schema.
  schema "public" {
    url = "file://database/rls.sql"
  }
}
locals {
  db_pass = urlescape(getenv("DB_PASSWORD"))
}
env "gorm" {
  src = data.external_schema.gorm.url
  url = "postgresql://postgres:${local.db_pass}@localhost:5432/ebsdb?search_path=public&sslmode=disable"
  dev = "docker://postgres/15/dev?search_path=public"
  migration {
    dir = "file://migrations"
  }
  format {
    migrate {
      diff = "{{ sql . \"  \" }}"
    }
  }
}
env "local" {
  src = data.composite_schema.app.url
  dev = "docker://postgres/15/dev?search_path=public"
}