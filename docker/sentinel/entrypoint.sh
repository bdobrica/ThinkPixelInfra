#!/bin/bash

# Function to resolve a hostname to an IP
resolve_hostname() {
    local hostname=$1
    getent hosts "$hostname" | awk '{ print $1 }'
}

# Scan the sentinel.conf file for master hostnames and replace them with IPs
echo "Scanning sentinel.conf for master hostnames..."
while IFS= read -r line; do
    if [[ $line == sentinel*monitor* ]]; then
        # Extract the hostname from the monitor line
        master_host=$(echo "$line" | awk '{ print $4 }')
        master_ip=$(resolve_hostname "$master_host")

        if [[ -n $master_ip ]]; then
            echo "Resolved $master_host to $master_ip"
            # Replace the hostname with the resolved IP in sentinel.conf
            sed -i "s/$master_host/$master_ip/g" /etc/sentinel.conf
        else
            echo "Failed to resolve $master_host, exiting."
            exit 1
        fi
    fi
done < /etc/sentinel.conf

# Start Redis Sentinel
echo "Starting Redis Sentinel with updated sentinel.conf..."
exec redis-sentinel /etc/sentinel.conf
