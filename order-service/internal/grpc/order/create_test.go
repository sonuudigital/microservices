package order_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	cartv1 "github.com/sonuudigital/microservices/gen/cart/v1"
	orderv1 "github.com/sonuudigital/microservices/gen/order/v1"
	paymentv1 "github.com/sonuudigital/microservices/gen/payment/v1"
	"github.com/sonuudigital/microservices/order-service/internal/grpc/clients"
	"github.com/sonuudigital/microservices/order-service/internal/grpc/order"
	"github.com/sonuudigital/microservices/order-service/internal/repository"
	"github.com/sonuudigital/microservices/shared/logs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

const (
	testUserID    = "a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11"
	testOrderID   = "b0eebc99-9c0b-4ef8-bb6d-6bb9bd380a12"
	testCartID    = "c0eebc99-9c0b-4ef8-bb6d-6bb9bd380a13"
	testProductID = "d0eebc99-9c0b-4ef8-bb6d-6bb9bd380a14"
	testStatusID  = "e0eebc99-9c0b-4ef8-bb6d-6bb9bd380a15"
	testPaymentID = "f0eebc99-9c0b-4ef8-bb6d-6bb9bd380a16"
	testAmount    = "100.50"

	repositoryCreateOrderParamsType       = "repository.CreateOrderParams"
	repositoryUpdateOrderStatusParamsType = "repository.UpdateOrderStatusParams"
)

type MockQuerier struct {
	mock.Mock
}

func (m *MockQuerier) CreateOrder(ctx context.Context, arg repository.CreateOrderParams) (repository.Order, error) {
	args := m.Called(ctx, arg)
	return args.Get(0).(repository.Order), args.Error(1)
}

func (m *MockQuerier) GetOrderStatusByName(ctx context.Context, name string) (repository.GetOrderStatusByNameRow, error) {
	args := m.Called(ctx, name)
	return args.Get(0).(repository.GetOrderStatusByNameRow), args.Error(1)
}

func (m *MockQuerier) UpdateOrderStatus(ctx context.Context, arg repository.UpdateOrderStatusParams) (repository.Order, error) {
	args := m.Called(ctx, arg)
	return args.Get(0).(repository.Order), args.Error(1)
}

type MockCartClient struct {
	mock.Mock
}

func (m *MockCartClient) GetCart(ctx context.Context, in *cartv1.GetCartRequest, opts ...grpc.CallOption) (*cartv1.GetCartResponse, error) {
	args := m.Called(ctx, in)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*cartv1.GetCartResponse), args.Error(1)
}

func (m *MockCartClient) AddProductToCart(ctx context.Context, in *cartv1.AddProductToCartRequest, opts ...grpc.CallOption) (*cartv1.AddProductToCartResponse, error) {
	args := m.Called(ctx, in, opts)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*cartv1.AddProductToCartResponse), args.Error(1)
}

func (m *MockCartClient) RemoveProductFromCart(ctx context.Context, in *cartv1.RemoveProductFromCartRequest, opts ...grpc.CallOption) (*emptypb.Empty, error) {
	args := m.Called(ctx, in, opts)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*emptypb.Empty), args.Error(1)
}

func (m *MockCartClient) ClearCart(ctx context.Context, in *cartv1.ClearCartRequest, opts ...grpc.CallOption) (*emptypb.Empty, error) {
	args := m.Called(ctx, in, opts)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*emptypb.Empty), args.Error(1)
}

func (m *MockCartClient) DeleteCart(ctx context.Context, in *cartv1.DeleteCartRequest, opts ...grpc.CallOption) (*emptypb.Empty, error) {
	args := m.Called(ctx, in, opts)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*emptypb.Empty), args.Error(1)
}

type MockPaymentClient struct {
	mock.Mock
}

func (m *MockPaymentClient) GetPayment(ctx context.Context, in *paymentv1.GetPaymentRequest, opts ...grpc.CallOption) (*paymentv1.Payment, error) {
	args := m.Called(ctx, in, opts)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*paymentv1.Payment), args.Error(1)
}

func (m *MockPaymentClient) ProcessPayment(ctx context.Context, in *paymentv1.ProcessPaymentRequest, opts ...grpc.CallOption) (*paymentv1.Payment, error) {
	args := m.Called(ctx, in)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*paymentv1.Payment), args.Error(1)
}

type MockRabbitMQ struct {
	mock.Mock
}

func (m *MockRabbitMQ) Publish(ctx context.Context, exchange string, body []byte) error {
	args := m.Called(ctx, exchange, body)
	return args.Error(0)
}

func TestCreateOrder(t *testing.T) {
	req := &orderv1.CreateOrderRequest{
		UserId: testUserID,
	}

	t.Run("Success", func(t *testing.T) {
		mockQuerier := new(MockQuerier)
		mockCartClient := new(MockCartClient)
		mockPaymentClient := new(MockPaymentClient)
		mockRabbitMQ := new(MockRabbitMQ)

		var userUUID pgtype.UUID
		_ = userUUID.Scan(testUserID)
		var orderUUID pgtype.UUID
		_ = orderUUID.Scan(testOrderID)
		var statusUUID pgtype.UUID
		_ = statusUUID.Scan(testStatusID)

		mockCartClient.On("GetCart", mock.Anything, &cartv1.GetCartRequest{
			UserId: testUserID,
		}).Return(&cartv1.GetCartResponse{
			Id:         testCartID,
			UserId:     testUserID,
			TotalPrice: 100.50,
			Products: []*cartv1.CartProduct{
				{ProductId: testProductID, Quantity: 2},
			},
		}, nil).Once()

		var totalAmount pgtype.Numeric
		_ = totalAmount.Scan(testAmount)

		createdOrder := repository.Order{
			ID:          orderUUID,
			UserID:      userUUID,
			TotalAmount: totalAmount,
			Status:      statusUUID,
			CreatedAt:   pgtype.Timestamptz{Time: time.Now(), Valid: true},
		}

		mockQuerier.On("CreateOrder", mock.Anything, mock.AnythingOfType(repositoryCreateOrderParamsType)).
			Return(createdOrder, nil).Once()

		mockQuerier.On("GetOrderStatusByName", mock.Anything, "CREATED").
			Return(repository.GetOrderStatusByNameRow{
				ID:   statusUUID,
				Name: "CREATED",
			}, nil).Once()

		mockPaymentClient.On("ProcessPayment", mock.Anything, mock.MatchedBy(func(req *paymentv1.ProcessPaymentRequest) bool {
			return req.OrderId == testOrderID && req.UserId == testUserID && req.Amount == 100.50
		})).Return(&paymentv1.Payment{
			Id:      testPaymentID,
			OrderId: testOrderID,
			UserId:  testUserID,
			Amount:  100.50,
			Status:  "COMPLETED",
		}, nil).Once()

		mockRabbitMQ.On("Publish", mock.Anything, "order_created_exchange", mock.Anything).Return(nil).Once()

		server := order.New(
			logs.NewSlogLogger(),
			mockQuerier,
			&clients.Clients{
				CartServiceClient:    mockCartClient,
				PaymentServiceClient: mockPaymentClient,
			},
			mockRabbitMQ,
		)

		res, err := server.CreateOrder(context.Background(), req)

		assert.NoError(t, err)
		assert.NotNil(t, res)
		assert.Equal(t, testOrderID, res.Id)
		assert.Equal(t, testUserID, res.UserId)
		assert.Equal(t, 100.50, res.TotalAmount)
		assert.Equal(t, "CREATED", res.Status)
		mockQuerier.AssertExpectations(t)
		mockCartClient.AssertExpectations(t)
		mockPaymentClient.AssertExpectations(t)
		mockRabbitMQ.AssertExpectations(t)
	})

	t.Run("Empty User ID", func(t *testing.T) {
		mockQuerier := new(MockQuerier)
		server := order.New(logs.NewSlogLogger(), mockQuerier, nil, nil)

		emptyReq := &orderv1.CreateOrderRequest{UserId: ""}

		res, err := server.CreateOrder(context.Background(), emptyReq)

		assert.Error(t, err)
		assert.Nil(t, res)
		st, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, st.Code())
		assert.Contains(t, st.Message(), "user ID is required")
		mockQuerier.AssertNotCalled(t, "CreateOrder", mock.Anything, mock.Anything)
	})

	t.Run("Cart Not Found", func(t *testing.T) {
		mockQuerier := new(MockQuerier)
		mockCartClient := new(MockCartClient)

		mockCartClient.On("GetCart", mock.Anything, &cartv1.GetCartRequest{
			UserId: testUserID,
		}).Return(nil, status.Error(codes.NotFound, "cart not found")).Once()

		server := order.New(
			logs.NewSlogLogger(),
			mockQuerier,
			&clients.Clients{
				CartServiceClient: mockCartClient,
			},
			nil,
		)

		res, err := server.CreateOrder(context.Background(), req)

		assert.Error(t, err)
		assert.Nil(t, res)
		st, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.FailedPrecondition, st.Code())
		assert.Contains(t, st.Message(), "cart not found")
		mockCartClient.AssertExpectations(t)
		mockQuerier.AssertNotCalled(t, "CreateOrder", mock.Anything, mock.Anything)
	})

	t.Run("Empty Cart", func(t *testing.T) {
		mockQuerier := new(MockQuerier)
		mockCartClient := new(MockCartClient)

		mockCartClient.On("GetCart", mock.Anything, &cartv1.GetCartRequest{
			UserId: testUserID,
		}).Return(&cartv1.GetCartResponse{
			Id:         testCartID,
			UserId:     testUserID,
			TotalPrice: 0,
			Products:   []*cartv1.CartProduct{},
		}, nil).Once()

		server := order.New(
			logs.NewSlogLogger(),
			mockQuerier,
			&clients.Clients{
				CartServiceClient: mockCartClient,
			},
			nil,
		)

		res, err := server.CreateOrder(context.Background(), req)

		assert.Error(t, err)
		assert.Nil(t, res)
		st, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.FailedPrecondition, st.Code())
		assert.Contains(t, st.Message(), "cart is empty")
		mockCartClient.AssertExpectations(t)
		mockQuerier.AssertNotCalled(t, "CreateOrder", mock.Anything, mock.Anything)
	})

	t.Run("Invalid Cart Total Price", func(t *testing.T) {
		mockQuerier := new(MockQuerier)
		mockCartClient := new(MockCartClient)

		mockCartClient.On("GetCart", mock.Anything, &cartv1.GetCartRequest{
			UserId: testUserID,
		}).Return(&cartv1.GetCartResponse{
			Id:         testCartID,
			UserId:     testUserID,
			TotalPrice: -10.00,
			Products: []*cartv1.CartProduct{
				{ProductId: testProductID, Quantity: 2},
			},
		}, nil).Once()

		server := order.New(
			logs.NewSlogLogger(),
			mockQuerier,
			&clients.Clients{
				CartServiceClient: mockCartClient,
			},
			nil,
		)

		res, err := server.CreateOrder(context.Background(), req)

		assert.Error(t, err)
		assert.Nil(t, res)
		st, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.FailedPrecondition, st.Code())
		assert.Contains(t, st.Message(), "invalid cart total price")
		mockCartClient.AssertExpectations(t)
		mockQuerier.AssertNotCalled(t, "CreateOrder", mock.Anything, mock.Anything)
	})

	t.Run("Create Order Error", func(t *testing.T) {
		mockQuerier := new(MockQuerier)
		mockCartClient := new(MockCartClient)

		mockCartClient.On("GetCart", mock.Anything, &cartv1.GetCartRequest{
			UserId: testUserID,
		}).Return(&cartv1.GetCartResponse{
			Id:         testCartID,
			UserId:     testUserID,
			TotalPrice: 100.50,
			Products: []*cartv1.CartProduct{
				{ProductId: testProductID, Quantity: 2},
			},
		}, nil).Once()

		mockQuerier.On("CreateOrder", mock.Anything, mock.AnythingOfType(repositoryCreateOrderParamsType)).
			Return(repository.Order{}, errors.New("database error")).Once()

		server := order.New(
			logs.NewSlogLogger(),
			mockQuerier,
			&clients.Clients{
				CartServiceClient: mockCartClient,
			},
			nil,
		)

		res, err := server.CreateOrder(context.Background(), req)

		assert.Error(t, err)
		assert.Nil(t, res)
		st, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.Internal, st.Code())
		assert.Contains(t, st.Message(), "failed to create order")
		mockQuerier.AssertExpectations(t)
		mockCartClient.AssertExpectations(t)
	})

	t.Run("Get Order Status Error", func(t *testing.T) {
		mockQuerier := new(MockQuerier)
		mockCartClient := new(MockCartClient)

		var userUUID pgtype.UUID
		_ = userUUID.Scan(testUserID)
		var orderUUID pgtype.UUID
		_ = orderUUID.Scan(testOrderID)
		var statusUUID pgtype.UUID
		_ = statusUUID.Scan(testStatusID)

		mockCartClient.On("GetCart", mock.Anything, &cartv1.GetCartRequest{
			UserId: testUserID,
		}).Return(&cartv1.GetCartResponse{
			Id:         testCartID,
			UserId:     testUserID,
			TotalPrice: 100.50,
			Products: []*cartv1.CartProduct{
				{ProductId: testProductID, Quantity: 2},
			},
		}, nil).Once()

		var totalAmount pgtype.Numeric
		_ = totalAmount.Scan(testAmount)

		mockQuerier.On("CreateOrder", mock.Anything, mock.AnythingOfType(repositoryCreateOrderParamsType)).
			Return(repository.Order{
				ID:          orderUUID,
				UserID:      userUUID,
				TotalAmount: totalAmount,
				Status:      statusUUID,
				CreatedAt:   pgtype.Timestamptz{Time: time.Now(), Valid: true},
			}, nil).Once()

		mockQuerier.On("GetOrderStatusByName", mock.Anything, "CREATED").
			Return(repository.GetOrderStatusByNameRow{}, errors.New("status not found")).Once()

		server := order.New(
			logs.NewSlogLogger(),
			mockQuerier,
			&clients.Clients{
				CartServiceClient: mockCartClient,
			},
			nil,
		)

		res, err := server.CreateOrder(context.Background(), req)

		assert.Error(t, err)
		assert.Nil(t, res)
		st, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.Internal, st.Code())
		assert.Contains(t, st.Message(), "failed to get CREATED status")
		mockQuerier.AssertExpectations(t)
		mockCartClient.AssertExpectations(t)
	})

	t.Run("Payment Failed", func(t *testing.T) {
		mockQuerier := new(MockQuerier)
		mockCartClient := new(MockCartClient)
		mockPaymentClient := new(MockPaymentClient)

		var userUUID pgtype.UUID
		_ = userUUID.Scan(testUserID)
		var orderUUID pgtype.UUID
		_ = orderUUID.Scan(testOrderID)
		var statusUUID pgtype.UUID
		_ = statusUUID.Scan(testStatusID)

		mockCartClient.On("GetCart", mock.Anything, &cartv1.GetCartRequest{
			UserId: testUserID,
		}).Return(&cartv1.GetCartResponse{
			Id:         testCartID,
			UserId:     testUserID,
			TotalPrice: 100.50,
			Products: []*cartv1.CartProduct{
				{ProductId: testProductID, Quantity: 2},
			},
		}, nil).Once()

		var totalAmount pgtype.Numeric
		_ = totalAmount.Scan(testAmount)

		mockQuerier.On("CreateOrder", mock.Anything, mock.AnythingOfType(repositoryCreateOrderParamsType)).
			Return(repository.Order{
				ID:          orderUUID,
				UserID:      userUUID,
				TotalAmount: totalAmount,
				Status:      statusUUID,
				CreatedAt:   pgtype.Timestamptz{Time: time.Now(), Valid: true},
			}, nil).Once()

		mockQuerier.On("GetOrderStatusByName", mock.Anything, "CREATED").
			Return(repository.GetOrderStatusByNameRow{
				ID:   statusUUID,
				Name: "CREATED",
			}, nil).Once()

		mockPaymentClient.On("ProcessPayment", mock.Anything, mock.Anything).
			Return(nil, status.Error(codes.FailedPrecondition, "insufficient funds")).Once()

		var cancelledStatusUUID pgtype.UUID
		_ = cancelledStatusUUID.Scan(testStatusID)

		mockQuerier.On("GetOrderStatusByName", mock.Anything, "CANCELLED").
			Return(repository.GetOrderStatusByNameRow{
				ID:   cancelledStatusUUID,
				Name: "CANCELLED",
			}, nil).Once()

		mockQuerier.On("UpdateOrderStatus", mock.Anything, mock.AnythingOfType(repositoryUpdateOrderStatusParamsType)).
			Return(repository.Order{}, nil).Once()

		server := order.New(
			logs.NewSlogLogger(),
			mockQuerier,
			&clients.Clients{
				CartServiceClient:    mockCartClient,
				PaymentServiceClient: mockPaymentClient,
			},
			nil,
		)

		res, err := server.CreateOrder(context.Background(), req)

		assert.Error(t, err)
		assert.Nil(t, res)
		st, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.FailedPrecondition, st.Code())
		assert.Contains(t, st.Message(), "payment failed")
		mockQuerier.AssertExpectations(t)
		mockCartClient.AssertExpectations(t)
		mockPaymentClient.AssertExpectations(t)
	})

	t.Run("Event Publish Failed", func(t *testing.T) {
		mockQuerier := new(MockQuerier)
		mockCartClient := new(MockCartClient)
		mockPaymentClient := new(MockPaymentClient)
		mockRabbitMQ := new(MockRabbitMQ)

		var userUUID pgtype.UUID
		_ = userUUID.Scan(testUserID)
		var orderUUID pgtype.UUID
		_ = orderUUID.Scan(testOrderID)
		var statusUUID pgtype.UUID
		_ = statusUUID.Scan(testStatusID)

		mockCartClient.On("GetCart", mock.Anything, &cartv1.GetCartRequest{
			UserId: testUserID,
		}).Return(&cartv1.GetCartResponse{
			Id:         testCartID,
			UserId:     testUserID,
			TotalPrice: 100.50,
			Products: []*cartv1.CartProduct{
				{ProductId: testProductID, Quantity: 2},
			},
		}, nil).Once()

		var totalAmount pgtype.Numeric
		_ = totalAmount.Scan(testAmount)

		mockQuerier.On("CreateOrder", mock.Anything, mock.AnythingOfType(repositoryCreateOrderParamsType)).
			Return(repository.Order{
				ID:          orderUUID,
				UserID:      userUUID,
				TotalAmount: totalAmount,
				Status:      statusUUID,
				CreatedAt:   pgtype.Timestamptz{Time: time.Now(), Valid: true},
			}, nil).Once()

		mockQuerier.On("GetOrderStatusByName", mock.Anything, "CREATED").
			Return(repository.GetOrderStatusByNameRow{
				ID:   statusUUID,
				Name: "CREATED",
			}, nil).Once()

		mockPaymentClient.On("ProcessPayment", mock.Anything, mock.Anything).
			Return(&paymentv1.Payment{
				Id:      testPaymentID,
				OrderId: testOrderID,
				UserId:  testUserID,
				Amount:  100.50,
				Status:  "COMPLETED",
			}, nil).Once()

		mockRabbitMQ.On("Publish", mock.Anything, "order_created_exchange", mock.Anything).
			Return(errors.New("rabbitmq connection failed")).Once()

		var cancelledStatusUUID pgtype.UUID
		_ = cancelledStatusUUID.Scan(testStatusID)

		mockQuerier.On("GetOrderStatusByName", mock.Anything, "CANCELLED").
			Return(repository.GetOrderStatusByNameRow{
				ID:   cancelledStatusUUID,
				Name: "CANCELLED",
			}, nil).Once()

		mockQuerier.On("UpdateOrderStatus", mock.Anything, mock.AnythingOfType(repositoryUpdateOrderStatusParamsType)).
			Return(repository.Order{}, nil).Once()

		server := order.New(
			logs.NewSlogLogger(),
			mockQuerier,
			&clients.Clients{
				CartServiceClient:    mockCartClient,
				PaymentServiceClient: mockPaymentClient,
			},
			mockRabbitMQ,
		)

		res, err := server.CreateOrder(context.Background(), req)

		assert.Error(t, err)
		assert.Nil(t, res)
		st, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.Internal, st.Code())
		assert.Contains(t, st.Message(), "failed to publish order created event")
		mockQuerier.AssertExpectations(t)
		mockCartClient.AssertExpectations(t)
		mockPaymentClient.AssertExpectations(t)
		mockRabbitMQ.AssertExpectations(t)
	})

	t.Run("Context Canceled", func(t *testing.T) {
		mockQuerier := new(MockQuerier)
		server := order.New(logs.NewSlogLogger(), mockQuerier, nil, nil)

		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		res, err := server.CreateOrder(ctx, req)

		assert.Error(t, err)
		assert.Nil(t, res)
		st, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.Canceled, st.Code())
		mockQuerier.AssertNotCalled(t, "CreateOrder", mock.Anything, mock.Anything)
	})

	t.Run("Cart Service Unavailable", func(t *testing.T) {
		mockQuerier := new(MockQuerier)
		mockCartClient := new(MockCartClient)

		mockCartClient.On("GetCart", mock.Anything, &cartv1.GetCartRequest{
			UserId: testUserID,
		}).Return(nil, status.Error(codes.Unavailable, "service unavailable")).Once()

		server := order.New(
			logs.NewSlogLogger(),
			mockQuerier,
			&clients.Clients{
				CartServiceClient: mockCartClient,
			},
			nil,
		)

		res, err := server.CreateOrder(context.Background(), req)

		assert.Error(t, err)
		assert.Nil(t, res)
		st, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.Unavailable, st.Code())
		assert.Contains(t, st.Message(), "cart service temporarily unavailable")
		mockCartClient.AssertExpectations(t)
		mockQuerier.AssertNotCalled(t, "CreateOrder", mock.Anything, mock.Anything)
	})

	t.Run("Payment Service Unavailable", func(t *testing.T) {
		mockQuerier := new(MockQuerier)
		mockCartClient := new(MockCartClient)
		mockPaymentClient := new(MockPaymentClient)

		var userUUID pgtype.UUID
		_ = userUUID.Scan(testUserID)
		var orderUUID pgtype.UUID
		_ = orderUUID.Scan(testOrderID)
		var statusUUID pgtype.UUID
		_ = statusUUID.Scan(testStatusID)

		mockCartClient.On("GetCart", mock.Anything, &cartv1.GetCartRequest{
			UserId: testUserID,
		}).Return(&cartv1.GetCartResponse{
			Id:         testCartID,
			UserId:     testUserID,
			TotalPrice: 100.50,
			Products: []*cartv1.CartProduct{
				{ProductId: testProductID, Quantity: 2},
			},
		}, nil).Once()

		var totalAmount pgtype.Numeric
		_ = totalAmount.Scan(testAmount)

		mockQuerier.On("CreateOrder", mock.Anything, mock.AnythingOfType(repositoryCreateOrderParamsType)).
			Return(repository.Order{
				ID:          orderUUID,
				UserID:      userUUID,
				TotalAmount: totalAmount,
				Status:      statusUUID,
				CreatedAt:   pgtype.Timestamptz{Time: time.Now(), Valid: true},
			}, nil).Once()

		mockQuerier.On("GetOrderStatusByName", mock.Anything, "CREATED").
			Return(repository.GetOrderStatusByNameRow{
				ID:   statusUUID,
				Name: "CREATED",
			}, nil).Once()

		mockPaymentClient.On("ProcessPayment", mock.Anything, mock.Anything).
			Return(nil, status.Error(codes.Unavailable, "service unavailable")).Once()

		var cancelledStatusUUID pgtype.UUID
		_ = cancelledStatusUUID.Scan(testStatusID)

		mockQuerier.On("GetOrderStatusByName", mock.Anything, "CANCELLED").
			Return(repository.GetOrderStatusByNameRow{
				ID:   cancelledStatusUUID,
				Name: "CANCELLED",
			}, nil).Once()

		mockQuerier.On("UpdateOrderStatus", mock.Anything, mock.AnythingOfType(repositoryUpdateOrderStatusParamsType)).
			Return(repository.Order{}, nil).Once()

		server := order.New(
			logs.NewSlogLogger(),
			mockQuerier,
			&clients.Clients{
				CartServiceClient:    mockCartClient,
				PaymentServiceClient: mockPaymentClient,
			},
			nil,
		)

		res, err := server.CreateOrder(context.Background(), req)

		assert.Error(t, err)
		assert.Nil(t, res)
		st, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.Unavailable, st.Code())
		assert.Contains(t, st.Message(), "payment service temporarily unavailable")
		mockQuerier.AssertExpectations(t)
		mockCartClient.AssertExpectations(t)
		mockPaymentClient.AssertExpectations(t)
	})
}
