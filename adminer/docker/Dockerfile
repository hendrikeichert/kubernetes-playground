FROM adminer:4.8.1

USER root

RUN if command -v apk >/dev/null 2>&1; then \
            apk add --no-cache php84-dev gcc musl-dev make openssl-dev; \
            pecl install mongodb; \
            echo "extension=mongodb.so" > /usr/local/etc/php/conf.d/mongodb.ini; \
            php -m | grep -i mongodb; \
            apk del php84-dev gcc musl-dev make openssl-dev; \
            rm -rf /tmp/pear /var/cache/apk/*; \
        elif command -v apt-get >/dev/null 2>&1; then \
            apt-get update; \
            apt-get install -y --no-install-recommends php-mongodb; \
            php -m | grep -i mongodb; \
            apt-get clean; \
            rm -rf /var/lib/apt/lists/* /tmp/pear; \
        else \
            echo "Unsupported base image: no apk or apt-get found"; \
            exit 1; \
        fi