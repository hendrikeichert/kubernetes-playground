FROM adminer:latest
# see https://github.com/TimWolla/docker-adminer/blob/0c0fd187e8646ad61fe592a80d66abec2c1dc951/4/Dockerfile

USER root

# Install build dependencies and PHP MongoDB extension
RUN apk add --no-cache \
    php84-dev \
    gcc \
    musl-dev \
    make \
    openssl-dev \
    && pecl install mongodb \
    && echo "extension=mongodb.so" > /usr/local/etc/php/conf.d/mongodb.ini \
    && php -m | grep mongodb || { echo "MongoDB not loaded!"; exit 1; } \
    && rm -rf /var/cache/apk/*

ENTRYPOINT	[ "entrypoint.sh", "docker-php-entrypoint" ]

USER	adminer
CMD	[ "php", "-S", "[::]:8080", "-t", "/var/www/html" ]

EXPOSE 8080