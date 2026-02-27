package cmd

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"

	"github.com/spf13/cobra"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	pb "github.com/alex-storchak/gophkeeper/gen/proto/gophkeeper/v1"
	"github.com/alex-storchak/gophkeeper/internal/client/input"
	"github.com/alex-storchak/gophkeeper/internal/client/models"
	"github.com/alex-storchak/gophkeeper/internal/client/service"
	"github.com/alex-storchak/gophkeeper/internal/client/validator"
)

const (
	msgEmptyMasterPassword = "master password cannot be empty"
	msgEmptyIdAndTitle     = "one of the flags --id or --title required"
)

var (
	ErrEntryNotFound           = errors.New("entry not found")
	ErrProcessDataFuncNotFound = errors.New("function for print received data not found")
	ErrEmptySaveFilePath       = errors.New("save file path cannot be empty")
)

type getCmdOpts struct {
	id         int64
	title      string
	masterPass string
}

func (g getCmdOpts) Valid() validator.Problems {
	problems := make(validator.Problems)

	if g.masterPass == "" {
		problems["masterPass"] = msgEmptyMasterPassword
	}

	if g.id == 0 && g.title == "" {
		problems["id or title"] = msgEmptyIdAndTitle
	}

	return problems
}

type DataGetter interface {
	Get(ctx context.Context, req *pb.GetRequest) (*pb.GetResponse, error)
}

func NewGetCmd(l *slog.Logger, dg DataGetter) *cobra.Command {
	getCmd := &cobra.Command{
		Use:   "get",
		Short: "Get data from secure storage by ID or title.",
		RunE:  newRunGet(l, dg),
	}

	getCmd.Flags().Int64("id", 0, "entry ID to search")
	getCmd.Flags().String("title", "", "entry title to search")

	return getCmd
}

func newRunGet(l *slog.Logger, dg DataGetter) func(*cobra.Command, []string) error {
	return func(cmd *cobra.Command, _ []string) error {
		cmd.SilenceUsage = true

		var opts getCmdOpts
		var err error

		opts.id, opts.title = parseGetFlagValues(cmd, l)

		opts.masterPass, err = input.ReadPassword("Enter master password to decrypt the data: ")
		if err != nil {
			return fmt.Errorf("reading master password from input: %w", err)
		}

		if _, err := validator.IsValid(opts); err != nil {
			return err
		}

		// Process
		req := pb.GetRequest_builder{
			Id:    opts.id,
			Title: opts.title,
		}.Build()
		resp, err := dg.Get(cmd.Context(), req)
		if err != nil {
			st, ok := status.FromError(err)
			if ok && st.Code() == codes.NotFound {
				return fmt.Errorf("%w: entry_id: %d, entry_title: %s", ErrEntryNotFound, opts.id, opts.title)
			}
			return fmt.Errorf("get entry with id = %d, title = %s: %w", opts.id, opts.title, err)
		}

		if err := processData(resp, opts.masterPass); err != nil {
			return fmt.Errorf("process and print response data: %w", err)
		}

		return nil
	}
}

func parseGetFlagValues(cmd *cobra.Command, l *slog.Logger) (int64, string) {
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

func processData(resp *pb.GetResponse, masterPass string) error {
	var err error
	var mapFunc = map[models.DataType]func(*pb.GetResponse, string) error{
		models.DataTypeCredentials: processCredsData,
		models.DataTypeCard:        processCardData,
		models.DataTypeText:        processTextData,
		models.DataTypeBinary:      processBinaryData,
	}

	dataType := models.DataType(resp.GetDataType())

	if printFn, ok := mapFunc[dataType]; ok {
		err = printFn(resp, masterPass)
	} else {
		return fmt.Errorf("%w, dataType: %s", ErrProcessDataFuncNotFound, dataType)
	}

	if err != nil {
		return fmt.Errorf("process received data: %w", err)
	}

	return nil
}

func printCommonFields(resp *pb.GetResponse) {
	fmt.Printf("\n--- RESULT ---\n")
	fmt.Printf("Type: %s\n", resp.GetDataType())
	fmt.Printf("Title: %s\n", resp.GetTitle())
	fmt.Printf("Metadata: %s\n", resp.GetMetadata())
	fmt.Printf("Created at: %s\n", resp.GetCreatedAt().AsTime().Format("2006-01-02 15:04:05"))
}

func processBinaryData(resp *pb.GetResponse, masterPass string) error {
	decrypted, err := service.DecryptRawData(resp.GetEncryptedData(), resp.GetSalt(), masterPass)
	if err != nil {
		return fmt.Errorf("decrypting binary data: %w", err)
	}

	savePath, err := input.ReadLine("Enter file path to save data (required): ")
	if err != nil {
		return fmt.Errorf("reading save file path from input: %w", err)
	}
	if savePath == "" {
		return fmt.Errorf("%w for binary data", ErrEmptySaveFilePath)
	}

	printCommonFields(resp)
	if err := os.WriteFile(savePath, decrypted, 0600); err != nil {
		return fmt.Errorf("save text to file %s: %w", savePath, err)
	}
	fmt.Printf("\nBinary data saved to file: %s\n", savePath)
	return nil
}

func processTextData(resp *pb.GetResponse, masterPass string) error {
	decrypted, err := service.DecryptRawData(resp.GetEncryptedData(), resp.GetSalt(), masterPass)
	if err != nil {
		return fmt.Errorf("decrypting text data: %w", err)
	}

	savePath, err := input.ReadLine("Save to file (enter path or press `Enter` to show text in console): ")
	if err != nil {
		return fmt.Errorf("reading save file path from input: %w", err)
	}

	printCommonFields(resp)
	if savePath == "" {
		fmt.Println("\n--- TEXT DATA ---")
		fmt.Println(string(decrypted))
	} else {
		if err := os.WriteFile(savePath, decrypted, 0600); err != nil {
			return fmt.Errorf("save text to file %s: %w", savePath, err)
		}
		fmt.Printf("\nText data saved to file: %s\n", savePath)
	}
	return nil
}

func processCardData(resp *pb.GetResponse, masterPass string) error {
	cardData, err := service.DecryptCardData(resp.GetEncryptedData(), resp.GetSalt(), masterPass)
	if err != nil {
		return fmt.Errorf("decrypting card data: %w", err)
	}
	printCommonFields(resp)
	fmt.Println("\n--- CARD DATA ---")
	fmt.Printf("Number: %s\n", cardData.Number)
	fmt.Printf("Holder: %s\n", cardData.Holder)
	fmt.Printf("Expiry date: %s/%s\n", cardData.ExpMonth, cardData.ExpYear)
	fmt.Printf("CVV: %s\n", cardData.CVV)
	return nil
}

func processCredsData(resp *pb.GetResponse, masterPass string) error {
	credsData, err := service.DecryptCredsData(resp.GetEncryptedData(), resp.GetSalt(), masterPass)
	if err != nil {
		return fmt.Errorf("decrypting creds data: %w", err)
	}
	printCommonFields(resp)
	fmt.Println("\n--- CREDENTIALS DATA ---")
	fmt.Printf("Login: %s\n", credsData.Login)
	fmt.Printf("Password: %s\n", credsData.Password)
	return nil
}
