package utils

import (
	"fmt"
	"log"
	"os"
	"strconv"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	pb "chord-url-shortening/chordurlshortening"
)

var GRPC_PORT int = GetEnvInt("GRPC_PORT", 50051)
var HTTP_PORT int = GetEnvInt("HTTP_PORT", 8080)
var POD_IP string = GetEnvString("POD_IP", "0.0.0.0")
var CHORD_URL_SHORTENING_SERVICE_HOST string = GetEnvString("CHORD_URL_SHORTENING_SERVICE_HOST", "0.0.0.0")
var CHORD_URL_SHORTENING_SERVICE_PORT int = GetEnvInt("CHORD_URL_SHORTENING_SERVICE_PORT", 8080)
var CHORD_URL_SHORTENING_SERVICE_PORT_GRPC int = GetEnvInt("CHORD_URL_SHORTENING_SERVICE_PORT_GRPC", 50051)

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

func GetClientPod(ipAddress string) pb.NodeServiceClient {
	// Set up a grpc connection to another pod via cluster IP (which pod is dependent on how k8s load balances, since using cluster IP, can be itself)
	conn, err := grpc.NewClient(
		fmt.Sprintf("%s:%d", ipAddress, CHORD_URL_SHORTENING_SERVICE_PORT_GRPC),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		log.Printf("Did not connect: %v", err)
	}
	defer conn.Close()
	client := pb.NewNodeServiceClient(conn)
	return client
}
