#!/bin/sh
# Production-ready entrypoint script

# Exit immediately if a command exits with a non-zero status.
set -e

# These variables should be passed from your environment (e.g., docker-compose.yml)
DB_HOST=${ENSSI_DATABASE_HOST:-db}
DB_PORT=${ENSSI_DATABASE_PORT:-5432}
DB_USER=${ENSSI_DATABASE_USER:-postgres}
MIGRATE_PATH="/app/migrate"
SEEDER_PATH="/app/seeder"

# --- Wait for Database ---
# This loop will wait until the PostgreSQL database is ready to accept connections.
echo "Entrypoint: Waiting for database at $DB_HOST:$DB_PORT..."
# The `pg_isready` command is a standard PostgreSQL utility.
# We use `until` to keep trying until the command succeeds.
until pg_isready -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -q; do
  >&2 echo "Postgres is unavailable - sleeping"
  sleep 1
done
echo "Entrypoint: Database is ready."

# --- Run Migrations ---
# It's good practice to only run migrations from one service to avoid race conditions.
# We'll designate the 'bot' service as the one responsible for migrations.
# The 'run_migrations' variable can be set to 'true' on the service that should handle this.
if [ "$run_migrations" = "true" ]; then
  echo "Entrypoint: This container is designated to run migrations."
  echo "Entrypoint: Running database migrations..."
  $MIGRATE_PATH up
  echo "Entrypoint: Migrations finished."

  # --- Run Seeder (only after migrations) ---
  echo "Entrypoint: Running database seeder..."
  $SEEDER_PATH
  echo "Entrypoint: Seeder finished."
else
  echo "Entrypoint: Skipping migrations and seeding on this container."
fi


# --- Execute Main Application ---
# Finally, execute the main command that was passed to the container.
# This will be "./enssi-bot" or "./enssi-worker" from your docker-compose file.
echo "Entrypoint: Handing over to main application..."
exec "$@"
