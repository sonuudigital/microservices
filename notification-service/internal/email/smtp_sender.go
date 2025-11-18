package email

import (
	"fmt"

	"github.com/sonuudigital/microservices/shared/events"
	"gopkg.in/gomail.v2"
)

type SMTPSender struct {
	dialer *gomail.Dialer
	from   string
}

func NewSMTPSender(host string, port int, username, password, from string) *SMTPSender {
	dialer := gomail.NewDialer(host, port, username, password)
	dialer.TLSConfig = nil
	return &SMTPSender{
		dialer: dialer,
		from:   from,
	}
}

func (s *SMTPSender) Send(data any) error {
	switch event := data.(type) {
	case events.OrderCreatedEvent:
		return s.sendOrderCreated(event)
	default:
		return fmt.Errorf("unsupported event type: %T", data)
	}
}

func (s *SMTPSender) sendOrderCreated(event events.OrderCreatedEvent) error {
	m := gomail.NewMessage(func(m *gomail.Message) {
		m.SetHeader("From", s.from)
		m.SetHeader("To", event.UserEmail)
		m.SetHeader("Subject", "Order Created: "+event.OrderID)
		body := "Dear User,\n\n"
		body += "Thank you for your order. Here are the details:\n\n"
		for _, item := range event.Products {
			body += "- Product ID: " + item.ProductID + ", Quantity: " + fmt.Sprintf("%d", item.Quantity) + "\n"
		}
		body += "\nWe appreciate your business!\n"
		m.SetBody("text/plain", body)
	})
	return s.dialer.DialAndSend(m)
}
