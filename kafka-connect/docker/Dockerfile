FROM confluentinc/cp-kafka-connect:7.8.0

USER root

# Install the JDBC connector plugin
RUN confluent-hub install --no-prompt confluentinc/kafka-connect-jdbc:latest \
    && confluent-hub install --no-prompt debezium/debezium-connector-postgresql:latest \
    && rm -f /usr/share/confluent-hub-components/confluentinc-kafka-connect-jdbc/lib/postgresql-*.jar

# Add the PostgreSQL JDBC driver
COPY postgresql-42.7.5.jar  /usr/share/confluent-hub-components/confluentinc-kafka-connect-jdbc/lib/
# https://jdbc.postgresql.org/download/postgresql-42.7.5.jar

USER appuser