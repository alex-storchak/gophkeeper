package cmd

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/spf13/cobra"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	pb "github.com/alex-storchak/gophkeeper/gen/proto/gophkeeper/v1"
	"github.com/alex-storchak/gophkeeper/internal/client/input"
)

var (
	ErrDeleteCmdFlagRequired = errors.New("one of the flags --id or --title required")
	ErrDeleteEntryNotFound   = errors.New("entry to delete not found")
)

type DataDeleter interface {
	Delete(ctx context.Context, req *pb.DeleteRequest) error
}

func NewDeleteCmd(l *slog.Logger, dd DataDeleter) *cobra.Command {
	deleteCmd := &cobra.Command{
		Use:   "delete",
		Short: "Delete entry from storage by ID or title",
		RunE:  newRunDelete(l, dd),
	}

	deleteCmd.Flags().Int64("id", 0, "entry ID to delete")
	deleteCmd.Flags().String("title", "", "entry title to delete")

	return deleteCmd
}

func newRunDelete(l *slog.Logger, dd DataDeleter) func(*cobra.Command, []string) error {
	return func(cmd *cobra.Command, _ []string) error {
		cmd.SilenceUsage = true

		id, title := parseDeleteFlagValues(cmd, l)
		if id == 0 && title == "" {
			return ErrDeleteCmdFlagRequired
		}

		confirm, err := input.Confirm("Are you sure to delete entry?")
		if err != nil {
			return fmt.Errorf("reading confirm input: %w", err)
		}
		if !confirm {
			fmt.Println("Delete operation canceled.")
			return nil
		}

		req := pb.DeleteRequest_builder{
			Id:    id,
			Title: title,
		}.Build()
		err = dd.Delete(cmd.Context(), req)
		if err != nil {
			st, ok := status.FromError(err)
			if ok && st.Code() == codes.NotFound {
				return fmt.Errorf("%w: entry_id: %d, entry_title: %s", ErrDeleteEntryNotFound, id, title)
			}
			return fmt.Errorf("delete entry with id = %d, title = %s: %w", id, title, err)
		}

		fmt.Println("Entry deleted successfully.")
		return nil
	}
}

func parseDeleteFlagValues(cmd *cobra.Command, l *slog.Logger) (int64, string) {
	id, err := cmd.Flags().GetInt64("id")
	if err != nil {
		l.Debug("failed to parse `id` flag")
		id = 0
	}
	title, err := cmd.Flags().GetString("title")
	if err != nil {
		l.Debug("failed to parse `title` flag")
		title = ""
	}
	return id, title
}
