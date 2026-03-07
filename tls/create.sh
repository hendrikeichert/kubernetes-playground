#!/bin/bash

create_certs() {
    # if files exist, skip creation, otherwise create new ones
    if [[ -f tls.crt && -f tls.key ]]; then
        echo "tls.crt and tls.key already exist, skipping creation"
        return
    fi

    cat > openssl.cnf <<'EOF'
[req]
distinguished_name = req_distinguished_name
req_extensions = v3_req
prompt = no

[req_distinguished_name]
CN = k3s.local

[v3_req]
basicConstraints = CA:FALSE
keyUsage = digitalSignature, keyEncipherment
extendedKeyUsage = serverAuth
subjectAltName = @alt_names

[alt_names]
DNS.1 = k3s.local
DNS.2 = *.k3s.local
EOF
    openssl req -x509 -newkey rsa:4096 -keyout tls.key -out tls.crt -days 365 -nodes -config openssl.cnf -extensions v3_req
}


create_secret() {
    # if secret exists, delete it first
    if kubectl get secret wildcard-tls -n camunda > /dev/null 2>&1; then
        kubectl delete secret wildcard-tls -n camunda
    fi
    kubectl create secret tls wildcard-tls --cert=tls.crt --key=tls.key -n camunda
}

script_dir=$(dirname "$0")
cd "$script_dir" || exit

create_certs
create_secret