package memory

import (
	"context"

	"example.com/order-service/internal/order"
)

type OrderRepository struct {
	items  []order.Order
	nextID int64
}

func NewOrderRepository() *OrderRepository {
	return &OrderRepository{
		items:  make([]order.Order, 0),
		nextID: 1,
	}
}

func (r *OrderRepository) Create(_ context.Context, value order.Order) (order.Order, error) {
	value.ID = r.nextID
	r.nextID++
	saved := value
	itemsCopy := make([]order.Item, len(value.Items))
	historyCopy := make([]order.StatusChange, len(value.History))
	copy(itemsCopy, value.Items)
	copy(historyCopy, value.History)
	saved.Items = itemsCopy
	saved.History = historyCopy
	r.items = append(r.items, saved)
	return value, nil

}

func (r *OrderRepository) GetByID(_ context.Context, id int64) (order.Order, error) {
	for _, ord := range r.items {
		if ord.ID == id {
			found := ord
			itemsCopy := make([]order.Item, len(ord.Items))
			historyCopy := make([]order.StatusChange, len(ord.History))
			copy(itemsCopy, ord.Items)
			copy(historyCopy, ord.History)
			found.Items = itemsCopy
			found.History = historyCopy
			return found, nil
		}
	}

	return order.Order{}, order.ErrOrderNotFound
}

func (r *OrderRepository) List(_ context.Context) ([]order.Order, error) {
	res := make([]order.Order, len(r.items))
	for i, ord := range r.items {
		ordCopy := ord
		itemsCopy := make([]order.Item, len(ord.Items))
		historyCopy := make([]order.StatusChange, len(ord.History))
		copy(itemsCopy, ord.Items)
		copy(historyCopy, ord.History)
		ordCopy.Items = itemsCopy
		ordCopy.History = historyCopy
		res[i] = ordCopy
	}
	return res, nil
}

func (r *OrderRepository) Update(_ context.Context, value order.Order) (order.Order, error) {
	for i, oldOrder := range r.items {
		if oldOrder.ID == value.ID {
			newOrder := value
			itemsCopy := make([]order.Item, len(value.Items))
			historyCopy := make([]order.StatusChange, len(value.History))
			copy(itemsCopy, value.Items)
			copy(historyCopy, value.History)
			newOrder.Items = itemsCopy
			newOrder.History = historyCopy
			r.items[i] = newOrder
			return value, nil
		}
	}
	return order.Order{}, order.ErrOrderNotFound
}
