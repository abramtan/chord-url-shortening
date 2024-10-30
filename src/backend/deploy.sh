#!/bin/bash

# Define variables
DOCKER_IMAGE_NAME="chord-url-shortening:v1.0"
DEPLOYMENT_NAME="chord-url-shortening"
REPLICAS=5

# Build Docker image
echo "Building Docker image..."
eval $(minikube docker-env)
docker build -t $DOCKER_IMAGE_NAME .

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

echo "Deployment scaled successfully."