package subscription

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/hngprojects/telex_be/internal/models"
	"github.com/stripe/stripe-go/v72"
	"github.com/stripe/stripe-go/v72/sub"
	"gorm.io/gorm"
)

func CreateSubscription(req *models.CreateSubscriptionRequest, db *gorm.DB) (*gin.H, int, error) {
	var subscriptionPlan models.SubscriptionPlan

	if err := db.Where("name = ?", req.PlanName).First(&subscriptionPlan).Error; err != nil {
		return nil, http.StatusNotFound, fmt.Errorf("subscription plan not found: %v", err)
	}

	oneMonthLater := time.Now().AddDate(0, 1, 0).Unix()

	params := &stripe.SubscriptionParams{
		Customer: stripe.String(req.UserID),
		Items: []*stripe.SubscriptionItemsParams{
			{
				Price: stripe.String(subscriptionPlan.Name),
			},
		},
		TrialEnd: stripe.Int64(oneMonthLater),
	}
	Usersub, err := sub.New(params)
	if err != nil {
		return nil, http.StatusInternalServerError, err
	}

	if err := db.Model(&models.User{}).Where("id = ?", req.UserID).Update("subscription_plan_id", subscriptionPlan.ID).Error; err != nil {
		return nil, http.StatusInternalServerError, fmt.Errorf("failed to update user with subscription plan: %v", err)
	}

	responseData := gin.H{
		"subscription_id": Usersub.ID,
		"status":          Usersub.Status,
		"plan":            subscriptionPlan.Name,
		"start_date":      time.Unix(Usersub.CurrentPeriodStart, 0),
		"end_date":        time.Unix(oneMonthLater, 0),
	}

	return &responseData, http.StatusOK, nil
}

func ListSubscriptions(customerID string, db *gorm.DB) (*gin.H, int, error) {
	params := &stripe.SubscriptionListParams{
		Customer: customerID,
	}
	i := sub.List(params)

	var subscriptions []*stripe.Subscription
	for i.Next() {
		subscriptions = append(subscriptions, i.Subscription())
	}

	if err := i.Err(); err != nil {
		return nil, http.StatusInternalServerError, err
	}

	responseData := gin.H{
		"subscriptions": subscriptions,
	}

	return &responseData, http.StatusOK, nil
}

func ModifySubscription(req *models.ModifySubscriptionRequest, db *gorm.DB) (*gin.H, int, error) {
	var subscriptionPlan models.SubscriptionPlan
	if err := db.Where("id = ?", req.PlanID).First(&subscriptionPlan).Error; err != nil {
		return nil, http.StatusNotFound, fmt.Errorf("subscription plan not found: %v", err)
	}

	params := &stripe.SubscriptionParams{
		Items: []*stripe.SubscriptionItemsParams{
			{
				ID:    stripe.String(req.UserID),
				Price: stripe.String(subscriptionPlan.Name),
			},
		},
	}

	subscription, err := sub.Update(req.UserID, params)
	if err != nil {
		return nil, http.StatusInternalServerError, err
	}

	responseData := gin.H{
		"subscription_id": subscription.ID,
		"status":          subscription.Status,
		"plan":            subscriptionPlan.Name,
	}

	return &responseData, http.StatusOK, nil
}

func DeleteSubscription(userId string, db *gorm.DB) (int, error) {
	var user *models.User
	if err := db.First(&user, "id = ?", userId).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return http.StatusNotFound, fmt.Errorf("user not found")
		}
		return http.StatusInternalServerError, fmt.Errorf("error finding user: %w", err)
	}
	_, err := sub.Cancel(strconv.FormatUint(uint64(user.SubscriptionPlan.ID), 10), nil)
	if err != nil {
		return http.StatusInternalServerError, fmt.Errorf("error cancelling subscription: %w", err)
	}
	if err := db.Model(&user).Association("SubscriptionPlan").Clear(); err != nil {
		return http.StatusInternalServerError, fmt.Errorf("error removing subscription plan: %w", err)
	}
	return http.StatusOK, nil
}
