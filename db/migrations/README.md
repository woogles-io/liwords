We use the golang-migrate repo (https://github.com/golang-migrate/migrate) to handle migrations.

The version numbers are UTC timestamps in YYYYMMDDHHmm format.

### Creating a migration

To create a set of migrations use the following command:

```
docker-compose run --rm goutils migrate -database "postgres://postgres:pass@db:5432/liwords?sslmode=disable" -verbose  create -dir db/migrations -format 200601021504 -ext sql NAME_OF_MIGRATION
```

Replacing `NAME_OF_MIGRATION` with your chosen name.

If you're using Linux you may need to change the permissions of the generated file. You can do so with:

`sudo chown -R ${USER}:${USER} db/migrations`

### Down migration

If you wish to run a down migration locally (for example, your migration file was missing some stuff and you want to migrate down before migrating back up after adding what you were missing):

```
docker-compose run --rm goutils migrate -database "postgres://postgres:pass@db:5432/liwords?sslmode=disable" -source file://./db/migrations down 1
```

### Up migration

You can either run the above command but replace `down` with `up`, or you can just start the `app` server which automatically runs missing migrations.