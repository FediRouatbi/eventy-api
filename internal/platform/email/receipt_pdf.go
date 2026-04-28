package email

import (
	"bytes"
	"fmt"
	"strings"
	"time"

	"github.com/jung-kurt/gofpdf"
)

type ReceiptPDFItem struct {
	TicketTypeName  string
	EventTitle      string
	Quantity        int32
	UnitPrice       float64
	Currency        string
	SessionStartsAt time.Time
	SessionEndsAt   time.Time
}

type ReceiptPDFInput struct {
	OrderNumber   string
	Status        string
	CustomerName  string
	CustomerEmail string
	Currency      string
	Subtotal      float64
	CreatedAt     time.Time
	UpdatedAt     time.Time
	PaidAt        *time.Time
	Items         []ReceiptPDFItem
}

func BuildReceiptPDF(input ReceiptPDFInput) ([]byte, error) {
	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.SetMargins(14, 14, 14)
	pdf.SetAutoPageBreak(true, 14)
	pdf.AddPage()

	write := func(text string, style string, size float64) {
		pdf.SetFont("Helvetica", style, size)
		pdf.MultiCell(0, 6, text, "", "L", false)
	}

	money := func(amount float64, currency string) string {
		code := strings.ToUpper(strings.TrimSpace(currency))
		if code == "" {
			code = "EUR"
		}
		return fmt.Sprintf("%s %.2f", code, amount)
	}

	write("Eventy Receipt", "B", 18)
	pdf.Ln(1)
	write(fmt.Sprintf("Order: %s", strings.TrimSpace(input.OrderNumber)), "B", 12)
	write(fmt.Sprintf("Status: %s", strings.TrimSpace(input.Status)), "", 11)
	pdf.Ln(1)
	write(fmt.Sprintf("Customer: %s", strings.TrimSpace(input.CustomerName)), "", 11)
	write(fmt.Sprintf("Email: %s", strings.TrimSpace(input.CustomerEmail)), "", 11)
	write(fmt.Sprintf("Created: %s", input.CreatedAt.Local().Format(time.RFC1123)), "", 10)
	if input.PaidAt != nil {
		write(fmt.Sprintf("Paid: %s", input.PaidAt.Local().Format(time.RFC1123)), "", 10)
	}
	write(fmt.Sprintf("Updated: %s", input.UpdatedAt.Local().Format(time.RFC1123)), "", 10)
	pdf.Ln(2)
	write("Items", "B", 13)

	for _, item := range input.Items {
		total := float64(item.Quantity) * item.UnitPrice
		write(fmt.Sprintf("%s x %d — %s", strings.TrimSpace(item.TicketTypeName), item.Quantity, money(total, item.Currency)), "B", 11)
		write(strings.TrimSpace(item.EventTitle), "", 10)
		write(
			fmt.Sprintf(
				"%s - %s",
				item.SessionStartsAt.Local().Format("2006-01-02 15:04"),
				item.SessionEndsAt.Local().Format("2006-01-02 15:04"),
			),
			"",
			10,
		)
		write(fmt.Sprintf("Unit price: %s", money(item.UnitPrice, item.Currency)), "", 10)
		pdf.Ln(1)
	}

	write(fmt.Sprintf("Total: %s", money(input.Subtotal, input.Currency)), "B", 12)

	var buf bytes.Buffer
	if err := pdf.Output(&buf); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}
