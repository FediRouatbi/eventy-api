package admins

import (
	"bytes"
	"context"
	"encoding/csv"
	"fmt"
	"strings"
	"time"

	"eventy-api/internal/platform/jwt"
	"eventy-api/internal/platform/roles"

	"github.com/google/uuid"
)

type Service struct {
	repository *Repository
}

func NewService(repository *Repository) *Service {
	return &Service{repository: repository}
}

func (s *Service) CreateOrganizerAdmin(ctx context.Context, input CreateOrganizerAdminInput) (CreateOrganizerAdminResult, error) {
	input.OrganizerName = strings.TrimSpace(input.OrganizerName)
	input.OrganizerSlug = strings.TrimSpace(input.OrganizerSlug)
	input.AdminName = strings.TrimSpace(input.AdminName)
	input.AdminEmail = strings.TrimSpace(strings.ToLower(input.AdminEmail))

	if err := validateCreateOrganizerAdminInput(input); err != nil {
		return CreateOrganizerAdminResult{}, err
	}

	return s.repository.CreateOrganizerAdmin(ctx, input)
}

func (s *Service) GetOverview(ctx context.Context, claims *jwt.Claims) (AdminOverview, error) {
	switch claims.Role {
	case roles.SuperAdmin:
		return s.repository.GetOverview(ctx, nil, true)
	case roles.OrganizerAdmin:
		if claims.OrganizerID == nil {
			return AdminOverview{}, ErrOrganizerScopeRequired
		}

		return s.repository.GetOverview(ctx, claims.OrganizerID, false)
	default:
		return AdminOverview{}, ErrOrganizerScopeRequired
	}
}

func (s *Service) GetPayments(ctx context.Context, claims *jwt.Claims, limit int) (AdminPayments, error) {
	if limit <= 0 {
		return AdminPayments{}, ErrInvalidLimit
	}
	if limit > 100 {
		limit = 100
	}

	switch claims.Role {
	case roles.SuperAdmin:
		return s.repository.GetPayments(ctx, nil, true, limit)
	case roles.OrganizerAdmin:
		if claims.OrganizerID == nil {
			return AdminPayments{}, ErrOrganizerScopeRequired
		}

		return s.repository.GetPayments(ctx, claims.OrganizerID, false, limit)
	default:
		return AdminPayments{}, ErrOrganizerScopeRequired
	}
}

func (s *Service) ExportPaymentsCSV(ctx context.Context, claims *jwt.Claims, filters AdminPaymentsExportFilters, timezoneLabel string) ([]byte, string, error) {
	var (
		rows []AdminPaymentExportRow
		err  error
	)

	switch claims.Role {
	case roles.SuperAdmin:
		rows, err = s.repository.ListPaymentsForExport(ctx, filters)
	case roles.OrganizerAdmin:
		if claims.OrganizerID == nil {
			return nil, "", ErrOrganizerScopeRequired
		}

		filters.OrganizerID = claims.OrganizerID
		rows, err = s.repository.ListPaymentsForExport(ctx, filters)
	default:
		return nil, "", ErrOrganizerScopeRequired
	}
	if err != nil {
		return nil, "", err
	}

	buf := &bytes.Buffer{}
	writer := csv.NewWriter(buf)

	if err := writer.Write([]string{
		"order_id",
		"order_number",
		"status",
		"amount",
		"currency",
		"customer_name",
		"customer_email",
		"event_titles",
		"organizer_id",
		"organizer_name",
		"paid_at",
		"created_at",
		"updated_at",
	}); err != nil {
		return nil, "", err
	}

	for _, row := range rows {
		organizerID := ""
		if row.OrganizerID != nil {
			organizerID = row.OrganizerID.String()
		}

		paidAt := ""
		if row.PaidAt != nil {
			paidAt = row.PaidAt.UTC().Format(time.RFC3339)
		}

		if err := writer.Write([]string{
			row.OrderID.String(),
			row.OrderNumber,
			row.Status,
			fmt.Sprintf("%.2f", row.Amount),
			row.Currency,
			row.CustomerName,
			row.CustomerEmail,
			row.EventTitles,
			organizerID,
			row.OrganizerName,
			paidAt,
			row.CreatedAt.UTC().Format(time.RFC3339),
			row.UpdatedAt.UTC().Format(time.RFC3339),
		}); err != nil {
			return nil, "", err
		}
	}

	writer.Flush()
	if err := writer.Error(); err != nil {
		return nil, "", err
	}

	filename := fmt.Sprintf(
		"finance-export-%s-to-%s-%s.csv",
		filters.FromDate.UTC().Format("2006-01-02"),
		filters.ToDate.UTC().Format("2006-01-02"),
		strings.NewReplacer("/", "-", "\\", "-", " ", "_", ":", "-").Replace(strings.TrimSpace(timezoneLabel)),
	)

	return buf.Bytes(), filename, nil
}

func (s *Service) UpdateOrganizer(ctx context.Context, organizerID uuid.UUID, input UpdateOrganizerInput) (Organizer, error) {
	input.OrganizerName = strings.TrimSpace(input.OrganizerName)
	input.OrganizerSlug = strings.TrimSpace(input.OrganizerSlug)

	if err := validateUpdateOrganizerInput(input); err != nil {
		return Organizer{}, err
	}

	return s.repository.UpdateOrganizer(ctx, organizerID, input)
}

func (s *Service) ListOrganizers(ctx context.Context) ([]OrganizerListItem, error) {
	return s.repository.ListOrganizers(ctx)
}

func (s *Service) GetOrganizer(ctx context.Context, organizerID uuid.UUID) (OrganizerDetail, error) {
	return s.repository.GetOrganizer(ctx, organizerID)
}

func (s *Service) ListOrganizerAdmins(ctx context.Context, organizerID uuid.UUID) ([]OrganizerAdmin, error) {
	return s.repository.ListOrganizerAdmins(ctx, organizerID)
}

func (s *Service) AddOrganizerAdmin(ctx context.Context, organizerID uuid.UUID, input AddOrganizerAdminInput) (OrganizerAdmin, error) {
	input.AdminName = strings.TrimSpace(input.AdminName)
	input.AdminEmail = strings.TrimSpace(strings.ToLower(input.AdminEmail))

	if err := validateAddOrganizerAdminInput(input); err != nil {
		return OrganizerAdmin{}, err
	}

	return s.repository.AddOrganizerAdmin(ctx, organizerID, input)
}

func (s *Service) UpdateOrganizerAdmin(ctx context.Context, organizerID uuid.UUID, adminID uuid.UUID, input UpdateOrganizerAdminInput) (OrganizerAdmin, error) {
	input.AdminName = strings.TrimSpace(input.AdminName)
	input.AdminEmail = strings.TrimSpace(strings.ToLower(input.AdminEmail))

	if err := validateUpdateOrganizerAdminInput(input); err != nil {
		return OrganizerAdmin{}, err
	}

	return s.repository.UpdateOrganizerAdmin(ctx, organizerID, adminID, input)
}

func (s *Service) GetOrganizerAdmin(ctx context.Context, organizerID uuid.UUID, adminID uuid.UUID) (OrganizerAdmin, error) {
	return s.repository.GetOrganizerAdmin(ctx, organizerID, adminID)
}

func (s *Service) DeleteOrganizerAdmin(ctx context.Context, organizerID uuid.UUID, adminID uuid.UUID) error {
	return s.repository.DeleteOrganizerAdmin(ctx, organizerID, adminID)
}

func (s *Service) DeleteOrganizer(ctx context.Context, organizerID uuid.UUID) error {
	return s.repository.DeleteOrganizer(ctx, organizerID)
}
