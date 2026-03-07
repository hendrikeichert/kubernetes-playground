docker run --rm \
    -v $(pwd)/data:/app/data \
    -p 9080:9080 \
    --name link-manager \
    link-manager