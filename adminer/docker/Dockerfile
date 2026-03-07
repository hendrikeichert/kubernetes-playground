FROM adminer:4.8.1

USER root

RUN apt-get update \
    && apt-get install -y --no-install-recommends php-mongodb \
    && php -m | grep -i mongodb \
    && apt-get clean \
    && rm -rf /var/lib/apt/lists/* \
    && rm -rf /tmp/pear