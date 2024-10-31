// package main

// import (
// 	pb "chord-url-shortening/chordurlshortening"
// 	"chord-url-shortening/internal/utils"
// 	"context"
// )

// // NodeService implements the NodeServiceServer interface
// type NodeService struct {
// 	pb.UnimplementedNodeServiceServer
// }

// // GetNodeIp handles GetIpRequest and returns the corresponding GetIpResponse
// func (s *NodeService) GetNodeIp(ctx context.Context, req *pb.GetIpRequest) (*pb.GetIpResponse, error) {
// 	var POD_IP string = utils.GetEnvString("POD_IP", "0.0.0.0")
// 	return &pb.GetIpResponse{IpAddress: POD_IP}, nil
// }

// func (s *NodeService) FindSuccessor(ctx context.Context, req *pb.FindSuccessorRequest) (*pb.FindSuccessorResponse, error) {
// 	var POD_IP string = utils.GetEnvString("POD_IP", "0.0.0.0")
// 	n.


// 	return &pb.FindSuccessorResponse{ipAddress: POD_IP}, nil
// }
