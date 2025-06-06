#!/bin/bash
set -e

# Configure SSL in postgresql.conf
cat >> "$PGDATA/postgresql.conf" <<EOF
ssl = on
ssl_cert_file = '/etc/postgresql/certs/server.crt'
ssl_key_file = '/etc/postgresql/certs/server.key'
EOF

# Configure pg_hba.conf for SSL connections
cat >> "$PGDATA/pg_hba.conf" <<EOF
hostssl all all 0.0.0.0/0 md5
EOF