#!/bin/bash

# Function to start a specific exporter
start_exporter() {
    local exporter_name=$1

    case $exporter_name in
        "node-exporter")
            echo "Starting Node Exporter..."
            docker run -d --name=node-exporter --network=host prom/node-exporter
            ;;
        "process-exporter")
            echo "Starting Process Exporter..."
            docker run -d --name=process-exporter --network=host ncabatoff/process-exporter
            ;;
        "otel-collector")
            echo "Starting OpenTelemetry Collector..."
            docker run -d --name=otel-collector \
                -v /path/to/config.yaml:/etc/otel/config.yaml \
                --network=host otel/opentelemetry-collector
            ;;
        *)
            echo "Unknown exporter: $exporter_name"
            ;;
    esac
}

# Main script
if [ "$#" -lt 1 ]; then
    echo "Usage: $0 <exporter_name> [<exporter_name>...]"
    exit 1
fi

for exporter in "$@"; do
    start_exporter "$exporter"
done

