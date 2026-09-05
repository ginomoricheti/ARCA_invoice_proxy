package invoice

import (
	"context"
	"errors"
	"time"

	"arca-invoice-proxy/internal/domain/credential"
	"arca-invoice-proxy/internal/domain/customer"
	"arca-invoice-proxy/internal/domain/errors"
	"arca-invoice-proxy/internal/domain/invoice"
)

type InvoiceRepository interface {
	Create(ctx context.Context, inv *invoice.Invoice, items []invoice.InvoiceItem) error
	GetByID(ctx context.Context, id string) (*invoice.Invoice, error)
	GetByIdempotencyKey(ctx context.Context, userID, key string) (*invoice.Invoice, error)
	Update(ctx context.Context, inv *invoice.Invoice) error
}

type UserRepository interface {
	GetByID(ctx context.Context, id string) (*customer.Customer, error)
}

type ARCACredentialRepository interface {
	GetByCustomerAndEnvironment(ctx context.Context, customerID string, env credential.Environment) (*credential.ARCACredential, error)
}

type ARCAClient interface {
	GetLastVoucher(ctx context.Context, cuit string, invoiceType string, pointOfSale int) (int64, error)
	CreateVoucher(ctx context.Context, request credential.VoucherRequest) (*credential.VoucherResponse, error)
}

type IssueInvoiceInput struct {
	CustomerID   string
	InvoiceType  string
	PointOfSale  int
	Items        []InvoiceItemInput
	IdempotencyKey string
}

type InvoiceItemInput struct {
	Description    string
	Quantity       int
	UnitPriceCents int64
}

type IssueInvoiceOutput struct {
	Invoice *invoice.Invoice
}

type Service struct {
	invoiceRepo       InvoiceRepository
	userRepo          UserRepository
	credentialRepo    ARCACredentialRepository
	arcaClient        ARCAClient
	environment       credential.Environment
}

func NewService(
	invoiceRepo InvoiceRepository,
	userRepo UserRepository,
	credentialRepo ARCACredentialRepository,
	arcaClient ARCAClient,
	environment credential.Environment,
) *Service {
	return &Service{
		invoiceRepo:    invoiceRepo,
		userRepo:       userRepo,
		credentialRepo: credentialRepo,
		arcaClient:     arcaClient,
		environment:    environment,
	}
}

func (s *Service) IssueInvoice(ctx context.Context, input IssueInvoiceInput) (*IssueInvoiceOutput, error) {
	if input.IdempotencyKey != "" {
		existing, err := s.invoiceRepo.GetByIdempotencyKey(ctx, input.CustomerID, input.IdempotencyKey)
		if err != nil && !errors.IsCode(err, errors.CodeNotFound) {
			return nil, err
		}
		if existing != nil {
			return &IssueInvoiceOutput{Invoice: existing}, nil
		}
	}

	invType, err := invoice.ParseInvoiceType(input.InvoiceType)
	if err != nil {
		return nil, errors.Wrap(err, errors.CodeInvalidInvoiceType, "invalid invoice type")
	}

	user, err := s.userRepo.GetByID(ctx, input.CustomerID)
	if err != nil {
		return nil, err
	}

	cred, err := s.credentialRepo.GetByCustomerAndEnvironment(ctx, input.CustomerID, s.environment)
	if err != nil {
		if errors.IsCode(err, errors.CodeNotFound) {
			return nil, errors.New(errors.CodeARCAAuthFailed, "arca credentials not configured")
		}
		return nil, err
	}

	if cred.IsExpired() {
		return nil, errors.New(errors.CodeARCAAuthFailed, "arca credentials expired")
	}

	var items []invoice.InvoiceItem
	for _, itemInput := range input.Items {
		item, err := invoice.NewInvoiceItem(itemInput.Description, itemInput.Quantity, itemInput.UnitPriceCents)
		if err != nil {
			return nil, errors.Wrap(err, errors.CodeInvalidAmount, "invalid invoice item")
		}
		items = append(items, item)
	}

	inv, err := invoice.NewInvoice(input.CustomerID, invType, input.PointOfSale, items, input.IdempotencyKey)
	if err != nil {
		return nil, errors.Wrap(err, errors.CodeValidationFailed, "create invoice")
	}

	lastVoucher, err := s.arcaClient.GetLastVoucher(ctx, cred.CUIT, invType.String(), input.PointOfSale)
	if err != nil {
		return nil, errors.Wrap(err, errors.CodeARCAServiceUnavailable, "get last voucher")
	}

	voucherNumber := lastVoucher + 1

	voucherReq := s.buildVoucherRequest(cred, user, inv, voucherNumber)
	arcaResp, err := s.arcaClient.CreateVoucher(ctx, voucherReq)
	if err != nil {
		inv.MarkRejected([]string{err.Error()})
		s.invoiceRepo.Create(ctx, inv, items)
		return nil, errors.Wrap(err, errors.CodeARCARejected, "arca rejected invoice")
	}

	if arcaResp.Result != "A" && arcaResp.Result != "approved" {
		observations := make([]string, len(arcaResp.Observations))
		for i, obs := range arcaResp.Observations {
			observations[i] = obs.Message
		}
		inv.MarkRejected(observations)
		s.invoiceRepo.Create(ctx, inv, items)
		return nil, errors.New(errors.CodeARCARejected, "arca rejected invoice")
	}

	inv.MarkIssued(invoice.ARCAResponse{
		CAE:               arcaResp.CAE,
		CAEExpirationDate: parseDate(arcaResp.CAEExpirationDate),
		VoucherNumber:     arcaResp.VoucherNumber,
		VoucherType:       arcaResp.VoucherType,
		Result:            arcaResp.Result,
		Observations:      observationsFromARCA(arcaResp.Observations),
	})

	if err := s.invoiceRepo.Create(ctx, inv, items); err != nil {
		return nil, errors.Wrap(err, errors.CodeDatabaseError, "persist invoice")
	}

	return &IssueInvoiceOutput{Invoice: inv}, nil
}

func (s *Service) buildVoucherRequest(cred *credential.ARCACredential, user *customer.Customer, inv *invoice.Invoice, voucherNumber int64) credential.VoucherRequest {
	voucherType := mapInvoiceTypeToARCA(inv.Type)
	docType := 80
	docNumber := user.CUIT

	if inv.Type == invoice.InvoiceTypeC {
		docType = 99
		docNumber = "0"
	}

	var items []credential.VoucherItem
	for _, item := range inv.Items {
		items = append(items, credential.VoucherItem{
			Description: item.Description,
			Quantity:    float64(item.Quantity),
			UnitPrice:   float64(item.UnitPriceCents) / 100.0,
			Bonus:       0,
			Taxes:       []credential.VoucherTax{},
		})
	}

	return credential.VoucherRequest{
		CUIT:            cred.CUIT,
		InvoiceType:     voucherType,
		PointOfSale:     inv.PointOfSale,
		ConceptType:     1,
		DocType:         docType,
		DocNumber:       docNumber,
		ServiceFrom:     time.Now().UTC().Format("20060102"),
		ServiceTo:       time.Now().UTC().Format("20060102"),
		ExpirationDate:  time.Now().UTC().AddDate(0, 0, 10).Format("20060102"),
		Items:           items,
		CurrencyID:      "PES",
		CurrencyRate:    1.0,
	}
}

func mapInvoiceTypeToARCA(t invoice.InvoiceType) string {
	switch t {
	case invoice.InvoiceTypeA:
		return "1"
	case invoice.InvoiceTypeB:
		return "6"
	case invoice.InvoiceTypeC:
		return "11"
	default:
		return "11"
	}
}

func parseDate(dateStr string) time.Time {
	t, _ := time.Parse("20060102", dateStr)
	return t
}

func observationsFromARCA(obs []credential.VoucherObservation) []string {
	result := make([]string, len(obs))
	for i, o := range obs {
		result[i] = o.Message
	}
	return result
}