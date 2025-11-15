package order_test

import (
	"context"
	"errors"
	"testing"

	cartv1 "github.com/sonuudigital/microservices/gen/cart/v1"
	orderv1 "github.com/sonuudigital/microservices/gen/order/v1"
	paymentv1 "github.com/sonuudigital/microservices/gen/payment/v1"
	"github.com/sonuudigital/microservices/order-service/internal/grpc/clients"
	"github.com/sonuudigital/microservices/order-service/internal/grpc/order"
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
	testPaymentID = "f0eebc99-9c0b-4ef8-bb6d-6bb9bd380a16"

	notImplementedError = "not implemented"
)

type MockOrderRepository struct {
	mock.Mock
}

func (m *MockOrderRepository) CreateOrder(ctx context.Context, userID string, totalAmount float64, products []*cartv1.CartProduct) (*orderv1.Order, error) {
	args := m.Called(ctx, userID, totalAmount, products)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*orderv1.Order), args.Error(1)
}

func (m *MockOrderRepository) CancelOrder(ctx context.Context, orderID string) error {
	args := m.Called(ctx, orderID)
	return args.Error(0)
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
	panic(notImplementedError)
}
func (m *MockCartClient) RemoveProductFromCart(ctx context.Context, in *cartv1.RemoveProductFromCartRequest, opts ...grpc.CallOption) (*emptypb.Empty, error) {
	panic(notImplementedError)
}
func (m *MockCartClient) ClearCart(ctx context.Context, in *cartv1.ClearCartRequest, opts ...grpc.CallOption) (*emptypb.Empty, error) {
	panic(notImplementedError)
}
func (m *MockCartClient) DeleteCart(ctx context.Context, in *cartv1.DeleteCartRequest, opts ...grpc.CallOption) (*emptypb.Empty, error) {
	panic(notImplementedError)
}

type MockPaymentClient struct {
	mock.Mock
}

func (m *MockPaymentClient) GetPayment(ctx context.Context, in *paymentv1.GetPaymentRequest, opts ...grpc.CallOption) (*paymentv1.Payment, error) {
	panic(notImplementedError)
}

func (m *MockPaymentClient) ProcessPayment(ctx context.Context, in *paymentv1.ProcessPaymentRequest, opts ...grpc.CallOption) (*paymentv1.Payment, error) {
	args := m.Called(ctx, in)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*paymentv1.Payment), args.Error(1)
}

func TestCreateOrder(t *testing.T) {
	req := &orderv1.CreateOrderRequest{
		UserId: testUserID,
	}
	cartProducts := []*cartv1.CartProduct{
		{ProductId: testProductID, Quantity: 2},
	}
	cartResponse := &cartv1.GetCartResponse{
		Id:         testCartID,
		UserId:     testUserID,
		TotalPrice: 100.50,
		Products:   cartProducts,
	}

	t.Run("Success", func(t *testing.T) {
		mockRepo := new(MockOrderRepository)
		mockCartClient := new(MockCartClient)
		mockPaymentClient := new(MockPaymentClient)

		mockCartClient.On("GetCart", mock.Anything, &cartv1.GetCartRequest{UserId: testUserID}).Return(cartResponse, nil).Once()

		mockPaymentClient.On("ProcessPayment", mock.Anything, mock.MatchedBy(func(req *paymentv1.ProcessPaymentRequest) bool {
			return req.UserId == testUserID && req.Amount == 100.50
		})).Return(&paymentv1.Payment{Id: testPaymentID, Status: "COMPLETED"}, nil).Once()

		expectedOrder := &orderv1.Order{Id: testOrderID, UserId: testUserID, TotalAmount: 100.50, Status: "CREATED"}
		mockRepo.On("CreateOrder", mock.Anything, testUserID, 100.50, cartProducts).Return(expectedOrder, nil).Once()

		server := order.New(
			logs.NewSlogLogger(),
			mockRepo,
			&clients.Clients{CartServiceClient: mockCartClient, PaymentServiceClient: mockPaymentClient},
		)

		res, err := server.CreateOrder(context.Background(), req)

		assert.NoError(t, err)
		assert.NotNil(t, res)
		assert.Equal(t, expectedOrder, res)
		mockRepo.AssertExpectations(t)
		mockCartClient.AssertExpectations(t)
		mockPaymentClient.AssertExpectations(t)
	})

	t.Run("Payment Failed", func(t *testing.T) {
		mockRepo := new(MockOrderRepository)
		mockCartClient := new(MockCartClient)
		mockPaymentClient := new(MockPaymentClient)

		mockCartClient.On("GetCart", mock.Anything, &cartv1.GetCartRequest{UserId: testUserID}).Return(cartResponse, nil).Once()

		expectedOrder := &orderv1.Order{Id: testOrderID, UserId: testUserID, TotalAmount: 100.50, Status: "CREATED"}
		mockRepo.On("CreateOrder", mock.Anything, testUserID, 100.50, cartProducts).Return(expectedOrder, nil).Once()

		paymentErr := status.Error(codes.FailedPrecondition, "insufficient funds")
		mockPaymentClient.On("ProcessPayment", mock.Anything, mock.MatchedBy(func(req *paymentv1.ProcessPaymentRequest) bool {
			return req.UserId == testUserID && req.OrderId == testOrderID && req.Amount == 100.50
		})).Return(nil, paymentErr).Once()

		mockRepo.On("CancelOrder", mock.Anything, testOrderID).Return(nil).Once()

		server := order.New(
			logs.NewSlogLogger(),
			mockRepo,
			&clients.Clients{CartServiceClient: mockCartClient, PaymentServiceClient: mockPaymentClient},
		)

		res, err := server.CreateOrder(context.Background(), req)

		assert.Error(t, err)
		assert.Nil(t, res)
		st, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.FailedPrecondition, st.Code())
		mockCartClient.AssertExpectations(t)
		mockPaymentClient.AssertExpectations(t)
		mockRepo.AssertExpectations(t)
	})

	t.Run("Payment Failed and CancelOrder Fails", func(t *testing.T) {
		mockRepo := new(MockOrderRepository)
		mockCartClient := new(MockCartClient)
		mockPaymentClient := new(MockPaymentClient)

		mockCartClient.On("GetCart", mock.Anything, &cartv1.GetCartRequest{UserId: testUserID}).Return(cartResponse, nil).Once()

		expectedOrder := &orderv1.Order{Id: testOrderID, UserId: testUserID, TotalAmount: 100.50, Status: "CREATED"}
		mockRepo.On("CreateOrder", mock.Anything, testUserID, 100.50, cartProducts).Return(expectedOrder, nil).Once()

		paymentErr := status.Error(codes.FailedPrecondition, "insufficient funds")
		mockPaymentClient.On("ProcessPayment", mock.Anything, mock.MatchedBy(func(req *paymentv1.ProcessPaymentRequest) bool {
			return req.UserId == testUserID && req.OrderId == testOrderID && req.Amount == 100.50
		})).Return(nil, paymentErr).Once()

		cancelErr := errors.New("failed to cancel order")
		mockRepo.On("CancelOrder", mock.Anything, testOrderID).Return(cancelErr).Once()

		server := order.New(
			logs.NewSlogLogger(),
			mockRepo,
			&clients.Clients{CartServiceClient: mockCartClient, PaymentServiceClient: mockPaymentClient},
		)

		res, err := server.CreateOrder(context.Background(), req)

		assert.Error(t, err)
		assert.Nil(t, res)
		st, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.FailedPrecondition, st.Code())
		mockCartClient.AssertExpectations(t)
		mockPaymentClient.AssertExpectations(t)
		mockRepo.AssertExpectations(t)
	})

	t.Run("Repository Create Order Fails", func(t *testing.T) {
		mockRepo := new(MockOrderRepository)
		mockCartClient := new(MockCartClient)
		mockPaymentClient := new(MockPaymentClient)

		mockCartClient.On("GetCart", mock.Anything, &cartv1.GetCartRequest{UserId: testUserID}).Return(cartResponse, nil).Once()

		repoErr := errors.New("database transaction failed")
		mockRepo.On("CreateOrder", mock.Anything, testUserID, 100.50, cartProducts).Return(nil, repoErr).Once()

		server := order.New(
			logs.NewSlogLogger(),
			mockRepo,
			&clients.Clients{CartServiceClient: mockCartClient, PaymentServiceClient: mockPaymentClient},
		)

		res, err := server.CreateOrder(context.Background(), req)

		assert.Error(t, err)
		assert.Nil(t, res)
		st, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.Internal, st.Code())
		mockRepo.AssertExpectations(t)
		mockCartClient.AssertExpectations(t)
		mockPaymentClient.AssertNotCalled(t, "ProcessPayment", mock.Anything, mock.Anything)
	})

	t.Run("Empty User ID", func(t *testing.T) {
		mockRepo := new(MockOrderRepository)
		server := order.New(logs.NewSlogLogger(), mockRepo, nil)
		res, err := server.CreateOrder(context.Background(), &orderv1.CreateOrderRequest{UserId: ""})

		assert.Error(t, err)
		assert.Nil(t, res)
		st, _ := status.FromError(err)
		assert.Equal(t, codes.InvalidArgument, st.Code())
		mockRepo.AssertNotCalled(t, "CreateOrder", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
	})

	t.Run("Cart Not Found", func(t *testing.T) {
		mockRepo := new(MockOrderRepository)
		mockCartClient := new(MockCartClient)
		mockCartClient.On("GetCart", mock.Anything, mock.Anything).Return(nil, status.Error(codes.NotFound, "cart not found")).Once()

		server := order.New(
			logs.NewSlogLogger(),
			mockRepo,
			&clients.Clients{CartServiceClient: mockCartClient},
		)
		res, err := server.CreateOrder(context.Background(), req)

		assert.Error(t, err)
		assert.Nil(t, res)
		st, _ := status.FromError(err)
		assert.Equal(t, codes.FailedPrecondition, st.Code())
		mockCartClient.AssertExpectations(t)
		mockRepo.AssertNotCalled(t, "CreateOrder", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
	})
}
