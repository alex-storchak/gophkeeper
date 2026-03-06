package grpcclient

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	pb "github.com/alex-storchak/gophkeeper/gen/proto/gophkeeper/v1"
	"github.com/alex-storchak/gophkeeper/internal/client/config"
	"github.com/alex-storchak/gophkeeper/internal/client/session"
)

const maxMsgSize = 64 * 1024 * 1024 // 64 MB

type Client struct {
	conn   *grpc.ClientConn
	Auth   pb.AuthServiceClient
	Data   pb.DataServiceClient
	cfg    *config.Config
	logger *slog.Logger
	sess   *session.Session
}

func NewConn(cfg *config.Config) (*grpc.ClientConn, error) {
	creds, err := credentials.NewClientTLSFromFile(cfg.TLS.CACertFile, "")
	if err != nil {
		return nil, fmt.Errorf("create TLS credentials: %w", err)
	}

	conn, err := grpc.NewClient(
		cfg.ServerAddress,
		grpc.WithTransportCredentials(creds),
		grpc.WithDefaultCallOptions(
			grpc.MaxCallRecvMsgSize(maxMsgSize),
			grpc.MaxCallSendMsgSize(maxMsgSize),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("connecting to server: %w", err)
	}

	return conn, nil
}

func New(cfg *config.Config, logger *slog.Logger) (*Client, error) {
	sess := session.NewSession(&cfg.Session, "", "")
	if err := sess.Load(); err != nil {
		return nil, fmt.Errorf("loading session: %w", err)
	}

	conn, err := NewConn(cfg)
	if err != nil {
		return nil, err
	}

	return &Client{
		conn:   conn,
		Auth:   pb.NewAuthServiceClient(conn),
		Data:   pb.NewDataServiceClient(conn),
		cfg:    cfg,
		logger: logger,
		sess:   sess,
	}, nil
}

func (c *Client) Close() error {
	if err := c.conn.Close(); err != nil {
		return fmt.Errorf("closing gRPC connection: %w", err)
	}
	return nil
}

func (c *Client) Register(ctx context.Context, username, password string) (string, error) {
	var token string
	err := c.withRetry(func() error {
		req := pb.RegisterRequest_builder{
			Username: username,
			Password: password,
		}.Build()
		resp, err := c.Auth.Register(ctx, req)
		if err != nil {
			return err
		}
		token = resp.GetToken()
		return nil
	})
	return token, err
}

func (c *Client) Login(ctx context.Context, username, password string) (string, error) {
	var token string
	err := c.withRetry(func() error {
		req := pb.LoginRequest_builder{
			Username: username,
			Password: password,
		}.Build()
		resp, err := c.Auth.Login(ctx, req)
		if err != nil {
			return err
		}
		token = resp.GetToken()
		return nil
	})
	return token, err
}

func (c *Client) Add(ctx context.Context, req *pb.AddRequest) (*pb.AddResponse, error) {
	var resp *pb.AddResponse
	err := c.withRetry(func() error {
		authCtx, err := c.authContext(ctx)
		if err != nil {
			return err
		}
		var header metadata.MD
		r, err := c.Data.Add(authCtx, req, grpc.Header(&header))
		if err != nil {
			return err
		}
		c.handleRefreshedToken(header)
		resp = r
		return nil
	})
	return resp, err
}

func (c *Client) Get(ctx context.Context, req *pb.GetRequest) (*pb.GetResponse, error) {
	var resp *pb.GetResponse
	err := c.withRetry(func() error {
		authCtx, err := c.authContext(ctx)
		if err != nil {
			return err
		}
		var header metadata.MD
		r, err := c.Data.Get(authCtx, req, grpc.Header(&header))
		if err != nil {
			return err
		}
		c.handleRefreshedToken(header)
		resp = r
		return nil
	})
	return resp, err
}

func (c *Client) List(ctx context.Context, req *pb.ListRequest) (*pb.ListResponse, error) {
	var resp *pb.ListResponse
	err := c.withRetry(func() error {
		authCtx, err := c.authContext(ctx)
		if err != nil {
			return err
		}
		var header metadata.MD
		r, err := c.Data.List(authCtx, req, grpc.Header(&header))
		if err != nil {
			return err
		}
		c.handleRefreshedToken(header)
		resp = r
		return nil
	})
	return resp, err
}

func (c *Client) Delete(ctx context.Context, req *pb.DeleteRequest) error {
	return c.withRetry(func() error {
		authCtx, err := c.authContext(ctx)
		if err != nil {
			return err
		}
		var header metadata.MD
		_, err = c.Data.Delete(authCtx, req, grpc.Header(&header))
		if err != nil {
			return err
		}
		c.handleRefreshedToken(header)
		return nil
	})
}

func (c *Client) authContext(ctx context.Context) (context.Context, error) {
	md := metadata.Pairs("authorization", "Bearer "+c.sess.Token)
	return metadata.NewOutgoingContext(ctx, md), nil
}

func (c *Client) handleRefreshedToken(header metadata.MD) {
	tokens := header.Get("x-refreshed-token")
	if len(tokens) > 0 && tokens[0] != "" {
		if err := c.sess.UpdateToken(tokens[0]); err != nil {
			c.logger.Error("failed to update refreshed token in session", slog.Any("err", err))
		}
	}
}

func (c *Client) withRetry(fn func() error) error {
	var lastErr error
	for i := 0; i <= c.cfg.MaxRetries; i++ {
		if err := fn(); err != nil {
			lastErr = err

			// No sense retry Unauthenticated or NotFound requests
			if st, ok := status.FromError(err); ok && (st.Code() == codes.Unauthenticated || st.Code() == codes.NotFound) {
				c.logger.Debug("authentication failed, stopping retries")
				return fmt.Errorf("authentication failed: %w", err)
			}

			if i < c.cfg.MaxRetries {
				c.logger.Warn("request failed, retrying",
					slog.Int("attempt", i+1),
					slog.Int("max_retries", c.cfg.MaxRetries),
					slog.Any("err", err),
				)
				time.Sleep(time.Duration(i+1) * 500 * time.Millisecond)
			}
			continue
		}
		return nil
	}
	return fmt.Errorf("all %d retries exhausted: %w", c.cfg.MaxRetries, lastErr)
}
