package cmd

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/spf13/cobra"

	pb "github.com/alex-storchak/gophkeeper/gen/proto/gophkeeper/v1"
)

var errEmptyEntriesList = errors.New("no entries found")

type DataLister interface {
	List(ctx context.Context, req *pb.ListRequest) (*pb.ListResponse, error)
}

func NewListCmd(l *slog.Logger, dl DataLister) *cobra.Command {
	listCmd := &cobra.Command{
		Use:   "list",
		Short: "Show user's data list",
		RunE:  newRunList(l, dl),
	}
	listCmd.Flags().String("type", "", "filter by type (credentials/card/text/binary)")
	return listCmd
}

func newRunList(l *slog.Logger, dl DataLister) func(*cobra.Command, []string) error {
	return func(cmd *cobra.Command, _ []string) error {
		cmd.SilenceUsage = true

		dataType, err := cmd.Flags().GetString("type")
		if err != nil {
			l.Warn("could not read data type from flag")
			dataType = ""
		}

		entries, err := processList(cmd.Context(), dataType, dl)
		if errors.Is(err, errEmptyEntriesList) {
			fmt.Println("No entries found.")
			return nil
		} else if err != nil {
			return fmt.Errorf("list user entries: %w; data_type: %s", err, dataType)
		}

		printEntries(entries)
		return nil
	}
}

func processList(
	ctx context.Context,
	dataType string,
	dl DataLister,
) ([]*pb.DataEntrySummary, error) {
	req := pb.ListRequest_builder{
		DataType: dataType,
	}.Build()
	resp, err := dl.List(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("get list data: %w", err)
	}

	entries := resp.GetEntries()
	if len(entries) == 0 {
		return nil, errEmptyEntriesList
	}
	return entries, nil
}

func printEntries(entries []*pb.DataEntrySummary) {
	fmt.Printf("%-10s %-15s %-30s %-30s %s\n", "ID", "Type", "Title", "Created_at", "Metadata")
	fmt.Println("------------------------------------------------------------------------------------------------------------")
	for _, e := range entries {
		fmt.Printf(
			"%-10d %-15s %-30s %-30s %s\n",
			e.GetId(),
			e.GetDataType(),
			e.GetTitle(),
			e.GetCreatedAt().AsTime().Format("2006-01-02 15:04:05"),
			e.GetMetadata(),
		)
	}
}
