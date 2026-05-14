package subscriptionadminconnect

import (
	"encoding/json"
	"time"

	"github.com/anthropics/agentsmesh/backend/internal/domain/billing"
	billingservice "github.com/anthropics/agentsmesh/backend/internal/service/billing"
	billingv1 "github.com/anthropics/agentsmesh/proto/gen/go/billing/v1"
)

func toProtoAdminSubscription(sub *billing.Subscription, seatUsage *billingservice.SeatUsage) *billingv1.AdminSubscription {
	if sub == nil {
		return nil
	}
	out := &billingv1.AdminSubscription{
		Subscription:    toProtoSubscriptionEntity(sub),
		HasStripe:       sub.StripeSubscriptionID != nil,
		HasAlipay:       sub.AlipayAgreementNo != nil,
		HasWechat:       sub.WeChatContractID != nil,
		HasLemonsqueezy: sub.LemonSqueezySubscriptionID != nil,
	}
	if seatUsage != nil {
		out.SeatUsage = toProtoSeatUsage(seatUsage)
	}
	if sub.CustomQuotas != nil {
		if data, err := json.Marshal(sub.CustomQuotas); err == nil {
			s := string(data)
			out.CustomQuotasJson = &s
		}
	}
	return out
}

func toProtoSubscriptionEntity(sub *billing.Subscription) *billingv1.AdminSubscriptionEntity {
	if sub == nil {
		return nil
	}
	out := &billingv1.AdminSubscriptionEntity{
		Id:                 sub.ID,
		OrganizationId:     sub.OrganizationID,
		PlanId:             sub.PlanID,
		Status:             sub.Status,
		BillingCycle:       sub.BillingCycle,
		CurrentPeriodStart: sub.CurrentPeriodStart.Format(time.RFC3339),
		CurrentPeriodEnd:   sub.CurrentPeriodEnd.Format(time.RFC3339),
		AutoRenew:          sub.AutoRenew,
		SeatCount:          int32(sub.SeatCount),
		CancelAtPeriodEnd:  sub.CancelAtPeriodEnd,
		CreatedAt:          sub.CreatedAt.Format(time.RFC3339),
		UpdatedAt:          sub.UpdatedAt.Format(time.RFC3339),
	}
	if sub.PaymentProvider != nil {
		v := *sub.PaymentProvider
		out.PaymentProvider = &v
	}
	if sub.PaymentMethod != nil {
		v := *sub.PaymentMethod
		out.PaymentMethod = &v
	}
	if sub.StripeCustomerID != nil {
		v := *sub.StripeCustomerID
		out.StripeCustomerId = &v
	}
	if sub.StripeSubscriptionID != nil {
		v := *sub.StripeSubscriptionID
		out.StripeSubscriptionId = &v
	}
	if sub.LemonSqueezyCustomerID != nil {
		v := *sub.LemonSqueezyCustomerID
		out.LemonsqueezyCustomerId = &v
	}
	if sub.LemonSqueezySubscriptionID != nil {
		v := *sub.LemonSqueezySubscriptionID
		out.LemonsqueezySubscriptionId = &v
	}
	if sub.CanceledAt != nil {
		v := sub.CanceledAt.Format(time.RFC3339)
		out.CanceledAt = &v
	}
	if sub.FrozenAt != nil {
		v := sub.FrozenAt.Format(time.RFC3339)
		out.FrozenAt = &v
	}
	if sub.DowngradeToPlan != nil {
		v := *sub.DowngradeToPlan
		out.DowngradeToPlan = &v
	}
	if sub.NextBillingCycle != nil {
		v := *sub.NextBillingCycle
		out.NextBillingCycle = &v
	}
	if sub.Plan != nil {
		out.Plan = toProtoPlan(sub.Plan)
	}
	return out
}

func toProtoPlan(p *billing.SubscriptionPlan) *billingv1.AdminSubscriptionPlan {
	if p == nil {
		return nil
	}
	return &billingv1.AdminSubscriptionPlan{
		Id:                  p.ID,
		Name:                p.Name,
		DisplayName:         p.DisplayName,
		PricePerSeatMonthly: p.PricePerSeatMonthly,
		PricePerSeatYearly:  p.PricePerSeatYearly,
		IncludedPodMinutes:  int32(p.IncludedPodMinutes),
		PricePerExtraMinute: p.PricePerExtraMinute,
		MaxUsers:            int32(p.MaxUsers),
		MaxRunners:          int32(p.MaxRunners),
		MaxConcurrentPods:   int32(p.MaxConcurrentPods),
		MaxRepositories:     int32(p.MaxRepositories),
		IsActive:            p.IsActive,
		CreatedAt:           p.CreatedAt.Format(time.RFC3339),
	}
}

func toProtoSeatUsage(u *billingservice.SeatUsage) *billingv1.AdminSeatUsage {
	return &billingv1.AdminSeatUsage{
		TotalSeats:     int32(u.TotalSeats),
		UsedSeats:      int32(u.UsedSeats),
		AvailableSeats: int32(u.AvailableSeats),
		MaxSeats:       int32(u.MaxSeats),
		CanAddSeats:    u.CanAddSeats,
	}
}
