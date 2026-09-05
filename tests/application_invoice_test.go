package tests

import (
	"context"
	"testing"
	"time"

	"arca-invoice-proxy/internal/application/invoice"
	"arca-invoice-proxy/internal/domain/credential"
	"arca-invoice-proxy/internal/domain/customer"
	"arca-invoice-proxy/internal/domain/errors"
	"arca-invoice-proxy/internal/domain/invoice"
)

// Mock implementations
type mockInvoiceRepo struct {
	invoices map[string]*invoice.Invoice
}

func newMockInvoiceRepo() *mockInvoiceRepo {
	return &mockInvoiceRepo{invoices: make(map[string]*invoice.Invoice)}
}

func (m *mockInvoiceRepo) Create(ctx context.Context, inv *invoice.Invoice, items []invoice.InvoiceItem) error {
	m.invoices[inv.ID] = inv
	return nil
}

func (m *mockInvoiceRepo) GetByID(ctx context.Context, id string) (*invoice.Invoice, error) {
	if inv, ok := m.invoices[id]; ok {
		return inv, nil
	}
	return nil, errors.New(errors.CodeNotFound, "not found")
}

func (m *mockInvoiceRepo) GetByIdempotencyKey(ctx context.Context, userID, key string) (*invoice.Invoice, error) {
	for _, inv := range m.invoices {
		if inv.CustomerID == userID && inv.IdempotencyKey == key {
			return inv, nil
		}
	}
	return nil, errors.New(errors.CodeNotFound, "not found")
}

func (m *mockInvoiceRepo) Update(ctx context.Context, inv *invoice.Invoice) error {
	m.invoices[inv.ID] = inv
	return nil
}

type mockUserRepo struct {
	users map[string]*customer.Customer
}

func newMockUserRepo() *mockUserRepo {
	return &mockUserRepo{users: make(map[string]*customer.Customer)}
}

func (m *mockUserRepo) GetByID(ctx context.Context, id string) (*customer.Customer, error) {
	if u, ok := m.users[id]; ok {
		return u, nil
	}
	return nil, errors.New(errors.CodeNotFound, "user not found")
}

func (m *mockUserRepo) AddUser(user *customer.Customer) {
	m.users[user.ID] = user
}

type mockCredentialRepo struct {
	creds map[string]*credential.ARCACredential
}

func newMockCredentialRepo() *mockCredentialRepo {
	return &mockCredentialRepo{creds: make(map[string]*credential.ARCACredential)}
}

func (m *mockCredentialRepo) GetByCustomerAndEnvironment(ctx context.Context, customerID string, env credential.Environment) (*credential.ARCACredential, error) {
	key := customerID + ":" + env.String()
	if c, ok := m.creds[key]; ok {
		return c, nil
	}
	return nil, errors.New(errors.CodeNotFound, "credential not found")
}

func (m *mockCredentialRepo) AddCredential(cred *credential.ARCACredential) {
	key := cred.CustomerID + ":" + cred.Environment.String()
	m.creds[key] = cred
}

type mockARCAClient struct {
	lastVoucher int64
	voucherResp *credential.VoucherResponse
	err         error
}

func newMockARCAClient() *mockARCAClient {
	return &mockARCAClient{
		lastVoucher: 0,
		voucherResp: &credential.VoucherResponse{
			CAE:               "12345678901234",
			CAEExpirationDate: time.Now().AddDate(0, 0, 10).Format("20060102"),
			VoucherNumber:     1,
			VoucherType:       11,
			Result:            "A",
			Observations:      []credential.VoucherObservation{},
		},
	}
}

func (m *mockARCAClient) GetLastVoucher(ctx context.Context, cuit, invoiceType string, pointOfSale int) (int64, error) {
	return m.lastVoucher, m.err
}

func (m *mockARCAClient) CreateVoucher(ctx context.Context, request credential.VoucherRequest) (*credential.VoucherResponse, error) {
	return m.voucherResp, m.err
}

func TestIssueInvoice_Success(t *testing.T) {
	ctx := context.Background()
	
	invoiceRepo := newMockInvoiceRepo()
	userRepo := newMockUserRepo()
	credentialRepo := newMockCredentialRepo()
	arcaClient := newMockARCAClient()

	// Setup user
	user, _ := customer.NewCustomer("20123456789", "Test Co", "test@example.com", "Address 123", customer.IVAConditionRI, "AR")
	user.ID = "cust_123"
	userRepo.AddUser(user)

	// Setup credential
	cred, _ := credential.NewARCACredential("cust_123", credential.EnvironmentHomologation, "20123456789", []byte("cert"), []byte("key"), time.Now().Add(24*time.Hour))
	credentialRepo.AddCredential(cred)

	svc := invoice.NewService(invoiceRepo, userRepo, credentialRepo, arcaClient, credential.EnvironmentHomologation)

	input := invoice.IssueInvoiceInput{
		CustomerID:   "cust_123",
		InvoiceType:  "C",
		PointOfSale:  1,
		IdempotencyKey: "idem_123",
		Items: []invoice.InvoiceItemInput{
			{Description: "Service", Quantity: 1, UnitPriceCents: 100000},
		},
	}

	output, err := svc.IssueInvoice(ctx, input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if output == nil {
		t.Fatal("output should not be nil")
	}
	if output.Invoice == nil {
		t.Fatal("invoice should not be nil")
	}
	if output.Invoice.Status != invoice.InvoiceStatusIssued {
		t.Errorf("status should be issued, got %s", output.Invoice.Status)
	}
	if output.Invoice.ARCAResponse == nil {
		t.Error("ARCA response should be set")
	}
	if output.Invoice.ARCAResponse.CAE != "12345678901234" {
		t.Error("CAE mismatch")
	}
}

func TestIssueInvoice_Idempotency(t *testing.T) {
	ctx := context.Background()
	
	invoiceRepo := newMockInvoiceRepo()
	userRepo := newMockUserRepo()
	credentialRepo := newMockCredentialRepo()
	arcaClient := newMockARCAClient()

	user, _ := customer.NewCustomer("20123456789", "Test Co", "test@example.com", "Address 123", customer.IVAConditionRI, "AR")
	user.ID = "cust_123"
	userRepo.AddUser(user)

	cred, _ := credential.NewARCACredential("cust_123", credential.EnvironmentHomologation, "20123456789", []byte("cert"), []byte("key"), time.Now().Add(24*time.Hour))
	credentialRepo.AddCredential(cred)

	svc := invoice.NewService(invoiceRepo, userRepo, credentialRepo, arcaClient, credential.EnvironmentHomologation)

	input := invoice.IssueInvoiceInput{
		CustomerID:   "cust_123",
		InvoiceType:  "C",
		PointOfSale:  1,
		IdempotencyKey: "idem_123",
		Items: []invoice.InvoiceItemInput{
			{Description: "Service", Quantity: 1, UnitPriceCents: 100000},
		},
	}

	// First call
	output1, err := svc.IssueInvoice(ctx, input)
	if err != nil {
		t.Fatalf("first call failed: %v", err)
	}

	// Second call with same idempotency key
	output2, err := svc.IssueInvoice(ctx, input)
	if err != nil {
		t.Fatalf("second call failed: %v", err)
	}

	// Should return same invoice
	if output1.Invoice.ID != output2.Invoice.ID {
		t.Error("idempotent calls should return same invoice")
	}
}

func TestIssueInvoice_InvalidType(t *testing.T) {
	ctx := context.Background()
	
	invoiceRepo := newMockInvoiceRepo()
	userRepo := newMockUserRepo()
	credentialRepo := newMockCredentialRepo()
	arcaClient := newMockARCAClient()

	user, _ := customer.NewCustomer("20123456789", "Test Co", "test@example.com", "Address 123", customer.IVAConditionRI, "AR")
	user.ID = "cust_123"
	userRepo.AddUser(user)

	cred, _ := credential.NewARCACredential("cust_123", credential.EnvironmentHomologation, "20123456789", []byte("cert"), []byte("key"), time.Now().Add(24*time.Hour))
	credentialRepo.AddCredential(cred)

	svc := invoice.NewService(invoiceRepo, userRepo, credentialRepo, arcaClient, credential.EnvironmentHomologation)

	input := invoice.IssueInvoiceInput{
		CustomerID:   "cust_123",
		InvoiceType:  "X",
		PointOfSale:  1,
		Items: []invoice.InvoiceItemInput{
			{Description: "Service", Quantity: 1, UnitPriceCents: 100000},
		},
	}

	_, err := svc.IssueInvoice(ctx, input)
	if err == nil {
		t.Error("expected error for invalid invoice type")
	}
	if !errors.IsCode(err, errors.CodeInvalidInvoiceType) {
		t.Errorf("expected CodeInvalidInvoiceType, got %s", errors.GetCode(err))
	}
}

func TestIssueInvoice_MissingCredential(t *testing.T) {
	ctx := context.Background()
	
	invoiceRepo := newMockInvoiceRepo()
	userRepo := newMockUserRepo()
	credentialRepo := newMockCredentialRepo()
	arcaClient := newMockARCAClient()

	user, _ := customer.NewCustomer("20123456789", "Test Co", "test@example.com", "Address 123", customer.IVAConditionRI, "AR")
	user.ID = "cust_123"
	userRepo.AddUser(user)
	// No credential added

	svc := invoice.NewService(invoiceRepo, userRepo, credentialRepo, arcaClient, credential.EnvironmentHomologation)

	input := invoice.IssueInvoiceInput{
		CustomerID:   "cust_123",
		InvoiceType:  "C",
		PointOfSale:  1,
		Items: []invoice.InvoiceItemInput{
			{Description: "Service", Quantity: 1, UnitPriceCents: 100000},
		},
	}

	_, err := svc.IssueInvoice(ctx, input)
	if err == nil {
		t.Error("expected error for missing credential")
	}
	if !errors.IsCode(err, errors.CodeARCAAuthFailed) {
		t.Errorf("expected CodeARCAAuthFailed, got %s", errors.GetCode(err))
	}
}

func TestIssueInvoice_ARCARejected(t *testing.T) {
	ctx := context.Background()
	
	invoiceRepo := newMockInvoiceRepo()
	userRepo := newMockUserRepo()
	credentialRepo := newMockCredentialRepo()
	arcaClient := newMockARCAClient()
	arcaClient.voucherResp = &credential.VoucherResponse{
		CAE:               "",
		CAEExpirationDate: "",
		VoucherNumber:     0,
		VoucherType:       11,
		Result:            "R",
		Observations:      []credential.VoucherObservation{{Code: 1001, Message: "CUIT no autorizado"}},
	}

	user, _ := customer.NewCustomer("20123456789", "Test Co", "test@example.com", "Address 123", customer.IVAConditionRI, "AR")
	user.ID = "cust_123"
	userRepo.AddUser(user)

	cred, _ := credential.NewARCACredential("cust_123", credential.EnvironmentHomologation, "20123456789", []byte("cert"), []byte("key"), time.Now().Add(24*time.Hour))
	credentialRepo.AddCredential(cred)

	svc := invoice.NewService(invoiceRepo, userRepo, credentialRepo, arcaClient, credential.EnvironmentHomologation)

	input := invoice.IssueInvoiceInput{
		CustomerID:   "cust_123",
		InvoiceType:  "C",
		PointOfSale:  1,
		Items: []invoice.InvoiceItemInput{
			{Description: "Service", Quantity: 1, UnitPriceCents: 100000},
		},
	}

	_, err := svc.IssueInvoice(ctx, input)
	if err == nil {
		t.Error("expected error for ARCA rejection")
	}
	if !errors.IsCode(err, errors.CodeARCARejected) {
		t.Errorf("expected CodeARCARejected, got %s", errors.GetCode(err))
	}

	// Invoice should be persisted with rejected status
	for _, inv := range invoiceRepo.invoices {
		if inv.Status != invoice.InvoiceStatusRejected {
			t.Error("invoice should be marked as rejected")
		}
	}
}