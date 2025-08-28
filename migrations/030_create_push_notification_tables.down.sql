-- Drop triggers
DROP TRIGGER IF EXISTS update_push_subscriptions_updated_at ON push_subscriptions;
DROP TRIGGER IF EXISTS update_push_notifications_updated_at ON push_notifications;
DROP TRIGGER IF EXISTS update_push_deliveries_updated_at ON push_deliveries;
DROP TRIGGER IF EXISTS update_push_templates_updated_at ON push_templates;
DROP TRIGGER IF EXISTS update_notification_preferences_updated_at ON notification_preferences;

-- Drop function
DROP FUNCTION IF EXISTS update_push_updated_at_column();

-- Drop tables (in reverse order due to foreign key constraints)
DROP TABLE IF EXISTS notification_preferences;
DROP TABLE IF EXISTS push_templates;
DROP TABLE IF EXISTS push_deliveries;
DROP TABLE IF EXISTS push_notifications;
DROP TABLE IF EXISTS push_subscriptions;