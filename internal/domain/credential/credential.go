package credential

import (
	"errors"
	"time"
)

var (
	ErrInvalidCredential = errors.New("invalid ARCA credential")
	ErrMissingCert       = errors.New("certificate is required")
	ErrMissingKey        = errors.New("private key is required")
	ErrExpiredCredential = errors.New("credential has expired")
)

type Environment string

const (
	EnvironmentHomologation Environment = "homologation"
	EnvironmentProduction   Environment = "production"
)

func ParseEnvironment(s string) (Environment, error) {
	switch s {
	case "homologation":
		return EnvironmentHomologation, nil
	case "production":
		return EnvironmentProduction, nil
	default:
		return "", errors.New("invalid environment: " + s)
	}
}

func (e Environment) String() string {
	return string(e)
}

func (e Environment) Valid() bool {
	return e == EnvironmentHomologation || e == EnvironmentProduction
}

type ARCACredential struct {
	ID          string
	CustomerID  string
	Environment Environment
	CUIT        string
	CertPEM     []byte
	KeyPEM      []byte
	ExpiresAt   time.Time
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

func NewARCACredential(
	customerID string,
	environment Environment,
	cuit string,
	certPEM, keyPEM []byte,
	expiresAt time.Time,
) (*ARCACredential, error) {
	if customerID == "" {
		return nil, errors.Join(ErrInvalidCredential, errors.New("customer ID is required"))
	}
	if !environment.Valid() {
		return nil, errors.Join(ErrInvalidCredential, errors.New("invalid environment"))
	}
	if cuit == "" {
		return nil, errors.Join(ErrInvalidCredential, errors.New("CUIT is required"))
	}
	if len(certPEM) == 0 {
		return nil, errors.Join(ErrInvalidCredential, ErrMissingCert)
	}
	if len(keyPEM) == 0 {
		return nil, errors.Join(ErrInvalidCredential, ErrMissingKey)
	}
	if expiresAt.IsZero() {
		return nil, errors.Join(ErrInvalidCredential, errors.New("expiration date is required"))
	}
	if time.Now().UTC().After(expiresAt) {
		return nil, errors.Join(ErrInvalidCredential, ErrExpiredCredential)
	}

	now := time.Now().UTC()
	return &ARCACredential{
		ID:          generateID(),
		CustomerID:  customerID,
		Environment: environment,
		CUIT:        cuit,
		CertPEM:     certPEM,
		KeyPEM:      keyPEM,
		ExpiresAt:   expiresAt,
		CreatedAt:   now,
		UpdatedAt:   now,
	}, nil
}

func (c *ARCACredential) IsExpired() bool {
	return time.Now().UTC().After(c.ExpiresAt)
}

func (c *ARCACredential) DaysUntilExpiry() int {
	return int(time.Until(c.ExpiresAt).Hours() / 24)
}

func generateID() string {
	return "cred_" + time.Now().UTC().Format("20060102150405") + "_" + randomString(8)
}

func randomString(n int) string {
	const letters = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, n)
	for i := range b {
		b[i] = letters[time.Now().UnixNano()%int64(len(letters))]
	}
	return string(b)
}

type ARCAClient interface {
	GetLastVoucher(ctx interface{}, cuit string, invoiceType string, pointOfSale int) (int64, error)
	CreateVoucher(ctx interface{}, request VoucherRequest) (*VoucherResponse, error)
}

type VoucherRequest struct {
	CUIT           string
	InvoiceType    string
	PointOfSale    int
	ConceptType    int
	DocType        int
	DocNumber      string
	ServiceFrom    string
	ServiceTo      string
	ExpirationDate string
	Items          []VoucherItem
	CurrencyID     string
	CurrencyRate   float64
}

type VoucherItem struct {
	Description string
	Quantity    float64
	UnitPrice   float64
	Bonus       float64
	Taxes       []VoucherTax
}

type VoucherTax struct {
	TaxID   int
	Base    float64
	Amount  float64
	TaxRate float64
}

type VoucherResponse struct {
	CAE               string
	CAEExpirationDate string
	VoucherNumber     int64
	VoucherType       int
	Result            string
	Observations      []VoucherObservation
}

type VoucherObservation struct {
	Code    int
	Message string
}
