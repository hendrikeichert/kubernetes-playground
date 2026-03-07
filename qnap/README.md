# QNAP k3s
/share/CACHEDEV1_DATA/.qpkg/container-station/var/lib/k3s
/share/CACHEDEV2_DATA/VMs/container-station-data/application/k3s
/share/CACHEDEV1_DATA/.qpkg/container-station/var/lib/k3s/etc/rancher/k3s/k3s.yaml
/share/VMs/container-station-data/application/

docker compose -f /share/CACHEDEV2_DATA/VMs/container-station-data/application/k3s/docker-compose.yml up -d
docker compose -f /share/CACHEDEV2_DATA/VMs/container-station-data/application/k3s/docker-compose.yml down

vi /share/CACHEDEV2_DATA/VMs/container-station-data/application/k3s/docker-compose.yml
docker cp k3s-server-1:/output/kubeconfig.yaml .

# NPM
/share/CACHEDEV1_DATA/.qpkg/container-station/data/application/npm
dev@benarco.net / adminadmin

# links
docker registry: 192.168.1.100:32768
gitea: 192.168.1.100:3003
