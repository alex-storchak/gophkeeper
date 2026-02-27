package service

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/alex-storchak/gophkeeper/internal/client/crypto"
	cmdModels "github.com/alex-storchak/gophkeeper/internal/client/models"
	"github.com/alex-storchak/gophkeeper/internal/models"
)

func PrepareCredsData(creds cmdModels.CredsData) ([]byte, []byte, error) {
	data := models.CredsData{
		Login:    creds.Login,
		Password: creds.Password,
	}

	plaintext, err := json.Marshal(data)
	if err != nil {
		return nil, nil, fmt.Errorf("marshaling login data: %w", err)
	}

	return encryptData(plaintext, creds.MasterPass)
}

func PrepareCardData(card cmdModels.CardData) ([]byte, []byte, error) {
	data := models.CardData{
		Number:   card.Number,
		Holder:   card.Holder,
		ExpMonth: card.ExpMonth,
		ExpYear:  card.ExpYear,
		CVV:      card.CVV,
	}

	plaintext, err := json.Marshal(data)
	if err != nil {
		return nil, nil, fmt.Errorf("marshaling card data: %w", err)
	}

	return encryptData(plaintext, card.MasterPass)
}

func PrepareTextData(data cmdModels.TextData) ([]byte, []byte, error) {
	plaintext, err := os.ReadFile(data.Source)
	if err != nil {
		return nil, nil, fmt.Errorf("reading text file %s: %w", data.Source, err)
	}
	return encryptData(plaintext, data.MasterPass)
}

func PrepareBinaryData(data cmdModels.BinaryData) ([]byte, []byte, error) {
	plaintext, err := os.ReadFile(data.Source)
	if err != nil {
		return nil, nil, fmt.Errorf("reading binary file %s: %w", data.Source, err)
	}
	return encryptData(plaintext, data.MasterPass)
}

func PrepareRawTextData(data cmdModels.TextData) ([]byte, []byte, error) {
	return encryptData([]byte(data.Text), data.MasterPass)
}

func DecryptCredsData(encrypted, salt []byte, masterPass string) (*models.CredsData, error) {
	plaintext, err := decryptData(encrypted, salt, masterPass)
	if err != nil {
		return nil, err
	}

	var data models.CredsData
	if err := json.Unmarshal(plaintext, &data); err != nil {
		return nil, fmt.Errorf("unmarshaling login data: %w", err)
	}
	return &data, nil
}

func DecryptCardData(encrypted, salt []byte, masterPass string) (*models.CardData, error) {
	plaintext, err := decryptData(encrypted, salt, masterPass)
	if err != nil {
		return nil, err
	}

	var data models.CardData
	if err := json.Unmarshal(plaintext, &data); err != nil {
		return nil, fmt.Errorf("unmarshaling card data: %w", err)
	}
	return &data, nil
}

func DecryptRawData(encrypted, salt []byte, masterPass string) ([]byte, error) {
	return decryptData(encrypted, salt, masterPass)
}

func encryptData(plaintext []byte, masterPass string) ([]byte, []byte, error) {
	salt, err := crypto.GenerateSalt()
	if err != nil {
		return nil, nil, fmt.Errorf("generating salt: %w", err)
	}

	key := crypto.DeriveKey(masterPass, salt)
	encrypted, err := crypto.Encrypt(plaintext, key)
	if err != nil {
		return nil, nil, fmt.Errorf("encrypting data: %w", err)
	}

	return encrypted, salt, nil
}

func decryptData(encrypted, salt []byte, masterPass string) ([]byte, error) {
	key := crypto.DeriveKey(masterPass, salt)
	plaintext, err := crypto.Decrypt(encrypted, key)
	if err != nil {
		return nil, fmt.Errorf("decrypting data: %w", err)
	}
	return plaintext, nil
}
