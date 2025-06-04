-- Create "settings" table
CREATE TABLE "settings" (
  "id" uuid NOT NULL DEFAULT gen_random_uuid(),
  "setting_key" text NULL,
  "setting_value" jsonb NULL,
  "group" text NULL,
  "created_at" timestamptz NULL,
  "updated_at" timestamptz NULL,
  "deleted_at" timestamptz NULL,
  PRIMARY KEY ("id")
);
-- Create index "idx_settings_deleted_at" to table: "settings"
CREATE INDEX "idx_settings_deleted_at" ON "settings" ("deleted_at");
-- Create index "name" to table: "settings"
CREATE UNIQUE INDEX "name" ON "settings" ("setting_key", "group");
-- Create "users" table
CREATE TABLE "users" (
  "id" bigserial NOT NULL,
  "name" text NULL,
  "email" text NULL,
  "role" text NULL,
  "uid" text NULL,
  "active_org" bigint NULL,
  "email_verified" boolean NULL,
  "phone_verified" boolean NULL,
  "verified_at" timestamptz NULL,
  "stripe_account_id" text NULL,
  "stripe_customer_id" text NULL,
  "metadata" jsonb NULL,
  "last_active" timestamptz NULL,
  "tenant_id" uuid NULL DEFAULT gen_random_uuid(),
  "created_at" timestamptz NULL,
  "updated_at" timestamptz NULL,
  "deleted_at" timestamptz NULL,
  PRIMARY KEY ("id")
);
-- Create index "idx_users_deleted_at" to table: "users"
CREATE INDEX "idx_users_deleted_at" ON "users" ("deleted_at");
-- Create index "idx_users_email" to table: "users"
CREATE UNIQUE INDEX "idx_users_email" ON "users" ("email");
-- Create index "idx_users_tenant_id" to table: "users"
CREATE UNIQUE INDEX "idx_users_tenant_id" ON "users" ("tenant_id");
-- Create "job_tasks" table
CREATE TABLE "job_tasks" (
  "id" uuid NOT NULL DEFAULT gen_random_uuid(),
  "name" text NULL,
  "job_type" text NULL,
  "runs_at" timestamptz NULL,
  "handler_params" jsonb NULL,
  "payload_id" text NULL,
  "payload" jsonb NULL,
  "source" text NULL,
  "source_type" text NULL,
  "status" text NULL DEFAULT 'pending',
  "topic" text NULL,
  PRIMARY KEY ("id")
);
-- Create "trail_logs" table
CREATE TABLE "trail_logs" (
  "id" uuid NOT NULL DEFAULT gen_random_uuid(),
  "type" text NULL,
  "initiator" text NULL,
  "group" text NULL,
  PRIMARY KEY ("id")
);
-- Create "organizations" table
CREATE TABLE "organizations" (
  "id" bigserial NOT NULL,
  "name" text NULL,
  "about" text NULL,
  "country" text NULL,
  "owner_id" bigint NULL,
  "type" text NULL DEFAULT 'standard',
  "stripe_account_id" text NULL,
  "metadata" jsonb NULL,
  "contact_email" text NULL,
  "connect_onboarding_url" text NULL,
  "status" text NULL DEFAULT 'pending',
  "verified" boolean NULL DEFAULT false,
  "payment_verified" boolean NULL DEFAULT false,
  "slug" text NULL,
  "tenant_id" uuid NULL,
  "created_at" timestamptz NULL,
  "updated_at" timestamptz NULL,
  "deleted_at" timestamptz NULL,
  PRIMARY KEY ("id"),
  CONSTRAINT "fk_users_organizations" FOREIGN KEY ("owner_id") REFERENCES "users" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION
);
-- Create index "idx_organizations_deleted_at" to table: "organizations"
CREATE INDEX "idx_organizations_deleted_at" ON "organizations" ("deleted_at");
-- Create index "slugid" to table: "organizations"
CREATE UNIQUE INDEX "slugid" ON "organizations" ("id", "slug");
-- Create "events" table
CREATE TABLE "events" (
  "id" bigserial NOT NULL,
  "title" text NULL,
  "name" text NULL,
  "about" text NULL,
  "type" text NULL DEFAULT 'general',
  "location" text NULL,
  "date_time" timestamptz NULL,
  "status" text NULL DEFAULT 'draft',
  "organizer_id" bigint NULL,
  "seats" bigint NULL,
  "created_by" bigint NULL,
  "mode" text NULL DEFAULT 'default',
  "opens_at" timestamptz NULL,
  "deadline" timestamptz NULL,
  "metadata" jsonb NULL,
  "identifier" text NULL,
  "tenant_id" uuid NULL,
  "created_at" timestamptz NULL,
  "updated_at" timestamptz NULL,
  "deleted_at" timestamptz NULL,
  PRIMARY KEY ("id"),
  CONSTRAINT "fk_events_creator" FOREIGN KEY ("created_by") REFERENCES "users" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION,
  CONSTRAINT "fk_organizations_events" FOREIGN KEY ("organizer_id") REFERENCES "organizations" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION
);
-- Create index "idx_events_deleted_at" to table: "events"
CREATE INDEX "idx_events_deleted_at" ON "events" ("deleted_at");
-- Create "tickets" table
CREATE TABLE "tickets" (
  "id" bigserial NOT NULL,
  "type" text NULL,
  "tier" text NULL,
  "status" text NULL DEFAULT 'draft',
  "price" numeric NULL,
  "currency" text NULL,
  "limited" boolean NULL,
  "limit" bigint NULL,
  "event_id" bigint NULL,
  "stripe_price_id" text NULL,
  "metadata" jsonb NULL,
  "identifier" text NULL,
  "tenant_id" uuid NULL,
  "created_at" timestamptz NULL,
  "updated_at" timestamptz NULL,
  "deleted_at" timestamptz NULL,
  PRIMARY KEY ("id"),
  CONSTRAINT "fk_events_tickets" FOREIGN KEY ("event_id") REFERENCES "events" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION
);
-- Create index "idx_tickets_deleted_at" to table: "tickets"
CREATE INDEX "idx_tickets_deleted_at" ON "tickets" ("deleted_at");
-- Create "transactions" table
CREATE TABLE "transactions" (
  "id" uuid NOT NULL DEFAULT gen_random_uuid(),
  "currency" text NULL,
  "amount" numeric NULL,
  "source_name" text NULL,
  "source_value" text NULL,
  "reference_id" text NULL,
  "status" text NULL,
  "metadata" jsonb NULL,
  "checkout_session_id" text NULL,
  "payment_intent_id" text NULL,
  "coupon_code" text NULL,
  "tenant_id" uuid NULL,
  "created_at" timestamptz NULL,
  "updated_at" timestamptz NULL,
  "deleted_at" timestamptz NULL,
  PRIMARY KEY ("id")
);
-- Create index "idx_transactions_deleted_at" to table: "transactions"
CREATE INDEX "idx_transactions_deleted_at" ON "transactions" ("deleted_at");
-- Create "bookings" table
CREATE TABLE "bookings" (
  "id" bigserial NOT NULL,
  "ticket_id" bigint NULL,
  "status" text NULL,
  "qty" smallint NULL,
  "unit_price" numeric NULL,
  "subtotal" numeric NULL,
  "currency" text NULL,
  "user_id" bigint NULL,
  "event_id" bigint NULL,
  "metadata" jsonb NULL,
  "checkout_session_id" text NULL,
  "payment_intent_id" text NULL,
  "transaction_id" uuid NULL,
  "slots_wanted" bigint NULL,
  "slots_taken" bigint NULL,
  "tenant_id" uuid NULL,
  "created_at" timestamptz NULL,
  "updated_at" timestamptz NULL,
  "deleted_at" timestamptz NULL,
  PRIMARY KEY ("id"),
  CONSTRAINT "fk_bookings_event" FOREIGN KEY ("event_id") REFERENCES "events" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION,
  CONSTRAINT "fk_bookings_ticket" FOREIGN KEY ("ticket_id") REFERENCES "tickets" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION,
  CONSTRAINT "fk_bookings_transaction" FOREIGN KEY ("transaction_id") REFERENCES "transactions" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION,
  CONSTRAINT "fk_users_bookings" FOREIGN KEY ("user_id") REFERENCES "users" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION
);
-- Create index "idx_bookings_deleted_at" to table: "bookings"
CREATE INDEX "idx_bookings_deleted_at" ON "bookings" ("deleted_at");
-- Create "reservations" table
CREATE TABLE "reservations" (
  "id" bigserial NOT NULL,
  "ticket_id" bigint NULL,
  "booking_id" bigint NULL,
  "valid_until" timestamptz NULL,
  "share_url" text NULL,
  "status" text NULL DEFAULT 'pending',
  "tenant_id" uuid NULL,
  "created_at" timestamptz NULL,
  "updated_at" timestamptz NULL,
  "deleted_at" timestamptz NULL,
  PRIMARY KEY ("id"),
  CONSTRAINT "fk_bookings_reservations" FOREIGN KEY ("booking_id") REFERENCES "bookings" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION,
  CONSTRAINT "fk_reservations_ticket" FOREIGN KEY ("ticket_id") REFERENCES "tickets" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION
);
-- Create index "idx_reservations_deleted_at" to table: "reservations"
CREATE INDEX "idx_reservations_deleted_at" ON "reservations" ("deleted_at");
-- Create "admissions" table
CREATE TABLE "admissions" (
  "id" bigserial NOT NULL,
  "by" bigint NULL,
  "reservation_id" bigint NULL,
  "type" text NULL,
  "status" text NULL,
  "tenant_id" uuid NULL,
  "created_at" timestamptz NULL,
  "updated_at" timestamptz NULL,
  "deleted_at" timestamptz NULL,
  PRIMARY KEY ("id"),
  CONSTRAINT "fk_admissions_admitted_by" FOREIGN KEY ("by") REFERENCES "users" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION,
  CONSTRAINT "fk_admissions_reservation" FOREIGN KEY ("reservation_id") REFERENCES "reservations" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION
);
-- Create index "idx_admissions_deleted_at" to table: "admissions"
CREATE INDEX "idx_admissions_deleted_at" ON "admissions" ("deleted_at");
-- Create "event_subscriptions" table
CREATE TABLE "event_subscriptions" (
  "id" bigserial NOT NULL,
  "event_id" bigint NOT NULL,
  "subscriber_id" bigint NOT NULL,
  "status" text NULL DEFAULT 'notify',
  "created_at" timestamptz NULL,
  "updated_at" timestamptz NULL,
  "deleted_at" timestamptz NULL,
  PRIMARY KEY ("id", "event_id", "subscriber_id"),
  CONSTRAINT "fk_event_subscriptions_event" FOREIGN KEY ("subscriber_id") REFERENCES "events" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION,
  CONSTRAINT "fk_event_subscriptions_user" FOREIGN KEY ("subscriber_id") REFERENCES "users" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION
);
-- Create index "idx_event_subscriptions_deleted_at" to table: "event_subscriptions"
CREATE INDEX "idx_event_subscriptions_deleted_at" ON "event_subscriptions" ("deleted_at");
-- Create "ratings" table
CREATE TABLE "ratings" (
  "id" bigserial NOT NULL,
  "organization_id" bigint NULL,
  "by_user" bigint NULL,
  "value" bigint NULL,
  "created_at" timestamptz NULL,
  "updated_at" timestamptz NULL,
  "deleted_at" timestamptz NULL,
  PRIMARY KEY ("id"),
  CONSTRAINT "fk_ratings_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION,
  CONSTRAINT "fk_ratings_user" FOREIGN KEY ("by_user") REFERENCES "users" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION
);
-- Create index "idx_ratings_deleted_at" to table: "ratings"
CREATE INDEX "idx_ratings_deleted_at" ON "ratings" ("deleted_at");
-- Create "permissions" table
CREATE TABLE "permissions" (
  "name" text NOT NULL,
  PRIMARY KEY ("name")
);
-- Create "roles" table
CREATE TABLE "roles" (
  "name" text NOT NULL,
  PRIMARY KEY ("name")
);
-- Create "role_permissions" table
CREATE TABLE "role_permissions" (
  "role_name" text NOT NULL,
  "permission_name" text NOT NULL,
  PRIMARY KEY ("role_name", "permission_name"),
  CONSTRAINT "fk_role_permissions_permission" FOREIGN KEY ("permission_name") REFERENCES "permissions" ("name") ON UPDATE NO ACTION ON DELETE NO ACTION,
  CONSTRAINT "fk_role_permissions_role" FOREIGN KEY ("role_name") REFERENCES "roles" ("name") ON UPDATE NO ACTION ON DELETE NO ACTION
);
-- Create "teams" table
CREATE TABLE "teams" (
  "id" bigserial NOT NULL,
  "name" text NULL,
  "owner_id" bigint NULL,
  "organization_id" bigint NULL,
  "status" text NULL,
  "tenant_id" uuid NULL,
  "created_at" timestamptz NULL,
  "updated_at" timestamptz NULL,
  "deleted_at" timestamptz NULL,
  PRIMARY KEY ("id"),
  CONSTRAINT "fk_teams_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION,
  CONSTRAINT "fk_teams_owner" FOREIGN KEY ("owner_id") REFERENCES "users" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION
);
-- Create index "idx_teams_deleted_at" to table: "teams"
CREATE INDEX "idx_teams_deleted_at" ON "teams" ("deleted_at");
-- Create "team_members" table
CREATE TABLE "team_members" (
  "id" bigserial NOT NULL,
  "team_id" bigint NOT NULL,
  "user_id" bigint NOT NULL,
  "role" text NULL,
  "status" text NULL,
  "created_at" timestamptz NULL,
  "updated_at" timestamptz NULL,
  "deleted_at" timestamptz NULL,
  PRIMARY KEY ("id", "team_id", "user_id"),
  CONSTRAINT "fk_team_members_team" FOREIGN KEY ("team_id") REFERENCES "teams" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION,
  CONSTRAINT "fk_team_members_user" FOREIGN KEY ("user_id") REFERENCES "users" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION
);
-- Create index "idx_team_members_deleted_at" to table: "team_members"
CREATE INDEX "idx_team_members_deleted_at" ON "team_members" ("deleted_at");
