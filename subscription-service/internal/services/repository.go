package services

import (
	"fmt"
	"math/rand"
	"sync"
	"time"

	"subscription-service/internal/models"
)

type SubscriptionRepository struct {
	mu            sync.RWMutex
	subscriptions map[string]models.Subscription
}

func NewSubscriptionRepository() *SubscriptionRepository {
	return &SubscriptionRepository{
		subscriptions: make(map[string]models.Subscription),
	}
}

func (r *SubscriptionRepository) Create(userID, plan string) models.Subscription {
	r.mu.Lock()
	defer r.mu.Unlock()

	sub := models.Subscription{
		ID:        fmt.Sprintf("sub_%d", rand.Int()),
		UserID:    userID,
		Plan:      plan,
		StartDate: time.Now(),
		EndDate:   time.Now().AddDate(1, 0, 0),
	}

	r.subscriptions[sub.ID] = sub
	return sub
}

func (r *SubscriptionRepository) GetAll() []models.Subscription {
	r.mu.RLock()
	defer r.mu.RUnlock()

	subs := make([]models.Subscription, 0, len(r.subscriptions))
	for _, sub := range r.subscriptions {
		subs = append(subs, sub)
	}
	return subs
}

func (r *SubscriptionRepository) GetByID(id string) (models.Subscription, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	sub, exists := r.subscriptions[id]
	return sub, exists
}

func (r *SubscriptionRepository) Update(id string, userID, plan string) (models.Subscription, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()

	sub, exists := r.subscriptions[id]
	if !exists {
		return models.Subscription{}, false
	}

	sub.UserID = userID
	sub.Plan = plan
	r.subscriptions[id] = sub
	return sub, true
}

func (r *SubscriptionRepository) Delete(id string) (models.Subscription, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()

	sub, exists := r.subscriptions[id]
	if !exists {
		return models.Subscription{}, false
	}

	delete(r.subscriptions, id)
	return sub, true
}

func (r *SubscriptionRepository) Count() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.subscriptions)
}
