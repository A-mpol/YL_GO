package main

import (
	"context"
	"log"
	"net"
	"os"
	"os/signal"
	"strconv"
	"sync"
	"syscall"
	test "yl-go/gen/go"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var (
	info = make(map[string]Order)
	mu   sync.RWMutex
)

type Order struct {
	item     string
	quantity int32
}

type Server struct {
	test.UnimplementedOrderServiceServer
}

func (*Server) ListOrders(ctx context.Context, in *test.ListOrdersRequest) (*test.ListOrdersResponse, error) {
	if len(info) == 0 {
		return &test.ListOrdersResponse{}, nil
	}

	var resp test.ListOrdersResponse
	for id, order := range info {
		resp.Orders = append(resp.Orders, &test.Order{Id: id, Item: order.item, Quantity: order.quantity})
	}

	return &resp, nil
}

func (*Server) DeleteOrder(ctx context.Context, in *test.DeleteOrderRequest) (*test.DeleteOrderResponse, error) {
	resp := test.DeleteOrderResponse{Success: false}

	mu.RLock()
	_, ok := info[in.Id]
	mu.Unlock()

	if ok != true {
		return &resp, status.Error(codes.NotFound, "заказа по такому id не существует")
	}

	mu.Lock()
	delete(info, in.Id)
	mu.Unlock()

	resp = test.DeleteOrderResponse{Success: true}

	return &resp, nil
}

func (*Server) UpdateOrder(ctx context.Context, in *test.UpdateOrderRequest) (*test.UpdateOrderResponse, error) {
	mu.RLock()
	_, ok := info[in.Id]
	mu.Unlock()

	if ok != true {
		return nil, status.Error(codes.NotFound, "заказа по такому id не существует")
	}

	var resp test.UpdateOrderResponse
	resp.Order.Id = in.Id
	resp.Order.Item = in.Item
	resp.Order.Quantity = in.Quantity

	mu.Lock()
	info[in.Id] = Order{in.Item, in.Quantity}
	mu.Unlock()

	return &resp, nil
}

func (*Server) GetOrder(ctx context.Context, in *test.GetOrderRequest) (*test.GetOrderResponse, error) {
	mu.RLock()
	order, ok := info[in.Id]
	mu.Unlock()

	if ok != true {
		return nil, status.Error(codes.NotFound, "заказа по такому id не существует")
	}

	var resp test.GetOrderResponse
	resp.Order.Id = in.Id
	resp.Order.Item = order.item
	resp.Order.Quantity = order.quantity

	return &resp, nil
}

func (*Server) CreateOrder(ctx context.Context, in *test.CreateOrderRequest) (*test.CreateOrderResponse, error) {
	order := Order{in.Item, in.Quantity}
	var resp test.CreateOrderResponse

	newOrderID := GetNewID()
	resp.Id = newOrderID

	mu.Lock()
	info[newOrderID] = order
	mu.Unlock()

	return &resp, nil
}

func GetNewID() string {
	mu.RLock()
	defer mu.Unlock()

	newID := strconv.Itoa(len(info) + 1)

	return newID
}

func main() {
	server := grpc.NewServer()
	test.RegisterOrderServiceServer(server, &Server{})

	listener, err := net.Listen("tcp", ":50051")
	if err != nil {
		log.Fatal(err)
	}

	go func() {
		if err := server.Serve(listener); err != nil {
			log.Fatal(err)
		}
	}()

	log.Println("server started")
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGTERM, syscall.SIGINT)
	<-ch
	log.Println("server stop")
	server.GracefulStop()
}
