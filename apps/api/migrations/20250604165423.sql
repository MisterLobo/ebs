-- Modify "event_subscriptions" table
ALTER TABLE "event_subscriptions" ADD COLUMN "tenant_id" uuid NULL;
-- Modify "users" table
ALTER TABLE "users" ADD COLUMN "stripe_subscription_id" text NULL;
