package cmd

import (
	"context"
	"errors"
	"fmt"

	"github.com/spf13/cobra"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	pb "github.com/alex-storchak/gophkeeper/gen/proto/gophkeeper/v1"
	"github.com/alex-storchak/gophkeeper/internal/client/input"
	"github.com/alex-storchak/gophkeeper/internal/client/models"
	"github.com/alex-storchak/gophkeeper/internal/client/service"
	"github.com/alex-storchak/gophkeeper/internal/client/validator"
)

var (
	ErrCollectDataFuncNotFound = errors.New("function for collecting specific data not found")
	ErrDataAlreadyExists       = errors.New("entry with the same title already exists, delete it first")
)

type DataAdder interface {
	Add(ctx context.Context, req *pb.AddRequest) (*pb.AddResponse, error)
}

func NewAddCmd(da DataAdder) *cobra.Command {
	return &cobra.Command{
		Use:   "add",
		Short: "Adds new data",
		Long:  "Adds new data entry to the secure storage.",
		RunE:  NewRunAdd(da),
	}
}

func NewRunAdd(da DataAdder) func(*cobra.Command, []string) error {
	return func(cmd *cobra.Command, _ []string) error {
		cmd.SilenceUsage = true

		var cf models.CommonFields
		var err error

		// Data type
		dataType, err := input.ReadLine("Enter data type (credentials/card/text/binary): ")
		if err != nil {
			return fmt.Errorf("reading data type from input: %w", err)
		}
		cf.DataType = models.DataType(dataType)

		// Title
		cf.Title, err = input.ReadLine("Enter entry title: ")
		if err != nil {
			return fmt.Errorf("reading title from input: %w", err)
		}

		if _, err := validator.IsValid(cf); err != nil {
			return err
		}

		// Metadata
		meta, err := input.ReadLine("Enter metadata (optional): ")
		if err != nil {
			return fmt.Errorf("reading metadata from input: %w", err)
		}

		// Specific data
		encryptedData, salt, err := collectSpecificData(cf.DataType)
		if err != nil {
			return fmt.Errorf("collecting specific data: %w", err)
		}

		// Process
		req := pb.AddRequest_builder{
			DataType:      string(cf.DataType),
			Title:         cf.Title,
			EncryptedData: encryptedData,
			Metadata:      meta,
			Salt:          salt,
		}.Build()
		resp, err := da.Add(cmd.Context(), req)
		if err != nil {
			return handleAddError(err)
		}

		fmt.Printf("Data added successfully (ID: %d).\n", resp.GetId())
		return nil
	}
}

func collectSpecificData(dataType models.DataType) ([]byte, []byte, error) {
	var encryptedData, salt []byte
	var err error
	var mapFunc = map[models.DataType]func() ([]byte, []byte, error){
		models.DataTypeCredentials: collectCredsData,
		models.DataTypeCard:        collectCardData,
		models.DataTypeText:        collectTextData,
		models.DataTypeBinary:      collectBinaryData,
	}

	if collectFn, ok := mapFunc[dataType]; ok {
		encryptedData, salt, err = collectFn()
	} else {
		return nil, nil, fmt.Errorf("%w, dataType: %s", ErrCollectDataFuncNotFound, dataType)
	}

	if err != nil {
		return nil, nil, fmt.Errorf("collecting specific data: %w", err)
	}

	return encryptedData, salt, nil
}

func collectCredsData() ([]byte, []byte, error) {
	var data models.CredsData
	var err error

	data.Login, err = input.ReadLine("Enter data login: ")
	if err != nil {
		return nil, nil, fmt.Errorf("reading data login from input: %w", err)
	}

	data.Password, err = input.ReadPassword("Enter data password: ")
	if err != nil {
		return nil, nil, fmt.Errorf("reading data password from input: %w", err)
	}

	data.MasterPass, err = input.ReadPassword("Enter master password to encrypt data: ")
	if err != nil {
		return nil, nil, fmt.Errorf("reading master password from input: %w", err)
	}

	if _, err := validator.IsValid(data); err != nil {
		return nil, nil, err
	}

	return service.PrepareCredsData(data)
}

func collectCardData() ([]byte, []byte, error) {
	var data models.CardData
	var err error

	data.Number, err = input.ReadLine("Enter card number: ")
	if err != nil {
		return nil, nil, fmt.Errorf("reading card number from input: %w", err)
	}

	data.Holder, err = input.ReadLine("Enter card holder name: ")
	if err != nil {
		return nil, nil, fmt.Errorf("reading card holder name from input: %w", err)
	}

	data.ExpMonth, err = input.ReadLine("Enter card expiry month: ")
	if err != nil {
		return nil, nil, fmt.Errorf("reading card expiry month from input: %w", err)
	}

	data.ExpYear, err = input.ReadLine("Enter card expiry year: ")
	if err != nil {
		return nil, nil, fmt.Errorf("reading card expiry year from input: %w", err)
	}

	data.CVV, err = input.ReadPassword("Enter card CVV: ")
	if err != nil {
		return nil, nil, fmt.Errorf("reading card CVV from input: %w", err)
	}

	data.MasterPass, err = input.ReadPassword("Enter master password to encrypt data: ")
	if err != nil {
		return nil, nil, fmt.Errorf("reading master password from input: %w", err)
	}

	if _, err := validator.IsValid(data); err != nil {
		return nil, nil, err
	}

	return service.PrepareCardData(data)
}

func collectTextData() ([]byte, []byte, error) {
	var data models.TextData
	var err error

	data.Source, err = input.ReadLine("Enter source file path or press `Enter` to type: ")
	if err != nil {
		return nil, nil, fmt.Errorf("reading Source from input: %w", err)
	}

	data.MasterPass, err = input.ReadPassword("Enter master password to encrypt data: ")
	if err != nil {
		return nil, nil, fmt.Errorf("reading master password from input: %w", err)
	}

	if _, err := validator.IsValid(data); err != nil {
		return nil, nil, err
	}

	if data.Source == "" { // direct text input
		data.Text, err = input.ReadLine("Enter text: ")
		if err != nil {
			return nil, nil, fmt.Errorf("reading text from input: %w", err)
		}
		return service.PrepareRawTextData(data)
	}

	return service.PrepareTextData(data)
}

func collectBinaryData() ([]byte, []byte, error) {
	var data models.BinaryData
	var err error

	data.Source, err = input.ReadLine("Enter source file path: ")
	if err != nil {
		return nil, nil, fmt.Errorf("reading Source from input: %w", err)
	}

	data.MasterPass, err = input.ReadPassword("Enter master password to encrypt data: ")
	if err != nil {
		return nil, nil, fmt.Errorf("reading master password from input: %w", err)
	}

	if _, err := validator.IsValid(data); err != nil {
		return nil, nil, err
	}

	return service.PrepareBinaryData(data)
}

func handleAddError(err error) error {
	st, ok := status.FromError(err)
	if !ok {
		return fmt.Errorf("parsing grpc status from error: %w", err)
	}

	if st.Code() == codes.AlreadyExists {
		for _, detail := range st.Details() {
			if conflict, ok := detail.(*pb.ConflictInfo); ok {
				fmt.Println("  Existing entry:")
				fmt.Printf("    ID: %d\n", conflict.GetId())
				fmt.Printf("    Type: %s\n", conflict.GetDataType())
				fmt.Printf("    Title: %s\n", conflict.GetTitle())
				fmt.Printf("    Metadata: %s\n", conflict.GetMetadata())
				fmt.Printf("    Created_at: %s\n", conflict.GetCreatedAt().AsTime().Format("2006-01-02 15:04:05"))
			}
		}
		return ErrDataAlreadyExists
	}

	return fmt.Errorf("adding data: %w", err)
}
