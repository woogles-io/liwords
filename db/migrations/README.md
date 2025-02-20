We use the golang-migrate repo (https://github.com/golang-migrate/migrate) to handle migrations.

The version numbers are UTC timestamps in YYYYMMDDHHmm format.

### Creating a migration

To create a set of migrations use the following command:

```
./gen-migration.sh {name-of-migration}
```

Replacing `{name-of-migration}` with your chosen name.


### Down migration

If you wish to run a down migration locally (for example, your migration file was missing some stuff and you want to migrate down before migrating back up after adding what you were missing):

```
./migrate-down.sh {number-of-migrations}
```

Replacing `{number_of_migrations}` with how many migrations you want to undo.

### Up migration

Starting the `app` server automatically runs missing migrations.