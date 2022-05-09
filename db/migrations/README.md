We use the golang-migrate repo (https://github.com/golang-migrate/migrate) to handle migrations.

The version numbers are UTC timestamps in YYYYMMDDHHmm format.

To create a set of migrations use the following command:

```
docker-compose run --rm goutils migrate -database "postgres://postgres:pass@db:5432/liwords?sslmode=disable" -verbose  create -dir db/migrations -format 200601021504 -ext sql {name-of-migration}
```

Replacing `{name-of-migration}` with your chosen name.
