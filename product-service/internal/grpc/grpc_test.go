package grpc_test

import (
	"github.com/jackc/pgx/v5/pgtype"
)

const (
	uuidTest           = "a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11"
	uuidTest2          = "c2ffbc99-9c0b-4ef8-bb6d-6bb9bd380b33"
	categoryUID        = "b1eebc99-9c0b-4ef8-bb6d-6bb9bd380b22"
	uuidCategoryTest   = "b1eebc99-9c0b-4ef8-bb6d-6bb9bd380b22"
	uuidMalformed      = "malformed-uuid"
	productCachePrefix = "product:"
)

func scanAndGetCategoryUUID() pgtype.UUID {
	var pgUUID pgtype.UUID
	_ = pgUUID.Scan(uuidCategoryTest)
	return pgUUID
}
