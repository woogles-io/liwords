We use the golang-migrate repo (https://github.com/golang-migrate/migrate) to handle migrations.

The version numbers are UTC timestamps in YYYYMMDDHHmm format.

To create a set of migrations use the following command locally whilst docker-compose is running:

```
migrate -database "postgres://postgres:pass@localhost:5432/liwords?sslmode=disable" -verbose  create -dir db/migrations -format 200601021504 -ext sql {name-of-migration}
```

Replacing `{name-of-migration}` with your chosen name.
