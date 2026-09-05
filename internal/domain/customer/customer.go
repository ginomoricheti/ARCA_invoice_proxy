package customer

import (
	"errors"
	"regexp"
)

var (
	ErrInvalidCUIT    = errors.New("invalid CUIT")
	ErrInvalidName    = errors.New("invalid name")
	ErrInvalidEmail   = errors.New("invalid email")
	ErrInvalidAddress = errors.New("invalid address")
	ErrInvalidIVA     = errors.New("invalid IVA condition")
	ErrInvalidCountry = errors.New("invalid country code")
)

var cuitRegex = regexp.MustCompile(`^\d{11}$`)

type IVACondition string

const (
	IVAConditionRI IVACondition = "RI"
	IVAConditionMT IVACondition = "MT"
	IVAConditionEX IVACondition = "EX"
	IVAConditionCF IVACondition = "CF"
	IVAConditionNC IVACondition = "NC"
)

func ParseIVACondition(s string) (IVACondition, error) {
	switch s {
	case "RI":
		return IVAConditionRI, nil
	case "MT":
		return IVAConditionMT, nil
	case "EX":
		return IVAConditionEX, nil
	case "CF":
		return IVAConditionCF, nil
	case "NC":
		return IVAConditionNC, nil
	default:
		return "", ErrInvalidIVA
	}
}

func (c IVACondition) String() string {
	return string(c)
}

func (c IVACondition) Valid() bool {
	switch c {
	case IVAConditionRI, IVAConditionMT, IVAConditionEX, IVAConditionCF, IVAConditionNC:
		return true
	default:
		return false
	}
}

type Customer struct {
	ID           string
	CUIT         string
	Name         string
	Email        string
	Address      string
	IVACondition IVACondition
	CountryCode  string
	CreatedAt    string
	UpdatedAt    string
}

func NewCustomer(cuit, name, email, address string, ivaCondition IVACondition, countryCode string) (*Customer, error) {
	if !isValidCUIT(cuit) {
		return nil, ErrInvalidCUIT
	}
	if name == "" {
		return nil, ErrInvalidName
	}
	if email != "" && !isValidEmail(email) {
		return nil, ErrInvalidEmail
	}
	if address == "" {
		return nil, ErrInvalidAddress
	}
	if !ivaCondition.Valid() {
		return nil, ErrInvalidIVA
	}
	if countryCode == "" || len(countryCode) != 2 {
		return nil, ErrInvalidCountry
	}

	return &Customer{
		CUIT:         cuit,
		Name:         name,
		Email:        email,
		Address:      address,
		IVACondition: ivaCondition,
		CountryCode:  countryCode,
	}, nil
}

func isValidCUIT(cuit string) bool {
	if !cuitRegex.MatchString(cuit) {
		return false
	}
	return validateCUITChecksum(cuit)
}

func validateCUITChecksum(cuit string) bool {
	weights := []int{5, 4, 3, 2, 7, 6, 5, 4, 3, 2}
	sum := 0
	for i, w := range weights {
		sum += int(cuit[i]-'0') * w
	}
	remainder := sum % 11
	checkDigit := 11 - remainder
	if checkDigit == 11 {
		checkDigit = 0
	} else if checkDigit == 10 {
		checkDigit = 9
	}
	return checkDigit == int(cuit[10]-'0')
}

func isValidEmail(email string) bool {
	emailRegex := regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
	return emailRegex.MatchString(email)
}
