package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"arca-invoice-proxy/internal/domain/apikey"
	"arca-invoice-proxy/internal/domain/credential"
	"arca-invoice-proxy/internal/domain/customer"
	"arca-invoice-proxy/internal/domain/errors"
	"arca-invoice-proxy/internal/domain/idempotency"
	"arca-invoice-proxy/internal/domain/invoice"
)

type UserRepository struct {
	pool *pgxpool.Pool
}

func NewUserRepository(pool *pgxpool.Pool) *UserRepository {
	return &UserRepository{pool: pool}
}

func (r *UserRepository) Create(ctx context.Context, c *customer.Customer) error {
	query := `
		INSERT INTO users (id, cuit, name, email, address, iva_condition, country_code, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`
	id := uuid.New().String()
	now := time.Now().UTC()
	_, err := r.pool.Exec(ctx, query, id, c.CUIT, c.Name, c.Email, c.Address, c.IVACondition.String(), c.CountryCode, now, now)
	if err != nil {
		return errors.Wrap(err, errors.CodeDatabaseError, "create user")
	}
	c.ID = id
	c.CreatedAt = now.Format(time.RFC3339)
	c.UpdatedAt = now.Format(time.RFC3339)
	return nil
}

func (r *UserRepository) GetByCUIT(ctx context.Context, cuit string) (*customer.Customer, error) {
	query := `
		SELECT id, cuit, name, email, address, iva_condition, country_code, created_at, updated_at
		FROM users WHERE cuit = $1
	`
	var c customer.Customer
	var ivaCondition string
	err := r.pool.QueryRow(ctx, query, cuit).Scan(
		&c.ID, &c.CUIT, &c.Name, &c.Email, &c.Address, &ivaCondition, &c.CountryCode, &c.CreatedAt, &c.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, errors.New(errors.CodeNotFound, "user not found")
		}
		return nil, errors.Wrap(err, errors.CodeDatabaseError, "get user by cuit")
	}
	c.IVACondition = customer.IVACondition(ivaCondition)
	return &c, nil
}

func (r *UserRepository) GetByID(ctx context.Context, id string) (*customer.Customer, error) {
	query := `
		SELECT id, cuit, name, email, address, iva_condition, country_code, created_at, updated_at
		FROM users WHERE id = $1
	`
	var c customer.Customer
	var ivaCondition string
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&c.ID, &c.CUIT, &c.Name, &c.Email, &c.Address, &ivaCondition, &c.CountryCode, &c.CreatedAt, &c.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, errors.New(errors.CodeNotFound, "user not found")
		}
		return nil, errors.Wrap(err, errors.CodeDatabaseError, "get user by id")
	}
	c.IVACondition = customer.IVACondition(ivaCondition)
	return &c, nil
}

type APIKeyRepository struct {
	pool *pgxpool.Pool
}

func NewAPIKeyRepository(pool *pgxpool.Pool) *APIKeyRepository {
	return &APIKeyRepository{pool: pool}
}

func (r *APIKeyRepository) Create(ctx context.Context, k *apikey.APIKey) error {
	query := `
		INSERT INTO api_keys (id, user_id, name, prefix, hash, last_four, scopes, expires_at, revoked_at, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	`
	_, err := r.pool.Exec(ctx, query,
		k.ID, k.CustomerID, k.Name, k.Prefix, k.Hash, k.LastFour, k.Scopes, k.ExpiresAt, k.RevokedAt, k.CreatedAt, k.UpdatedAt,
	)
	if err != nil {
		return errors.Wrap(err, errors.CodeDatabaseError, "create api key")
	}
	return nil
}

func (r *APIKeyRepository) GetByPrefixAndSuffix(ctx context.Context, prefix, suffix string) (*apikey.APIKey, error) {
	query := `
		SELECT id, user_id, name, prefix, hash, last_four, scopes, expires_at, revoked_at, created_at, updated_at
		FROM api_keys WHERE prefix = $1 AND last_four = $2
	`
	var k apikey.APIKey
	err := r.pool.QueryRow(ctx, query, prefix, suffix).Scan(
		&k.ID, &k.CustomerID, &k.Name, &k.Prefix, &k.Hash, &k.LastFour, &k.Scopes, &k.ExpiresAt, &k.RevokedAt, &k.CreatedAt, &k.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, errors.New(errors.CodeNotFound, "api key not found")
		}
		return nil, errors.Wrap(err, errors.CodeDatabaseError, "get api key")
	}
	return &k, nil
}

func (r *APIKeyRepository) GetByID(ctx context.Context, id string) (*apikey.APIKey, error) {
	query := `
		SELECT id, user_id, name, prefix, hash, last_four, scopes, expires_at, revoked_at, created_at, updated_at
		FROM api_keys WHERE id = $1
	`
	var k apikey.APIKey
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&k.ID, &k.CustomerID, &k.Name, &k.Prefix, &k.Hash, &k.LastFour, &k.Scopes, &k.ExpiresAt, &k.RevokedAt, &k.CreatedAt, &k.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, errors.New(errors.CodeNotFound, "api key not found")
		}
		return nil, errors.Wrap(err, errors.CodeDatabaseError, "get api key by id")
	}
	return &k, nil
}

func (r *APIKeyRepository) ListByCustomer(ctx context.Context, customerID string) ([]*apikey.APIKey, error) {
	query := `
		SELECT id, user_id, name, prefix, hash, last_four, scopes, expires_at, revoked_at, created_at, updated_at
		FROM api_keys WHERE user_id = $1 ORDER BY created_at DESC
	`
	rows, err := r.pool.Query(ctx, query, customerID)
	if err != nil {
		return nil, errors.Wrap(err, errors.CodeDatabaseError, "list api keys")
	}
	defer rows.Close()

	var keys []*apikey.APIKey
	for rows.Next() {
		var k apikey.APIKey
		if err := rows.Scan(&k.ID, &k.CustomerID, &k.Name, &k.Prefix, &k.Hash, &k.LastFour, &k.Scopes, &k.ExpiresAt, &k.RevokedAt, &k.CreatedAt, &k.UpdatedAt); err != nil {
			return nil, errors.Wrap(err, errors.CodeDatabaseError, "scan api key")
		}
		keys = append(keys, &k)
	}
	return keys, rows.Err()
}

func (r *APIKeyRepository) Update(ctx context.Context, k *apikey.APIKey) error {
	query := `
		UPDATE api_keys SET name = $2, scopes = $3, expires_at = $4, revoked_at = $5, updated_at = $6
		WHERE id = $1
	`
	k.UpdatedAt = time.Now().UTC()
	_, err := r.pool.Exec(ctx, query, k.ID, k.Name, k.Scopes, k.ExpiresAt, k.RevokedAt, k.UpdatedAt)
	if err != nil {
		return errors.Wrap(err, errors.CodeDatabaseError, "update api key")
	}
	return nil
}

type ARCACredentialRepository struct {
	pool *pgxpool.Pool
}

func NewARCACredentialRepository(pool *pgxpool.Pool) *ARCACredentialRepository {
	return &ARCACredentialRepository{pool: pool}
}

func (r *ARCACredentialRepository) Create(ctx context.Context, c *credential.ARCACredential) error {
	query := `
		INSERT INTO arca_credentials (id, user_id, environment, cuit, cert_pem, key_pem, expires_at, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`
	_, err := r.pool.Exec(ctx, query,
		c.ID, c.CustomerID, c.Environment.String(), c.CUIT, c.CertPEM, c.KeyPEM, c.ExpiresAt, c.CreatedAt, c.UpdatedAt,
	)
	if err != nil {
		return errors.Wrap(err, errors.CodeDatabaseError, "create arca credential")
	}
	return nil
}

func (r *ARCACredentialRepository) GetByCustomerAndEnvironment(ctx context.Context, customerID string, env credential.Environment) (*credential.ARCACredential, error) {
	query := `
		SELECT id, user_id, environment, cuit, cert_pem, key_pem, expires_at, created_at, updated_at
		FROM arca_credentials WHERE user_id = $1 AND environment = $2
	`
	var c credential.ARCACredential
	var envStr string
	err := r.pool.QueryRow(ctx, query, customerID, env.String()).Scan(
		&c.ID, &c.CustomerID, &envStr, &c.CUIT, &c.CertPEM, &c.KeyPEM, &c.ExpiresAt, &c.CreatedAt, &c.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, errors.New(errors.CodeNotFound, "arca credential not found")
		}
		return nil, errors.Wrap(err, errors.CodeDatabaseError, "get arca credential")
	}
	c.Environment = credential.Environment(envStr)
	return &c, nil
}

func (r *ARCACredentialRepository) GetByID(ctx context.Context, id string) (*credential.ARCACredential, error) {
	query := `
		SELECT id, user_id, environment, cuit, cert_pem, key_pem, expires_at, created_at, updated_at
		FROM arca_credentials WHERE id = $1
	`
	var c credential.ARCACredential
	var envStr string
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&c.ID, &c.CustomerID, &envStr, &c.CUIT, &c.CertPEM, &c.KeyPEM, &c.ExpiresAt, &c.CreatedAt, &c.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, errors.New(errors.CodeNotFound, "arca credential not found")
		}
		return nil, errors.Wrap(err, errors.CodeDatabaseError, "get arca credential by id")
	}
	c.Environment = credential.Environment(envStr)
	return &c, nil
}

type InvoiceRepository struct {
	pool *pgxpool.Pool
}

func NewInvoiceRepository(pool *pgxpool.Pool) *InvoiceRepository {
	return &InvoiceRepository{pool: pool}
}

func (r *InvoiceRepository) Create(ctx context.Context, inv *invoice.Invoice, items []invoice.InvoiceItem) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return errors.Wrap(err, errors.CodeDatabaseError, "begin transaction")
	}
	defer tx.Rollback(ctx)

	invQuery := `
		INSERT INTO invoices (id, user_id, invoice_type, point_of_sale, status, subtotal_cents, tax_cents, total_cents, idempotency_key, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	`
	_, err = tx.Exec(ctx, invQuery,
		inv.ID, inv.CustomerID, inv.Type.String(), inv.PointOfSale, inv.Status.String(),
		inv.SubtotalCents, inv.TaxCents, inv.TotalCents, inv.IdempotencyKey, inv.CreatedAt, inv.UpdatedAt,
	)
	if err != nil {
		return errors.Wrap(err, errors.CodeDatabaseError, "create invoice")
	}

	itemQuery := `
		INSERT INTO invoice_items (id, invoice_id, description, quantity, unit_price_cents, total_cents, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`
	for _, item := range items {
		itemID := uuid.New().String()
		_, err = tx.Exec(ctx, itemQuery, itemID, inv.ID, item.Description, item.Quantity, item.UnitPriceCents, item.TotalCents(), time.Now().UTC())
		if err != nil {
			return errors.Wrap(err, errors.CodeDatabaseError, "create invoice item")
		}
	}

	return tx.Commit(ctx)
}

func (r *InvoiceRepository) GetByID(ctx context.Context, id string) (*invoice.Invoice, error) {
	query := `
		SELECT i.id, i.user_id, i.invoice_type, i.point_of_sale, i.status, i.subtotal_cents, i.tax_cents, i.total_cents,
		       i.cae, i.cae_expires_at, i.voucher_number, i.voucher_type, i.arca_result, i.arca_observations,
		       i.idempotency_key, i.issued_at, i.created_at, i.updated_at
		FROM invoices i WHERE i.id = $1
	`
	var inv invoice.Invoice
	var invType, status, arcaResult string
	var caeExpiresAt sql.NullTime
	var voucherNumber, voucherType sql.NullInt64
	var arcaObservations []byte
	var issuedAt sql.NullTime

	err := r.pool.QueryRow(ctx, query, id).Scan(
		&inv.ID, &inv.CustomerID, &invType, &inv.PointOfSale, &status,
		&inv.SubtotalCents, &inv.TaxCents, &inv.TotalCents,
		&inv.ARCAResponse.CAE, &caeExpiresAt, &voucherNumber, &voucherType,
		&arcaResult, &arcaObservations, &inv.IdempotencyKey, &issuedAt, &inv.CreatedAt, &inv.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, errors.New(errors.CodeNotFound, "invoice not found")
		}
		return nil, errors.Wrap(err, errors.CodeDatabaseError, "get invoice")
	}

	inv.Type = invoice.InvoiceType(invType)
	inv.Status = invoice.InvoiceStatus(status)
	if caeExpiresAt.Valid {
		inv.ARCAResponse.CAEExpirationDate = caeExpiresAt.Time
	}
	if voucherNumber.Valid {
		inv.ARCAResponse.VoucherNumber = voucherNumber.Int64
	}
	if voucherType.Valid {
		inv.ARCAResponse.VoucherType = int(voucherType.Int64)
	}
	inv.ARCAResponse.Result = arcaResult
	if len(arcaObservations) > 0 {
		json.Unmarshal(arcaObservations, &inv.ARCAResponse.Observations)
	}
	if issuedAt.Valid {
		inv.IssuedAt = &issuedAt.Time
	}

	items, err := r.getItems(ctx, inv.ID)
	if err != nil {
		return nil, err
	}
	inv.Items = items

	return &inv, nil
}

func (r *InvoiceRepository) GetByIdempotencyKey(ctx context.Context, userID, key string) (*invoice.Invoice, error) {
	query := `
		SELECT i.id, i.user_id, i.invoice_type, i.point_of_sale, i.status, i.subtotal_cents, i.tax_cents, i.total_cents,
		       i.cae, i.cae_expires_at, i.voucher_number, i.voucher_type, i.arca_result, i.arca_observations,
		       i.idempotency_key, i.issued_at, i.created_at, i.updated_at
		FROM invoices i WHERE i.user_id = $1 AND i.idempotency_key = $2
	`
	var inv invoice.Invoice
	var invType, status, arcaResult string
	var caeExpiresAt sql.NullTime
	var voucherNumber, voucherType sql.NullInt64
	var arcaObservations []byte
	var issuedAt sql.NullTime

	err := r.pool.QueryRow(ctx, query, userID, key).Scan(
		&inv.ID, &inv.CustomerID, &invType, &inv.PointOfSale, &status,
		&inv.SubtotalCents, &inv.TaxCents, &inv.TotalCents,
		&inv.ARCAResponse.CAE, &caeExpiresAt, &voucherNumber, &voucherType,
		&arcaResult, &arcaObservations, &inv.IdempotencyKey, &issuedAt, &inv.CreatedAt, &inv.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, errors.New(errors.CodeNotFound, "invoice not found")
		}
		return nil, errors.Wrap(err, errors.CodeDatabaseError, "get invoice by idempotency key")
	}

	inv.Type = invoice.InvoiceType(invType)
	inv.Status = invoice.InvoiceStatus(status)
	if caeExpiresAt.Valid {
		inv.ARCAResponse.CAEExpirationDate = caeExpiresAt.Time
	}
	if voucherNumber.Valid {
		inv.ARCAResponse.VoucherNumber = voucherNumber.Int64
	}
	if voucherType.Valid {
		inv.ARCAResponse.VoucherType = int(voucherType.Int64)
	}
	inv.ARCAResponse.Result = arcaResult
	if len(arcaObservations) > 0 {
		json.Unmarshal(arcaObservations, &inv.ARCAResponse.Observations)
	}
	if issuedAt.Valid {
		inv.IssuedAt = &issuedAt.Time
	}

	items, err := r.getItems(ctx, inv.ID)
	if err != nil {
		return nil, err
	}
	inv.Items = items

	return &inv, nil
}

func (r *InvoiceRepository) Update(ctx context.Context, inv *invoice.Invoice) error {
	query := `
		UPDATE invoices SET
			status = $2, cae = $3, cae_expires_at = $4, voucher_number = $5, voucher_type = $6,
			arca_result = $7, arca_observations = $8, issued_at = $9, updated_at = $10
		WHERE id = $1
	`
	var arcaObservations []byte
	if inv.ARCAResponse != nil && len(inv.ARCAResponse.Observations) > 0 {
		arcaObservations, _ = json.Marshal(inv.ARCAResponse.Observations)
	}
	var issuedAt *time.Time
	if inv.IssuedAt != nil {
		issuedAt = inv.IssuedAt
	}

	inv.UpdatedAt = time.Now().UTC()
	_, err := r.pool.Exec(ctx, query,
		inv.ID, inv.Status.String(), inv.ARCAResponse.CAE, inv.ARCAResponse.CAEExpirationDate,
		inv.ARCAResponse.VoucherNumber, inv.ARCAResponse.VoucherType,
		inv.ARCAResponse.Result, arcaObservations, issuedAt, inv.UpdatedAt,
	)
	if err != nil {
		return errors.Wrap(err, errors.CodeDatabaseError, "update invoice")
	}
	return nil
}

func (r *InvoiceRepository) getItems(ctx context.Context, invoiceID string) ([]invoice.InvoiceItem, error) {
	query := `
		SELECT description, quantity, unit_price_cents FROM invoice_items WHERE invoice_id = $1 ORDER BY created_at
	`
	rows, err := r.pool.Query(ctx, query, invoiceID)
	if err != nil {
		return nil, errors.Wrap(err, errors.CodeDatabaseError, "get invoice items")
	}
	defer rows.Close()

	var items []invoice.InvoiceItem
	for rows.Next() {
		var item invoice.InvoiceItem
		if err := rows.Scan(&item.Description, &item.Quantity, &item.UnitPriceCents); err != nil {
			return nil, errors.Wrap(err, errors.CodeDatabaseError, "scan invoice item")
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

type IdempotencyRepository struct {
	pool *pgxpool.Pool
}

func NewIdempotencyRepository(pool *pgxpool.Pool) *IdempotencyRepository {
	return &IdempotencyRepository{pool: pool}
}

func (r *IdempotencyRepository) Create(ctx context.Context, record *idempotency.IdempotencyRecord) error {
	query := `
		INSERT INTO idempotency_keys (key, user_id, request_hash, status, created_at, updated_at, expires_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (key) DO UPDATE SET
			request_hash = EXCLUDED.request_hash,
			status = EXCLUDED.status,
			updated_at = EXCLUDED.updated_at,
			expires_at = EXCLUDED.expires_at
	`
	_, err := r.pool.Exec(ctx, query,
		record.Key, record.RequestHash, record.Status.String(), record.CreatedAt, record.UpdatedAt, record.ExpiresAt,
	)
	if err != nil {
		return errors.Wrap(err, errors.CodeDatabaseError, "create idempotency record")
	}
	return nil
}

func (r *IdempotencyRepository) Get(ctx context.Context, key string) (*idempotency.IdempotencyRecord, error) {
	query := `
		SELECT key, user_id, request_hash, status, response_body, error_message, created_at, updated_at, expires_at
		FROM idempotency_keys WHERE key = $1
	`
	var record idempotency.IdempotencyRecord
	var status string
	var responseBody []byte
	var errorMessage sql.NullString

	err := r.pool.QueryRow(ctx, query, key).Scan(
		&record.Key, &record.RequestHash, &status, &responseBody, &errorMessage,
		&record.CreatedAt, &record.UpdatedAt, &record.ExpiresAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, errors.New(errors.CodeIdempotencyNotFound, "idempotency key not found")
		}
		return nil, errors.Wrap(err, errors.CodeDatabaseError, "get idempotency record")
	}

	record.Status = idempotency.IdempotencyStatus(status)
	record.Response = responseBody
	if errorMessage.Valid {
		record.Error = errorMessage.String
	}
	return &record, nil
}

func (r *IdempotencyRepository) Update(ctx context.Context, record *idempotency.IdempotencyRecord) error {
	query := `
		UPDATE idempotency_keys SET
			status = $2, response_body = $3, error_message = $4, updated_at = $5
		WHERE key = $1
	`
	record.UpdatedAt = time.Now().UTC()
	_, err := r.pool.Exec(ctx, query,
		record.Key, record.Status.String(), record.Response, record.Error, record.UpdatedAt,
	)
	if err != nil {
		return errors.Wrap(err, errors.CodeDatabaseError, "update idempotency record")
	}
	return nil
}