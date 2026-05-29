//go:build tools

// Package tools declara as dependências da stack para que go mod tidy
// não as remova antes de qualquer código de domínio ser escrito.
package tools

import (
	_ "buf.build/go/protovalidate"
	_ "github.com/golang-jwt/jwt/v5"
	_ "github.com/golang-migrate/migrate/v4"
	_ "github.com/google/uuid"
	_ "github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/auth"
	_ "github.com/spf13/viper"
	_ "github.com/stretchr/testify/assert"
	_ "go.uber.org/fx"
	_ "go.uber.org/zap"
	_ "golang.org/x/crypto/bcrypt"
	_ "google.golang.org/grpc"
	_ "google.golang.org/protobuf/proto"
	_ "modernc.org/sqlite"
)
