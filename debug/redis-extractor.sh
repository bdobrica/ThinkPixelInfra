#!/bin/bash

# Default values
NAMESPACE="thinkpixel"
POD_NAME="redis-extractor-pod"
CONFIGMAP_NAME="redis-extractor-config"
SCRIPT_NAME="redis-extractor.py"

# Default Redis connection parameters
REDIS_HOST="localhost"
REDIS_PORT=6379
REDIS_DB=0
REDIS_PASSWORD=""
OUTPUT_FILE="redis-dump.json"

# Parse command-line arguments
while getopts "H:p:d:P:o:" opt; do
  case ${opt} in
    H ) REDIS_HOST=$OPTARG ;;
    p ) REDIS_PORT=$OPTARG ;;
    d ) REDIS_DB=$OPTARG ;;
    P ) REDIS_PASSWORD=$OPTARG ;;
    o ) OUTPUT_FILE=$OPTARG ;;
    \? ) echo "Usage: $0 [-H redis_host] [-p redis_port] [-d redis_db] [-P redis_password] [-o output_file]"
         exit 1 ;;
  esac
done

# Create a ConfigMap with the Python script
kubectl delete configmap $CONFIGMAP_NAME --namespace $NAMESPACE --ignore-not-found
kubectl create configmap $CONFIGMAP_NAME --from-file=$SCRIPT_NAME --namespace $NAMESPACE

# Deploy a Pod with a writable `emptyDir` volume
kubectl delete pod $POD_NAME --namespace $NAMESPACE --ignore-not-found
kubectl apply -f - <<EOF
apiVersion: v1
kind: Pod
metadata:
  name: $POD_NAME
  namespace: $NAMESPACE
spec:
  containers:
  - name: python-container
    image: python:3.9
    command: ["sleep", "3600"]
    env:
    - name: REDIS_HOST
      value: "$REDIS_HOST"
    - name: REDIS_PORT
      value: "$REDIS_PORT"
    - name: REDIS_DB
      value: "$REDIS_DB"
    - name: REDIS_PASSWORD
      value: "$REDIS_PASSWORD"
    - name: OUTPUT_FILE
      value: "/data/$OUTPUT_FILE"  # Save in writable location
    volumeMounts:
    - name: script-volume
      mountPath: /app
      readOnly: true
    - name: data-volume
      mountPath: /data  # Writable directory
  volumes:
  - name: script-volume
    configMap:
      name: $CONFIGMAP_NAME
  - name: data-volume
    emptyDir: {}  # Temporary writable volume
EOF

# Wait for the pod to be ready
echo "Waiting for pod to be ready..."
kubectl wait --for=condition=Ready pod/$POD_NAME --namespace $NAMESPACE --timeout=60s

# Install Redis Python package inside the Pod
echo "Installing Redis Python package..."
kubectl exec -n $NAMESPACE $POD_NAME -- pip install redis

# Execute the script inside the Pod
echo "Running Redis data extraction..."
kubectl exec -n $NAMESPACE $POD_NAME -- python /app/$SCRIPT_NAME

# Copy the output file from Pod to local
echo "Copying data from Pod..."
kubectl cp $NAMESPACE/$POD_NAME:/data/$OUTPUT_FILE ./$OUTPUT_FILE

# Cleanup: Delete the Pod after execution
echo "Cleaning up..."
kubectl delete pod $POD_NAME --namespace $NAMESPACE

echo "Done! Redis data is saved in $OUTPUT_FILE"
