package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"example.com/order-service/internal/order"
)

type OrderService struct {
	repository order.Repository
}

func NewOrderService(repository order.Repository) *OrderService {
	return &OrderService{repository: repository}
}

func (s *OrderService) Create(ctx context.Context, input order.CreateOrderInput) (order.Order, error) {
	if len(input.Items) == 0 {
		return order.Order{}, order.ErrInvalidOrder
	}
	totalPrice := int64(0) //кроме как int64 защиты от переполнения нет, я хз
	for _, item := range input.Items {
		item.Name = strings.TrimSpace(item.Name)
		if item.Name == "" {
			return order.Order{}, order.ErrInvalidOrder
		}
		if item.Quantity <= 0 || item.Price <= 0 {
			return order.Order{}, order.ErrInvalidOrder
		}
		totalPrice += item.Price * int64(item.Quantity)
	}
	now := time.Now()
	history := []order.StatusChange{{Status: order.StatusNew, ChangedAt: now}}

	ord := order.Order{
		Items:      input.Items,
		TotalPrice: totalPrice,
		Status:     order.StatusNew,
		CreatedAt:  now,
		History:    history,
	} //передаем без ID, в repository он присвоится

	res, err := s.repository.Create(ctx, ord)
	if err != nil {
		return order.Order{}, fmt.Errorf("InternalError: %w", err) //у меня из repo.Create всегда nil возвращается, но если (в теории) там возникает ошибка, то это 500
	}
	return res, nil
}

func (s *OrderService) GetByID(ctx context.Context, id int64) (order.Order, error) {
	if id <= 0 {
		return order.Order{}, order.ErrInvalidOrder
	}
	res, err := s.repository.GetByID(ctx, id)
	if err != nil {
		return order.Order{}, fmt.Errorf("получение заказа %d: %w", id, err)
	}
	return res, nil
}

func (s *OrderService) List(ctx context.Context) ([]order.Order, error) {
	res, err := s.repository.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("лист заказов: %w", err)
	}
	return res, nil
}

func (s *OrderService) UpdateStatus(ctx context.Context, id int64, status order.Status) (order.Order, error) {
	if id <= 0 {
		return order.Order{}, fmt.Errorf("invalid id: %w", order.ErrInvalidOrder)
	}
	if len(status) == 0 {
		return order.Order{}, fmt.Errorf("updating status : %w", order.ErrInvalidStatus)
	}
	ord, err := s.repository.GetByID(ctx, id)
	if err != nil {
		return order.Order{}, fmt.Errorf("updating status: %w", err)
	}
	switch status {
	case order.StatusCancelled, order.StatusCompleted, order.StatusNew, order.StatusProcessing:
		// проверка на то, чтобы status был в захардкоженных статусах из model
	default:
		return order.Order{}, order.ErrInvalidStatus
	}
	newOrder := ord
	switch ord.Status {
	case order.StatusNew:
		if status == order.StatusCompleted || status == order.StatusNew {
			return order.Order{}, order.ErrInvalidStatusTransition
		}

	case order.StatusProcessing:
		if status == order.StatusNew || status == order.StatusProcessing {
			return order.Order{}, order.ErrInvalidStatusTransition
		}
	case order.StatusCancelled, order.StatusCompleted:
		return order.Order{}, order.ErrInvalidStatusTransition

	default:
		return order.Order{}, order.ErrInvalidStatus
	}
	newOrder.Status = status
	newOrder.History = append(newOrder.History, order.StatusChange{Status: status, ChangedAt: time.Now()})

	res, err := s.repository.Update(ctx, newOrder)
	if err != nil {
		return order.Order{}, fmt.Errorf("updating status: %w", err)
	}
	return res, nil
} //да, два свича, но такая жизнь

func (s *OrderService) History(ctx context.Context, id int64) ([]order.StatusChange, error) {
	if id <= 0 {
		return nil, fmt.Errorf("invalid id: %w", order.ErrInvalidOrder)
	}
	ord, err := s.repository.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("extracting history: %w", err)
	}
	return ord.History, nil //не сортирую, тк и так история должна быть упорядочена
}
