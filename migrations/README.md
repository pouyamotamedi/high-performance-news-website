# Database Migrations

The complete database schema is maintained in `deployment/init-db.sql`.

This file is auto-generated from the production database using:

```bash
docker exec -e PGPASSWORD=$DB_PASSWORD <container_name> pg_dump -U <user> -d <dbname> --schema-only --no-owner --no-privileges > deployment/init-db.sql
```

## For New Installations

The `init-db.sql` file is automatically executed when the PostgreSQL container starts for the first time. No manual migration is needed.

## For Schema Updates

If you need to update the schema on an existing database:

1. Make changes directly to the database
2. Export the updated schema:
   ```bash
   docker exec -e PGPASSWORD=$DB_PASSWORD <container> pg_dump -U <user> -d <db> --schema-only --no-owner --no-privileges > deployment/init-db.sql
   ```
3. Commit and push the updated `init-db.sql`

## Notes

- Individual migration files are no longer used
- The `init-db.sql` contains the complete schema including all tables, indexes, functions, views, and partitions
- New site installations automatically get the full schema
