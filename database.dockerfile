# PostgreSQL Dockerfile for GO-API Database
FROM postgres:17-alpine

# Set environment variables
ENV POSTGRES_DB=postgres
ENV PG_USER=postgres
ENV POSTGRES_PASSWORD=postgres

# Copy the DDL script to the initialization directory
COPY opt/dump.sql /docker-entrypoint-initdb.d/

# Set proper permissions
RUN chmod 644 /docker-entrypoint-initdb.d/dump.sql

# Expose PostgreSQL port
EXPOSE 5432

# The postgres image automatically runs scripts in /docker-entrypoint-initdb.d/
# when the container starts for the first time
