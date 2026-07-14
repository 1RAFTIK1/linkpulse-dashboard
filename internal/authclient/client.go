// Package authclient — gRPC-клиент к Auth service (ValidateToken).
// Реализует ws.TokenValidator. Намеренно продублирован с linkpulse-link:
// в мульти-репо каждый сервис владеет своими клиентами, а contracts
// остаётся чистым репозиторием схем.
package authclient

import (
	"context"
	"fmt"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	authv1 "github.com/1RAFTIK1/linkpulse-contracts/gen/go/auth/v1"
)

const callTimeout = 3 * time.Second

type Client struct {
	conn *grpc.ClientConn
	api  authv1.AuthServiceClient
}

func New(addr string) (*Client, error) {
	conn, err := grpc.NewClient(addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("auth grpc client: %w", err)
	}
	return &Client{conn: conn, api: authv1.NewAuthServiceClient(conn)}, nil
}

func (c *Client) Close() error { return c.conn.Close() }

func (c *Client) Validate(ctx context.Context, token string) (string, bool, error) {
	ctx, cancel := context.WithTimeout(ctx, callTimeout)
	defer cancel()

	resp, err := c.api.ValidateToken(ctx, &authv1.ValidateTokenRequest{Token: token})
	if err != nil {
		return "", false, fmt.Errorf("validate token rpc: %w", err)
	}
	if !resp.GetValid() {
		return "", false, nil
	}
	return resp.GetUserId(), true, nil
}
