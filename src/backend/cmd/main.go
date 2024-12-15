package main

import (
	"chord-url-shortening/internal/api"
	"chord-url-shortening/internal/chord"
	"chord-url-shortening/internal/utils"
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

var node *chord.Node

func main() {
	// Load environment variables from .env file
	err := godotenv.Load()
	if err != nil {
		log.Println("Error loading .env file")
	}

	fmt.Println("\n\n------------------------------------------------")
	fmt.Println("           Environment variables used           ")
	fmt.Println("------------------------------------------------")

	fmt.Printf("[GRPC_PORT]: %d\n", utils.GRPC_PORT)
	fmt.Printf("[HTTP_PORT]: %d\n", utils.HTTP_PORT)
	fmt.Printf("[POD_IP]: %s\n", utils.POD_IP)
	fmt.Printf("[CHORD_URL_SHORTENING_SERVICE_HOST]: %s\n", utils.CHORD_URL_SHORTENING_SERVICE_HOST)
	fmt.Printf("[CHORD_URL_SHORTENING_SERVICE_PORT]: %d\n", utils.CHORD_URL_SHORTENING_SERVICE_PORT)
	fmt.Printf("[CHORD_URL_SHORTENING_SERVICE_PORT_GRPC]: %d\n", utils.CHORD_URL_SHORTENING_SERVICE_PORT_GRPC)

	fmt.Println("\n\n------------------------------------------------")
	fmt.Println("                Starting node...                ")
	fmt.Println("------------------------------------------------")

	// Start the gRPC server
	go startGRPCServer(utils.GRPC_PORT)

	// Initialize Gin router
	router := gin.Default()

	router.POST("/shorten", api.ShortenURL)
	router.GET("/long-url/:short_code", api.GetLongURL)

	// Define health check endpoint
	router.GET("/api/v1/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "healthy"})
	})

	router.GET("/api/v1/pod_ip", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"POD_IP": utils.POD_IP})
	})

	router.GET("/api/v1/other_pod_ip", func(c *gin.Context) {

		// Set up a grpc connection to another pod via cluster IP (which pod is dependent on how k8s load balances, since using cluster IP, can be itself)
		conn, err := grpc.NewClient(
			fmt.Sprintf("%s:%d", utils.CHORD_URL_SHORTENING_SERVICE_HOST, utils.CHORD_URL_SHORTENING_SERVICE_PORT_GRPC),
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

		req := &pb.GetIpRequest{NodeId: utils.POD_IP}
		res, err := client.GetNodeIp(ctx, req)
		if err != nil {
			log.Printf("Could not get other pod's IP: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get other pod's IP"})
		}

		var OTHER_POD_IP string = fmt.Sprintf("POD_IP: %s, OTHER_POD_IP: %s", utils.POD_IP, res.GetIpAddress())
		log.Printf("OTHER_POD_IP: %s", OTHER_POD_IP)
		c.JSON(http.StatusOK, gin.H{"OTHER_POD_IP": OTHER_POD_IP})
	})

	router.GET("/api/v1/grpc_check", func(c *gin.Context) {
		// Get the IP address from query parameters
		ipAddress := c.Query("ip")

		// Set up a gRPC connection to another pod using the provided IP address
		conn, err := grpc.NewClient(
			fmt.Sprintf("%s:%d", ipAddress, utils.CHORD_URL_SHORTENING_SERVICE_PORT_GRPC),
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

		req := &pb.GetIpRequest{NodeId: utils.POD_IP}
		res, err := client.GetNodeIp(ctx, req)
		if err != nil {
			log.Printf("Could not get other pod's IP: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get other pod's IP"})
		}

		var GRPC_CHECK string = fmt.Sprintf("POD_IP: %s, OTHER_POD_IP: %s", utils.POD_IP, res.GetIpAddress())
		log.Printf("GRPC_CHECK: %s", GRPC_CHECK)
		c.JSON(http.StatusOK, gin.H{"GRPC_CHECK": GRPC_CHECK})
	})

	// Start the HTTP server
	if err := router.Run(fmt.Sprintf(":%d", utils.HTTP_PORT)); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}

	node = chord.CreateNode(chord.IPAddress(utils.POD_IP))

	node.JoinRingIfExistsElseCreateRing() // create ring if no ring, else join existing ring

	log.Printf("%+v", node)
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
