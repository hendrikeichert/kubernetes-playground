# Kubernetes - K3s
https://github.com/k3s-io/k3s/releases

# Nginx Proxy Manager
https://nginxproxymanager.com/

# install ingress-nginx controller
helm upgrade --install ingress-nginx ingress-nginx \
  --repo https://kubernetes.github.io/ingress-nginx \
  --namespace ingress-nginx --create-namespace
