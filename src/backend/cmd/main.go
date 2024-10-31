package main

import (
	"chord-url-shortening/internal/utils"
	"chord-url-shortening/internal/chord"
	"context"
	"time"

	"fmt"
	"log"
	"net"
	"net/http"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	pb "chord-url-shortening/chordurlshortening"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func main() {
	// Load environment variables from .env file
	err := godotenv.Load()
	if err != nil {
		log.Println("Error loading .env file")
	}

	var GRPC_PORT int = utils.GetEnvInt("GRPC_PORT", 50051)
	var HTTP_PORT int = utils.GetEnvInt("HTTP_PORT", 8080)
	var POD_IP string = utils.GetEnvString("POD_IP", "0.0.0.0")
	var CHORD_URL_SHORTENING_SERVICE_HOST string = utils.GetEnvString("CHORD_URL_SHORTENING_SERVICE_HOST", "0.0.0.0")
	var CHORD_URL_SHORTENING_SERVICE_PORT int = utils.GetEnvInt("CHORD_URL_SHORTENING_SERVICE_PORT", 8080)
	var CHORD_URL_SHORTENING_SERVICE_PORT_GRPC int = utils.GetEnvInt("CHORD_URL_SHORTENING_SERVICE_PORT_GRPC", 50051)

	

	fmt.Println("\n\n------------------------------------------------")
	fmt.Println("           Environment variables used           ")
	fmt.Println("------------------------------------------------")

	fmt.Printf("[GRPC_PORT]: %d\n", GRPC_PORT)
	fmt.Printf("[HTTP_PORT]: %d\n", HTTP_PORT)
	fmt.Printf("[POD_IP]: %s\n", POD_IP)
	fmt.Printf("[CHORD_URL_SHORTENING_SERVICE_HOST]: %s\n", CHORD_URL_SHORTENING_SERVICE_HOST)
	fmt.Printf("[CHORD_URL_SHORTENING_SERVICE_PORT]: %d\n", CHORD_URL_SHORTENING_SERVICE_PORT)
	fmt.Printf("[CHORD_URL_SHORTENING_SERVICE_PORT_GRPC]: %d\n", CHORD_URL_SHORTENING_SERVICE_PORT_GRPC)

	fmt.Println("\n\n------------------------------------------------")
	fmt.Println("                Starting node...                ")
	fmt.Println("------------------------------------------------")

	// Start the gRPC server
	go startGRPCServer(GRPC_PORT)

	// Initialize Gin router
	router := gin.Default()

	// Define health check endpoint
	router.GET("/api/v1/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "healthy"})
	})

	router.GET("/api/v1/pod_ip", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"POD_IP": POD_IP})
	})

	router.GET("/api/v1/other_pod_ip", func(c *gin.Context) {

		// Set up a grpc connection to another pod via cluster IP (which pod is dependent on how k8s load balances, since using cluster IP, can be itself)
		conn, err := grpc.NewClient(
			fmt.Sprintf("%s:%d", CHORD_URL_SHORTENING_SERVICE_HOST, CHORD_URL_SHORTENING_SERVICE_PORT_GRPC),
			grpc.WithTransportCredentials(insecure.NewCredentials()),
		)
		if err != nil {
			log.Printf("Did not connect: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to connect via gRPC"})
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
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get other pod's IP"})
		}

		var OTHER_POD_IP string = fmt.Sprintf("POD_IP: %s, OTHER_POD_IP: %s", POD_IP, res.GetIpAddress())
		log.Printf("OTHER_POD_IP: %s", OTHER_POD_IP)
		c.JSON(http.StatusOK, gin.H{"OTHER_POD_IP": OTHER_POD_IP})
	})

	router.GET("/api/v1/grpc_check", func(c *gin.Context) {
		// Get the IP address from query parameters
		ipAddress := c.Query("ip")

		// Set up a gRPC connection to another pod using the provided IP address
		conn, err := grpc.NewClient(
			fmt.Sprintf("%s:%d", ipAddress, CHORD_URL_SHORTENING_SERVICE_PORT_GRPC),
			grpc.WithTransportCredentials(insecure.NewCredentials()),
		)
		if err != nil {
			log.Printf("Did not connect: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to connect via gRPC"})
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
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get other pod's IP"})
		}

		var GRPC_CHECK string = fmt.Sprintf("POD_IP: %s, OTHER_POD_IP: %s", POD_IP, res.GetIpAddress())
		log.Printf("GRPC_CHECK: %s", GRPC_CHECK)
		c.JSON(http.StatusOK, gin.H{"GRPC_CHECK": GRPC_CHECK})
	})

	// Start the HTTP server
	if err := router.Run(fmt.Sprintf(":%d", HTTP_PORT)); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}

	node := chord.CreateNode(chord.IPAddress(POD_IP))

	node.JoinRingIfExistsElseCreateRing() // create ring if no ring, else join existing ring

	
}

// Function to start the gRPC server
func startGRPCServer(port int) {
	// Create a new gRPC server
	grpcServer := grpc.NewServer()

	// Register your gRPC service here
	pb.RegisterNodeServiceServer(grpcServer, &NodeService{})

	// Listen on the specified port
	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		log.Fatalf("Failed to listen on port %d: %v", port, err)
	}

	log.Printf("gRPC server is running on port %d", port)

	// Start the gRPC server
	if err := grpcServer.Serve(listener); err != nil {
		log.Fatalf("Failed to serve gRPC server: %v", err)
	}
}

// NodeService implements the NodeServiceServer interface
type NodeService struct {
	pb.UnimplementedNodeServiceServer
}

// GetNodeIp handles GetIpRequest and returns the corresponding GetIpResponse
func (s *NodeService) GetNodeIp(ctx context.Context, req *pb.GetIpRequest) (*pb.GetIpResponse, error) {
	var POD_IP string = utils.GetEnvString("POD_IP", "0.0.0.0")
	return &pb.GetIpResponse{IpAddress: POD_IP}, nil
}