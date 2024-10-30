#!/bin/bash

# Define variables
DEPLOYMENT_NAME="chord-url-shortening"
REPLICAS=10

# Deploy to K8S cluster
kubectl apply -f deployment.yaml
kubectl apply -f service.yaml

# Scale down the Kubernetes deployment
echo "Scaling down the deployment..."
kubectl scale deployments/$DEPLOYMENT_NAME --replicas=0

# Wait for a moment to ensure pods are terminated
sleep 5

# Scale up the Kubernetes deployment
echo "Scaling up the deployment..."
kubectl scale deployments/$DEPLOYMENT_NAME --replicas=$REPLICAS

echo "Pods restarted."