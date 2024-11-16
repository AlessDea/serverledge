docker run -d \
  --name node-exporter \
  --net="host" \
  --pid="host" \
  quay.io/prometheus/node-exporter:latest \
  --path.rootfs=/

