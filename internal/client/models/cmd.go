package models

import (
	"fmt"

	"github.com/alex-storchak/gophkeeper/internal/client/validator"
)

type DataType string

const (
	DataTypeCredentials DataType = "credentials"
	DataTypeCard        DataType = "card"
	DataTypeText        DataType = "text"
	DataTypeBinary      DataType = "binary"
)

const (
	MsgUnknownDataType           = "unknown data type, allowed one of credentials/card/text/binary"
	MsgEmptyTitle                = "entry title cannot be empty"
	MsgEmptyDataMasterPassword   = "master password cannot be empty"
	MsgEmptyBinarySourceFilePath = "binary source file path cannot be empty"
)

type CommonFields struct {
	DataType DataType
	Title    string
}

func (c CommonFields) Valid() validator.Problems {
	problems := make(validator.Problems)

	var validDataTypes = map[DataType]struct{}{
		DataTypeCredentials: {},
		DataTypeCard:        {},
		DataTypeText:        {},
		DataTypeBinary:      {},
	}

	if _, ok := validDataTypes[c.DataType]; !ok {
		problems["DataType"] = fmt.Sprintf("%s, passed: %s", MsgUnknownDataType, c.DataType)
	}

	if c.Title == "" {
		problems["Title"] = MsgEmptyTitle
	}

	return problems
}

type CredsData struct {
	Login      string
	Password   string
	MasterPass string
}

func (c CredsData) Valid() validator.Problems {
	problems := make(validator.Problems)

	if c.MasterPass == "" {
		problems["MasterPass"] = MsgEmptyDataMasterPassword
	}

	return problems
}

type CardData struct {
	Number     string
	Holder     string
	ExpMonth   string
	ExpYear    string
	CVV        string
	MasterPass string
}

func (c CardData) Valid() validator.Problems {
	problems := make(validator.Problems)

	if c.MasterPass == "" {
		problems["MasterPass"] = MsgEmptyDataMasterPassword
	}

	return problems
}

type TextData struct {
	Text       string
	Source     string
	MasterPass string
}

func (t TextData) Valid() validator.Problems {
	problems := make(validator.Problems)

	if t.MasterPass == "" {
		problems["MasterPass"] = MsgEmptyDataMasterPassword
	}

	return problems
}

type BinaryData struct {
	Source     string
	MasterPass string
}

func (b BinaryData) Valid() validator.Problems {
	problems := make(validator.Problems)

	if b.Source == "" {
		problems["Source"] = MsgEmptyBinarySourceFilePath
	}

	if b.MasterPass == "" {
		problems["MasterPass"] = MsgEmptyDataMasterPassword
	}

	return problems
}
