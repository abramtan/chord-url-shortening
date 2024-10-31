package main

import (
	pb "chord-url-shortening/chordurlshortening"
	"chord-url-shortening/internal/chord"
	"chord-url-shortening/internal/utils"
	"context"
	"log"
)

// NodeService implements the NodeServiceServer interface
type NodeService struct {
	pb.UnimplementedNodeServiceServer
}

// GetNodeIp handles GetIpRequest and returns the corresponding GetIpResponse
func (s *NodeService) GetNodeIp(ctx context.Context, req *pb.GetIpRequest) (*pb.GetIpResponse, error) {
	var POD_IP string = utils.POD_IP
	return &pb.GetIpResponse{IpAddress: POD_IP}, nil
}

// TODO: no finger table implemented
func (s *NodeService) FindSuccessor(ctx context.Context, req *pb.FindSuccessorRequest) (*pb.FindSuccessorResponse, error) {
	// var POD_IP string = utils.GetEnvString("POD_IP", "0.0.0.0")

	if node.IdBetween(chord.IPAddress(req.Id)) { // Take the ID to be found and compare it
		return &pb.FindSuccessorResponse{Successor: string(node.Succ)}, nil
	} else {

		// returns IPAddress
		highestPredOfId := node.ClosestPrecedingNode(chord.IPAddress(req.Id))

		// Set up gRPC to highestPred
		client := utils.GetClientPod(string(highestPredOfId))

		// Perform gRPC call to get the IP of another pod
		// ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		// defer cancel()

		req := &pb.FindSuccessorRequest{Id: req.Id}
		res, err := client.FindSuccessor(ctx, req)
		if err != nil {
			log.Printf("Could not get find sucessor: %v", err)
		}

		// grpc call here to the other node to find findSuccessor?
		return &pb.FindSuccessorResponse{Successor: string(res.Successor)}, nil
	}
}

// func (s *NodeService) JoinRing(ctx context.Context, req *pb.JoinRingRequest) (*pb.JoinRingResponse, error) {

// }
