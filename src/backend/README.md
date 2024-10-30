# chord-url-shortening (Mac)

## PART 1: Backend Setup
Tested on an Apple Macbook Pro M1 16GB running:
- macOS Sequoia Version 15.0.1
- Docker Desktop v4.34.3
- Docker v23.0.2
- Kubernetes v1.26.3
- miniKube v1.34.0
- kubectl v1.31.2
- protoc-gen-go v1.35.1
- protoc-gen-go-grpc 1.5.1
- libprotoc 28.3
- go version 1.23.2 darwin/arm64

## Installation
```
brew install go
brew install kubectl
brew install Docker
brew install minikube
```
Note: Docker 4.34.0 is incompatible, upgrade or downgrade accordingly.

## Starting minikube
```
open /Applications/Docker.app
minikube start
```

## Build Docker Image (pushing directly to in-cluster Docker daemon)
Note: This allows usage of local Docker images, otherwise you'll need to push the image to a registry.
```
eval $(minikube docker-env)
docker ps   # To check containers running inside minikube
cd src/backend
docker build -t chord-url-shortening:v1.0 .
```

## Deploy to K8S Cluster
```
kubectl apply -f deployment.yaml
kubectl apply -f service.yaml
```

## Access Cluster via minikube
```
minikube service chord-url-shortening
```

## Useful Commands
### kubectl
1. List pods
```
kubectl get pods
```
2. Scale Deployments
```
kubectl scale deployments/chord-url-shortening --replicas=5
```
3. View deployments
```
kubectl get deployments
```
4. View cluster events
```
kubectl get events
```
5. View kubectl configuration
```
kubectl config view
```
6. View logs of a particular pod
```
kubectl logs [pod-name]
```
7. Expose an external IP for a deployment
```
kubectl expose deployment [deployment-name] --type=LoadBalancer --port=3000
```
8. View services
```
kubectl get services
```
9. Troubleshooting
```
kubectl describe pods
kubectl exec [pod-name] [cli-command] # Execute command on the container
kubectl exec "$POD_NAME" -- env # Prints the pod's environment variables
kubectl exec -ti [pod-name] -- bash # Starts a bash session in the pod's container
```
10. View ReplicaSet
```
kubectl get rs
```
11. Cleanup
```
kubectl delete service [deployment-name]
kubectl delete deployment [deployment-name]
```

### minikube
1. Launch minikube dashboard
```
minikube dashboard
```
2. Access a service's external IP via minikube
```
minikube service [deployment-name]
```
3. Stop the minikube cluster
```
minikube stop
```

---

## PART 2: Using gRPC
### Install protoc-gen-go and protoc-gen-go-grpc
```
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
export PATH="$PATH:$(go env GOPATH)/bin"
protoc-gen-go --version
protoc-gen-go-grpc --version
```

### Generate / Regenerate gRPC code
```
cd src/backend
protoc --go_out=. --go_opt=paths=source_relative \
    --go-grpc_out=. --go-grpc_opt=paths=source_relative \
    chordurlshortening/chordurlshortening.proto
```

## PART 3: Suggested Development Workflow
1. Install necessary software
2. Start minikube
```
open /Applications/Docker.app
minikube start
```
3. Build Docker image and deploy to K8S cluster (you can change the 'REPLICAS' var in deploy.sh to set number of pods)
```
chmod +x deploy.sh
./deploy.sh
```
4. Modify .go and / or .proto files
5. If you modified .proto files, generate / regenerate gRPC go files
```
cd src/backend
protoc --go_out=. --go_opt=paths=source_relative \
    --go-grpc_out=. --go-grpc_opt=paths=source_relative \
    chordurlshortening/chordurlshortening.proto
```
6. Re-build and re-deploy the Docker image onto K8S cluster
```
./deploy.sh
```
7. Test using curl
8. Test using Postman (remember to disable 'Connection': 'keep-alive' under the headers tab if you want to be able to test load-balancing, otherwise you'll be routed to the same pod)
9. Troubleshoot, debug, then repeat from step 4

## PART 4: Simulating Chord Node Failures
You can simulate a Chord node failure by simply adjusting the number of replicas in the K8S cluster (E.g. Change it from 5 to 4). Here's the command to do so:
```
kubectl scale deployments/chord-url-shortening --replicas=4
```

---

## References
1. https://kubernetes.io/docs/setup/
2. https://minikube.sigs.k8s.io/docs/start/
3. https://kubernetes.io/docs/tutorials/kubernetes-basics/create-cluster/cluster-intro/
4. https://minikube.sigs.k8s.io/docs/handbook/pushing/
5. https://www.tutorialworks.com/kubernetes-pod-ip/ 
6. https://grpc.io/docs/languages/go/quickstart/
7. https://github.com/grpc/grpc-go 
8. https://grpc.io/docs/languages/go/basics/ 

## Notes
- Currently, grpc calls initiate a new connection every time a call is made. How to reduce connections and use long-lived connections to improve performance?
- Also, grpc calls only use simple client-server calls, using more advanced features like bidirectional streaming might help make things more efficient.
- When a pod makes a call to the cluster IP, there is a chance that the request is routed back to the same pod. So a simple way to solve this is to check if IP of the 'other node' is the same, if same, just make the request again until the IP is different, denoting another pod is contacted, not itself. Probably should look into different load balancing algorithms or how to block re-routing a request back to the same pod.