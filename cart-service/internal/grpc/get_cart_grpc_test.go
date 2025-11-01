package grpc_test

import (
	"context"
	"testing"
	"time"

	"github.com/go-redis/redismock/v9"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	grpc_server "github.com/sonuudigital/microservices/cart-service/internal/grpc"
	"github.com/sonuudigital/microservices/cart-service/internal/repository"
	cartv1 "github.com/sonuudigital/microservices/gen/cart/v1"
	"github.com/sonuudigital/microservices/shared/logs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestGetCart(t *testing.T) {
	t.Setenv("CART_TTL_HOURS", "24")

	t.Run("Get existing cart successfully - Cache Miss", func(t *testing.T) {
		mockQuerier := new(MockQuerier)
		mockProductFetcher := new(MockProductFetcher)
		redisClient, redisMock := redismock.NewClientMock()
		server := grpc_server.NewGRPCServer(mockQuerier, mockProductFetcher, redisClient, logs.NewSlogLogger())

		req := &cartv1.GetCartRequest{UserId: uuidTest}

		var userUUID pgtype.UUID
		_ = userUUID.Scan(uuidTest)
		var cartUUID pgtype.UUID
		_ = cartUUID.Scan(cartUUIDTest)

		existingCart := repository.Cart{
			ID:        cartUUID,
			UserID:    userUUID,
			CreatedAt: pgtype.Timestamptz{Time: time.Now(), Valid: true},
		}

		redisMock.ExpectGet(cartCachePrefix + uuidTest).RedisNil()

		mockQuerier.On("GetCartByUserID", mock.Anything, userUUID).Return(existingCart, nil)
		mockQuerier.On("GetCartProductsByCartID", mock.Anything, cartUUID).Return([]repository.GetCartProductsByCartIDRow{}, nil)
		mockProductFetcher.On("GetProductsByIDs", mock.Anything, mock.Anything).Return(map[string]grpc_server.Product{}, nil)

		redisMock.MatchExpectationsInOrder(false)
		redisMock.ExpectSet(cartCachePrefix+uuidTest, mock.Anything, 15*time.Minute).SetVal("OK")

		resp, err := server.GetCart(context.Background(), req)

		assert.NoError(t, err)
		assert.NotNil(t, resp)
		assert.Equal(t, cartUUID.String(), resp.Id)
		assert.Equal(t, userUUID.String(), resp.UserId)
		assert.Empty(t, resp.Products)
		assert.Equal(t, 0.0, resp.TotalPrice)
		mockQuerier.AssertExpectations(t)
	})

	t.Run("Get existing cart successfully - Cache Hit", func(t *testing.T) {
		mockQuerier := new(MockQuerier)
		mockProductFetcher := new(MockProductFetcher)
		redisClient, redisMock := redismock.NewClientMock()
		server := grpc_server.NewGRPCServer(mockQuerier, mockProductFetcher, redisClient, logs.NewSlogLogger())

		req := &cartv1.GetCartRequest{UserId: uuidTest}

		var cartUUID pgtype.UUID
		_ = cartUUID.Scan(cartUUIDTest)

		cachedCart := `{"id":"` + cartUUIDTest + `","user_id":"` + uuidTest + `"}`

		redisMock.ExpectGet(cartCachePrefix + uuidTest).SetVal(cachedCart)

		resp, err := server.GetCart(context.Background(), req)

		assert.NoError(t, err)
		assert.NotNil(t, resp)
		assert.Equal(t, cartUUID.String(), resp.Id)
		assert.Equal(t, uuidTest, resp.UserId)
		assert.Empty(t, resp.Products)
		assert.Equal(t, 0.0, resp.TotalPrice)
		mockQuerier.AssertNotCalled(t, "GetCartByUserID", mock.Anything, mock.Anything)
	})

	t.Run("Create cart when it does not exist", func(t *testing.T) {
		mockQuerier := new(MockQuerier)
		mockProductFetcher := new(MockProductFetcher)
		redisClient, redisMock := redismock.NewClientMock()
		server := grpc_server.NewGRPCServer(mockQuerier, mockProductFetcher, redisClient, logs.NewSlogLogger())

		req := &cartv1.GetCartRequest{UserId: uuidTest}

		var userUUID pgtype.UUID
		_ = userUUID.Scan(uuidTest)
		var cartUUID pgtype.UUID
		_ = cartUUID.Scan(cartUUIDTest)

		newCart := repository.Cart{
			ID:        cartUUID,
			UserID:    userUUID,
			CreatedAt: pgtype.Timestamptz{Time: time.Now(), Valid: true},
		}

		redisMock.ExpectGet(cartCachePrefix + uuidTest).RedisNil()

		mockQuerier.On("GetCartByUserID", mock.Anything, userUUID).Return(repository.Cart{}, pgx.ErrNoRows)
		mockQuerier.On("CreateCart", mock.Anything, userUUID).Return(newCart, nil)
		mockQuerier.On("GetCartProductsByCartID", mock.Anything, cartUUID).Return([]repository.GetCartProductsByCartIDRow{}, nil)
		mockProductFetcher.On("GetProductsByIDs", mock.Anything, mock.Anything).Return(map[string]grpc_server.Product{}, nil)

		redisMock.MatchExpectationsInOrder(false)
		redisMock.ExpectSet(cartCachePrefix+uuidTest, mock.Anything, 15*time.Minute).SetVal("OK")

		resp, err := server.GetCart(context.Background(), req)

		assert.NoError(t, err)
		assert.NotNil(t, resp)
		assert.Equal(t, cartUUID.String(), resp.Id)
		assert.Equal(t, userUUID.String(), resp.UserId)
		assert.Empty(t, resp.Products)
		assert.Equal(t, 0.0, resp.TotalPrice)
		mockQuerier.AssertExpectations(t)
	})
}
