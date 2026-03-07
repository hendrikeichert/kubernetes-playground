FROM postgres:17

# Install openssl for certificate generation
USER root
RUN apt-get update && apt-get install -y openssl && rm -rf /var/lib/apt/lists/*

# Create directory for certificates
RUN mkdir /etc/postgresql/certs && chown postgres:postgres /etc/postgresql/certs && chmod 700 /etc/postgresql/certs

# Generate self-signed certificates
USER postgres
RUN openssl req -x509 -nodes -days 365 -newkey rsa:2048 \
    -keyout /etc/postgresql/certs/server.key \
    -out /etc/postgresql/certs/server.crt \
    -subj "/C=US/ST=State/L=City/O=Organization/OU=Unit/CN=postgres.obligations.svc.cluster.local" && \
    chmod 600 /etc/postgresql/certs/server.*

# Create initialization script for SSL configuration
COPY --chown=postgres:postgres init-ssl.sh /docker-entrypoint-initdb.d/init-ssl.sh
RUN chmod +x /docker-entrypoint-initdb.d/init-ssl.sh

# Ensure entrypoint is executable
USER root
RUN chmod +x /usr/local/bin/docker-entrypoint.sh

# Switch back to postgres user
USER postgres

# Expose PostgreSQL port
EXPOSE 5432

# Explicitly set entrypoint and command
ENTRYPOINT ["/usr/local/bin/docker-entrypoint.sh"]
CMD ["postgres"]