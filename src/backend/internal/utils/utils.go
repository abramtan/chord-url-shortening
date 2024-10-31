package utils

import (
	"context"
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	pb "chord-url-shortening/chordurlshortening"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// NodeService implements the NodeServiceServer interface
type NodeService struct {
	pb.UnimplementedNodeServiceServer
}

func GetEnvInt(key string, fallback int) (envInt int) {
	if value, exists := os.LookupEnv(key); exists {
		intValue, err := strconv.Atoi(value)
		if err == nil {
			envInt = intValue
			return
		}
	}
	envInt = fallback
	return
}

func GetEnvFloat(key string, fallback float32) float32 {
	if value, exists := os.LookupEnv(key); exists {
		floatValue, err := strconv.ParseFloat(value, 32)
		if err == nil {
			return float32(floatValue)
		}
	}
	return fallback
}

func GetEnvString(key string, fallback string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return fallback
}

func GetOtherPodIP() string {

	// Set up a grpc connection to another pod via cluster IP (which pod is dependent on how k8s load balances, since using cluster IP, can be itself)
	conn, err := grpc.NewClient(
		fmt.Sprintf("%s:%d", CHORD_URL_SHORTENING_SERVICE_HOST, CHORD_URL_SHORTENING_SERVICE_PORT_GRPC),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)

	if err != nil {
		log.Printf("Did not connect: %v", err)
	}
	defer conn.Close()

	client := pb.NewNodeServiceClient(conn)

	// Perform gRPC call to get the IP of another pod
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	req := &pb.GetIpRequest{NodeId: POD_IP}
	res, err := client.GetNodeIp(ctx, req)
	if err != nil {
		log.Printf("Could not get other pod's IP: %v", err)
	}

	return res.String()
}

// GetNodeIp handles GetIpRequest and returns the corresponding GetIpResponse
func (s *NodeService) GetNodeIp(ctx context.Context, req *pb.GetIpRequest) (*pb.GetIpResponse, error) {
	var POD_IP string = utils.GetEnvString("POD_IP", "0.0.0.0")
	return &pb.GetIpResponse{IpAddress: POD_IP}, nil
}
