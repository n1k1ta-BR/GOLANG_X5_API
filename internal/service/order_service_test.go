package service_test

import (
	"context"
	"testing"

	"example.com/order-service/internal/order"
	"example.com/order-service/internal/service"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeRepository struct {
	orders []order.Order
}

func (r *fakeRepository) Create(_ context.Context, value order.Order) (order.Order, error) {
	value.ID = int64(len(r.orders) + 1)
	r.orders = append(r.orders, value)
	return value, nil
}

func (r *fakeRepository) GetByID(_ context.Context, id int64) (order.Order, error) {
	for _, value := range r.orders {
		if value.ID == id {
			return value, nil
		}
	}
	return order.Order{}, order.ErrOrderNotFound
}

func (r *fakeRepository) List(_ context.Context) ([]order.Order, error) {
	return r.orders, nil
}

func (r *fakeRepository) Update(_ context.Context, value order.Order) (order.Order, error) {
	for i, old := range r.orders {
		if old.ID == value.ID {
			r.orders[i] = value
			return value, nil
		}
	}
	return order.Order{}, order.ErrOrderNotFound
}

func TestCreate(t *testing.T) {
	tests := []struct {
		name  string
		items []order.Item
		total int64
		valid bool
	}{
		{"одна позиция", []order.Item{{Name: "Чай", Quantity: 2, Price: 100}}, 200, true},
		{"несколько позиций", []order.Item{{Name: "Чай", Quantity: 2, Price: 100}, {Name: "Кофе", Quantity: 3, Price: 50}}, 350, true},
		{"nil", nil, 0, false},
		{"пустой список", []order.Item{}, 0, false},
		{"пустое название", []order.Item{{Name: "", Quantity: 1, Price: 100}}, 0, false},
		{"пробельное название", []order.Item{{Name: " \t", Quantity: 1, Price: 100}}, 0, false},
		{"нулевое количество", []order.Item{{Name: "Чай", Quantity: 0, Price: 100}}, 0, false},
		{"отрицательное количество", []order.Item{{Name: "Чай", Quantity: -1, Price: 100}}, 0, false},
		{"нулевая цена", []order.Item{{Name: "Чай", Quantity: 1, Price: 0}}, 0, false},
		{"отрицательная цена", []order.Item{{Name: "Чай", Quantity: 1, Price: -100}}, 0, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &fakeRepository{}
			s := service.NewOrderService(repo)
			value, err := s.Create(context.Background(), order.CreateOrderInput{Items: tt.items})
			if !tt.valid {
				require.ErrorIs(t, err, order.ErrInvalidOrder)
				assert.Empty(t, repo.orders)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, int64(1), value.ID)
			assert.Equal(t, tt.items, value.Items)
			assert.Equal(t, tt.total, value.TotalPrice)
			assert.Equal(t, order.StatusNew, value.Status)
			assert.False(t, value.CreatedAt.IsZero())
			require.Len(t, value.History, 1)
			assert.Equal(t, order.StatusNew, value.History[0].Status)
			assert.True(t, value.CreatedAt.Equal(value.History[0].ChangedAt))
			assert.Equal(t, []order.Order{value}, repo.orders)
		})
	}
}

func TestGetAndList(t *testing.T) {
	ctx := context.Background()
	repo := &fakeRepository{}
	s := service.NewOrderService(repo)
	values, err := s.List(ctx)
	require.NoError(t, err)
	assert.Empty(t, values)

	repo.orders = []order.Order{{ID: 1, Status: order.StatusNew}, {ID: 2, Status: order.StatusProcessing}}
	for _, tt := range []struct {
		name string
		id   int64
		err  error
	}{
		{"существующий", 1, nil},
		{"второй заказ", 2, nil},
		{"отсутствующий", 99, order.ErrOrderNotFound},
		{"нулевой ID", 0, order.ErrInvalidOrder},
		{"отрицательный ID", -1, order.ErrInvalidOrder},
	} {
		t.Run(tt.name, func(t *testing.T) {
			value, err := s.GetByID(ctx, tt.id)
			require.ErrorIs(t, err, tt.err)
			if tt.err == nil {
				assert.Equal(t, repo.orders[tt.id-1], value)
			}
		})
	}
	values, err = s.List(ctx)
	require.NoError(t, err)
	assert.Equal(t, []order.Order{{ID: 1, Status: order.StatusNew}, {ID: 2, Status: order.StatusProcessing}}, values)
}

func TestUpdateStatus(t *testing.T) {
	tests := []struct {
		name     string
		id       int64
		from, to order.Status
		err      error
	}{
		{"new -> processing", 1, order.StatusNew, order.StatusProcessing, nil},
		{"new -> cancelled", 1, order.StatusNew, order.StatusCancelled, nil},
		{"new -> completed", 1, order.StatusNew, order.StatusCompleted, order.ErrInvalidStatusTransition},
		{"new -> new", 1, order.StatusNew, order.StatusNew, order.ErrInvalidStatusTransition},
		{"processing -> completed", 1, order.StatusProcessing, order.StatusCompleted, nil},
		{"processing -> cancelled", 1, order.StatusProcessing, order.StatusCancelled, nil},
		{"processing -> new", 1, order.StatusProcessing, order.StatusNew, order.ErrInvalidStatusTransition},
		{"processing -> processing", 1, order.StatusProcessing, order.StatusProcessing, order.ErrInvalidStatusTransition},
		{"completed -> new", 1, order.StatusCompleted, order.StatusNew, order.ErrInvalidStatusTransition},
		{"completed -> processing", 1, order.StatusCompleted, order.StatusProcessing, order.ErrInvalidStatusTransition},
		{"completed -> completed", 1, order.StatusCompleted, order.StatusCompleted, order.ErrInvalidStatusTransition},
		{"completed -> cancelled", 1, order.StatusCompleted, order.StatusCancelled, order.ErrInvalidStatusTransition},
		{"cancelled -> new", 1, order.StatusCancelled, order.StatusNew, order.ErrInvalidStatusTransition},
		{"cancelled -> processing", 1, order.StatusCancelled, order.StatusProcessing, order.ErrInvalidStatusTransition},
		{"cancelled -> completed", 1, order.StatusCancelled, order.StatusCompleted, order.ErrInvalidStatusTransition},
		{"cancelled -> cancelled", 1, order.StatusCancelled, order.StatusCancelled, order.ErrInvalidStatusTransition},
		{"пустой статус", 1, order.StatusNew, "", order.ErrInvalidStatus},
		{"неизвестный статус", 1, order.StatusNew, "unknown", order.ErrInvalidStatus},
		{"нулевой ID", 0, order.StatusNew, order.StatusProcessing, order.ErrInvalidOrder},
		{"отрицательный ID", -1, order.StatusNew, order.StatusProcessing, order.ErrInvalidOrder},
		{"нет заказа", 99, order.StatusNew, order.StatusProcessing, order.ErrOrderNotFound},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			before := order.Order{ID: 1, Status: tt.from, History: []order.StatusChange{{Status: tt.from}}}
			repo := &fakeRepository{orders: []order.Order{before}}
			s := service.NewOrderService(repo)
			value, err := s.UpdateStatus(context.Background(), tt.id, tt.to)
			require.ErrorIs(t, err, tt.err)
			if tt.err != nil {
				assert.Equal(t, tt.from, repo.orders[0].Status)
				assert.Equal(t, []order.StatusChange{{Status: tt.from}}, repo.orders[0].History)
				return
			}
			assert.Equal(t, tt.to, value.Status)
			require.Len(t, value.History, 2)
			assert.Equal(t, order.StatusChange{Status: tt.from}, value.History[0])
			assert.Equal(t, tt.to, value.History[1].Status)
			assert.False(t, value.History[1].ChangedAt.IsZero())
			assert.Equal(t, value, repo.orders[0])
		})
	}
}

func TestHistory(t *testing.T) {
	ctx := context.Background()
	s := service.NewOrderService(&fakeRepository{})
	value, err := s.Create(ctx, order.CreateOrderInput{Items: []order.Item{{Name: "Чай", Quantity: 1, Price: 100}}})
	require.NoError(t, err)
	for _, status := range []order.Status{order.StatusProcessing, order.StatusCompleted} {
		_, err = s.UpdateStatus(ctx, value.ID, status)
		require.NoError(t, err)
	}
	history, err := s.History(ctx, value.ID)
	require.NoError(t, err)
	require.Len(t, history, 3)
	for i, status := range []order.Status{order.StatusNew, order.StatusProcessing, order.StatusCompleted} {
		assert.Equal(t, status, history[i].Status)
		if i > 0 {
			assert.False(t, history[i].ChangedAt.Before(history[i-1].ChangedAt))
		}
	}
	for _, tt := range []struct {
		id  int64
		err error
	}{
		{0, order.ErrInvalidOrder},
		{-1, order.ErrInvalidOrder},
		{99, order.ErrOrderNotFound},
	} {
		_, err := s.History(ctx, tt.id)
		require.ErrorIs(t, err, tt.err)
	}
}
