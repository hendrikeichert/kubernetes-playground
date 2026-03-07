#!/bin/bash

set -euo pipefail

# change into script directory
cd "$(dirname "$0")"

TLS_CERT_PATH="../tls/tls.crt"
TLS_KEY_PATH="../tls/tls.key"
TLS_SECRET_NAME="camunda-platform-tls"
TRAEFIK_MIDDLEWARE_PATH="./middleware.yaml"

ensure_tls_secret() {
    if [[ ! -f "$TLS_CERT_PATH" || ! -f "$TLS_KEY_PATH" ]]; then
        echo "TLS files not found: $TLS_CERT_PATH and/or $TLS_KEY_PATH"
        exit 1
    fi

    echo "Creating/updating TLS secret '$TLS_SECRET_NAME' in namespace camunda..."
    kubectl create secret tls "$TLS_SECRET_NAME" \
        --namespace camunda \
        --cert="$TLS_CERT_PATH" \
        --key="$TLS_KEY_PATH" \
        --dry-run=client -o yaml | kubectl apply -f -
}

ensure_traefik_middleware() {
    if [[ -f "$TRAEFIK_MIDDLEWARE_PATH" ]]; then
        echo "Applying Traefik middleware from $TRAEFIK_MIDDLEWARE_PATH..."
        kubectl apply -f "$TRAEFIK_MIDDLEWARE_PATH"
    fi
}

install() {
    echo "Installing Camunda Platform..."
    helm install camunda camunda/camunda-platform \
        --namespace camunda \
        -f values.yaml
}

upgrade() {
    echo "Upgrading Camunda Platform..."
    ensure_tls_secret
    ensure_traefik_middleware
    helm upgrade camunda camunda/camunda-platform \
        --namespace camunda \
        -f values.yaml
}

#install
upgrade