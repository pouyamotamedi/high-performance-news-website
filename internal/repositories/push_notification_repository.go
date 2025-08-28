package repositories

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/lib/pq"
	"your-project/internal/models"
)

type PushNotificationRepository struct {
	db *sql.DB
}

func NewPushNotificationRepository(db *sql.DB) *PushNotificationRepository {
	return &PushNotificationRepository{db: db}
}

// Subscription management
func (r *PushNotificationRepository) CreateSubscription(subscription *models.PushSubscription) error {
	query := `
		INSERT INTO push_subscriptions (user_id, endpoint, p256dh, auth, user_agent, ip_address, is_active, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, NOW(), NOW())
		ON CONFLICT (endpoint) DO UPDATE SET
			user_id = EXCLUDED.user_id,
			p256dh = EXCLUDED.p256dh,
			auth = EXCLUDED.auth,
			user_agent = EXCLUDED.user_agent,
			ip_address = EXCLUDED.ip_address,
			is_active = EXCLUDED.is_active,
			updated_at = NOW()
		RETURNING id, created_at, updated_at`

	return r.db.QueryRow(query,
		subscription.UserID,
		subscription.Endpoint,
		subscription.P256DH,
		subscription.Auth,
		subscription.UserAgent,
		subscription.IPAddress,
		subscription.IsActive,
	).Scan(&subscription.ID, &subscription.CreatedAt, &subscription.UpdatedAt)
}

func (r *PushNotificationRepository) GetSubscriptionByEndpoint(endpoint string) (*models.PushSubscription, error) {
	subscription := &models.PushSubscription{}
	query := `
		SELECT id, user_id, endpoint, p256dh, auth, user_agent, ip_address, is_active, created_at, updated_at
		FROM push_subscriptions
		WHERE endpoint = $1`

	err := r.db.QueryRow(query, endpoint).Scan(
		&subscription.ID,
		&subscription.UserID,
		&subscription.Endpoint,
		&subscription.P256DH,
		&subscription.Auth,
		&subscription.UserAgent,
		&subscription.IPAddress,
		&subscription.IsActive,
		&subscription.CreatedAt,
		&subscription.UpdatedAt,
	)

	if err != nil {
		return nil, err
	}
	return subscription, nil
}

func (r *PushNotificationRepository) GetActiveSubscriptions() ([]*models.PushSubscription, error) {
	query := `
		SELECT id, user_id, endpoint, p256dh, auth, user_agent, ip_address, is_active, created_at, updated_at
		FROM push_subscriptions
		WHERE is_active = true
		ORDER BY created_at DESC`

	rows, err := r.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var subscriptions []*models.PushSubscription
	for rows.Next() {
		subscription := &models.PushSubscription{}
		err := rows.Scan(
			&subscription.ID,
			&subscription.UserID,
			&subscription.Endpoint,
			&subscription.P256DH,
			&subscription.Auth,
			&subscription.UserAgent,
			&subscription.IPAddress,
			&subscription.IsActive,
			&subscription.CreatedAt,
			&subscription.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		subscriptions = append(subscriptions, subscription)
	}

	return subscriptions, rows.Err()
}

func (r *PushNotificationRepository) GetTargetedSubscriptions(targetType, targetValue string) ([]*models.PushSubscription, error) {
	var query string
	var args []interface{}

	switch targetType {
	case models.TargetTypeAll:
		query = `
			SELECT DISTINCT ps.id, ps.user_id, ps.endpoint, ps.p256dh, ps.auth, ps.user_agent, ps.ip_address, ps.is_active, ps.created_at, ps.updated_at
			FROM push_subscriptions ps
			WHERE ps.is_active = true`
		args = []interface{}{}

	case models.TargetTypeCategory:
		query = `
			SELECT DISTINCT ps.id, ps.user_id, ps.endpoint, ps.p256dh, ps.auth, ps.user_agent, ps.ip_address, ps.is_active, ps.created_at, ps.updated_at
			FROM push_subscriptions ps
			LEFT JOIN notification_preferences np ON ps.id = np.subscription_id
			WHERE ps.is_active = true
			AND (np.category_updates = true OR np.category_updates IS NULL)
			AND (np.preferred_categories IS NULL OR np.preferred_categories @> $1::jsonb)`
		args = []interface{}{fmt.Sprintf(`[%s]`, targetValue)}

	case models.TargetTypeTag:
		query = `
			SELECT DISTINCT ps.id, ps.user_id, ps.endpoint, ps.p256dh, ps.auth, ps.user_agent, ps.ip_address, ps.is_active, ps.created_at, ps.updated_at
			FROM push_subscriptions ps
			LEFT JOIN notification_preferences np ON ps.id = np.subscription_id
			WHERE ps.is_active = true
			AND (np.tag_updates = true)
			AND (np.preferred_tags IS NULL OR np.preferred_tags @> $1::jsonb)`
		args = []interface{}{fmt.Sprintf(`[%s]`, targetValue)}

	case models.TargetTypeUser:
		query = `
			SELECT ps.id, ps.user_id, ps.endpoint, ps.p256dh, ps.auth, ps.user_agent, ps.ip_address, ps.is_active, ps.created_at, ps.updated_at
			FROM push_subscriptions ps
			WHERE ps.is_active = true AND ps.user_id = $1`
		args = []interface{}{targetValue}

	default:
		return nil, fmt.Errorf("invalid target type: %s", targetType)
	}

	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var subscriptions []*models.PushSubscription
	for rows.Next() {
		subscription := &models.PushSubscription{}
		err := rows.Scan(
			&subscription.ID,
			&subscription.UserID,
			&subscription.Endpoint,
			&subscription.P256DH,
			&subscription.Auth,
			&subscription.UserAgent,
			&subscription.IPAddress,
			&subscription.IsActive,
			&subscription.CreatedAt,
			&subscription.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		subscriptions = append(subscriptions, subscription)
	}

	return subscriptions, rows.Err()
}

func (r *PushNotificationRepository) DeactivateSubscription(endpoint string) error {
	query := `UPDATE push_subscriptions SET is_active = false, updated_at = NOW() WHERE endpoint = $1`
	_, err := r.db.Exec(query, endpoint)
	return err
}

// Notification management
func (r *PushNotificationRepository) CreateNotification(notification *models.PushNotification) error {
	dataJSON, err := json.Marshal(notification.Data)
	if err != nil {
		return err
	}

	query := `
		INSERT INTO push_notifications (title, body, icon, badge, image, url, data, target_type, target_value, scheduled_at, status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, NOW(), NOW())
		RETURNING id, created_at, updated_at`

	return r.db.QueryRow(query,
		notification.Title,
		notification.Body,
		notification.Icon,
		notification.Badge,
		notification.Image,
		notification.URL,
		dataJSON,
		notification.TargetType,
		notification.TargetValue,
		notification.ScheduledAt,
		notification.Status,
	).Scan(&notification.ID, &notification.CreatedAt, &notification.UpdatedAt)
}

func (r *PushNotificationRepository) GetNotificationByID(id uint64) (*models.PushNotification, error) {
	notification := &models.PushNotification{}
	var dataJSON []byte

	query := `
		SELECT id, title, body, icon, badge, image, url, data, target_type, target_value, 
		       scheduled_at, sent_at, status, total_sent, total_delivered, total_clicked, created_at, updated_at
		FROM push_notifications
		WHERE id = $1`

	err := r.db.QueryRow(query, id).Scan(
		&notification.ID,
		&notification.Title,
		&notification.Body,
		&notification.Icon,
		&notification.Badge,
		&notification.Image,
		&notification.URL,
		&dataJSON,
		&notification.TargetType,
		&notification.TargetValue,
		&notification.ScheduledAt,
		&notification.SentAt,
		&notification.Status,
		&notification.TotalSent,
		&notification.TotalDelivered,
		&notification.TotalClicked,
		&notification.CreatedAt,
		&notification.UpdatedAt,
	)

	if err != nil {
		return nil, err
	}

	if len(dataJSON) > 0 {
		err = json.Unmarshal(dataJSON, &notification.Data)
		if err != nil {
			return nil, err
		}
	}

	return notification, nil
}

func (r *PushNotificationRepository) GetPendingNotifications() ([]*models.PushNotification, error) {
	query := `
		SELECT id, title, body, icon, badge, image, url, data, target_type, target_value, 
		       scheduled_at, sent_at, status, total_sent, total_delivered, total_clicked, created_at, updated_at
		FROM push_notifications
		WHERE status = $1 AND (scheduled_at IS NULL OR scheduled_at <= NOW())
		ORDER BY created_at ASC`

	rows, err := r.db.Query(query, models.NotificationStatusPending)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var notifications []*models.PushNotification
	for rows.Next() {
		notification := &models.PushNotification{}
		var dataJSON []byte

		err := rows.Scan(
			&notification.ID,
			&notification.Title,
			&notification.Body,
			&notification.Icon,
			&notification.Badge,
			&notification.Image,
			&notification.URL,
			&dataJSON,
			&notification.TargetType,
			&notification.TargetValue,
			&notification.ScheduledAt,
			&notification.SentAt,
			&notification.Status,
			&notification.TotalSent,
			&notification.TotalDelivered,
			&notification.TotalClicked,
			&notification.CreatedAt,
			&notification.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}

		if len(dataJSON) > 0 {
			err = json.Unmarshal(dataJSON, &notification.Data)
			if err != nil {
				return nil, err
			}
		}

		notifications = append(notifications, notification)
	}

	return notifications, rows.Err()
}

func (r *PushNotificationRepository) UpdateNotificationStatus(id uint64, status string, totalSent int) error {
	query := `
		UPDATE push_notifications 
		SET status = $2, total_sent = $3, sent_at = CASE WHEN $2 = 'sent' THEN NOW() ELSE sent_at END, updated_at = NOW()
		WHERE id = $1`

	_, err := r.db.Exec(query, id, status, totalSent)
	return err
}

func (r *PushNotificationRepository) UpdateNotificationStats(id uint64, totalDelivered, totalClicked int) error {
	query := `
		UPDATE push_notifications 
		SET total_delivered = $2, total_clicked = $3, updated_at = NOW()
		WHERE id = $1`

	_, err := r.db.Exec(query, id, totalDelivered, totalClicked)
	return err
}

// Delivery tracking
func (r *PushNotificationRepository) CreateDelivery(delivery *models.PushDelivery) error {
	query := `
		INSERT INTO push_deliveries (notification_id, subscription_id, status, error_message, created_at, updated_at)
		VALUES ($1, $2, $3, $4, NOW(), NOW())
		RETURNING id, created_at, updated_at`

	return r.db.QueryRow(query,
		delivery.NotificationID,
		delivery.SubscriptionID,
		delivery.Status,
		delivery.ErrorMessage,
	).Scan(&delivery.ID, &delivery.CreatedAt, &delivery.UpdatedAt)
}

func (r *PushNotificationRepository) UpdateDeliveryStatus(id uint64, status string) error {
	var query string
	switch status {
	case models.DeliveryStatusDelivered:
		query = `UPDATE push_deliveries SET status = $2, delivered_at = NOW(), updated_at = NOW() WHERE id = $1`
	case models.DeliveryStatusClicked:
		query = `UPDATE push_deliveries SET status = $2, clicked_at = NOW(), updated_at = NOW() WHERE id = $1`
	default:
		query = `UPDATE push_deliveries SET status = $2, updated_at = NOW() WHERE id = $1`
	}

	_, err := r.db.Exec(query, id, status)
	return err
}

func (r *PushNotificationRepository) GetDeliveryStats(notificationID uint64) (delivered, clicked int, err error) {
	query := `
		SELECT 
			COUNT(CASE WHEN status IN ('delivered', 'clicked') THEN 1 END) as delivered,
			COUNT(CASE WHEN status = 'clicked' THEN 1 END) as clicked
		FROM push_deliveries
		WHERE notification_id = $1`

	err = r.db.QueryRow(query, notificationID).Scan(&delivered, &clicked)
	return delivered, clicked, err
}

// Template management
func (r *PushNotificationRepository) CreateTemplate(template *models.PushTemplate) error {
	variablesJSON, err := json.Marshal(template.Variables)
	if err != nil {
		return err
	}

	query := `
		INSERT INTO push_templates (name, title, body, icon, badge, image, url, variables, is_active, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, NOW(), NOW())
		RETURNING id, created_at, updated_at`

	return r.db.QueryRow(query,
		template.Name,
		template.Title,
		template.Body,
		template.Icon,
		template.Badge,
		template.Image,
		template.URL,
		variablesJSON,
		template.IsActive,
	).Scan(&template.ID, &template.CreatedAt, &template.UpdatedAt)
}

func (r *PushNotificationRepository) GetTemplateByName(name string) (*models.PushTemplate, error) {
	template := &models.PushTemplate{}
	var variablesJSON []byte

	query := `
		SELECT id, name, title, body, icon, badge, image, url, variables, is_active, created_at, updated_at
		FROM push_templates
		WHERE name = $1 AND is_active = true`

	err := r.db.QueryRow(query, name).Scan(
		&template.ID,
		&template.Name,
		&template.Title,
		&template.Body,
		&template.Icon,
		&template.Badge,
		&template.Image,
		&template.URL,
		&variablesJSON,
		&template.IsActive,
		&template.CreatedAt,
		&template.UpdatedAt,
	)

	if err != nil {
		return nil, err
	}

	if len(variablesJSON) > 0 {
		err = json.Unmarshal(variablesJSON, &template.Variables)
		if err != nil {
			return nil, err
		}
	}

	return template, nil
}

func (r *PushNotificationRepository) GetActiveTemplates() ([]*models.PushTemplate, error) {
	query := `
		SELECT id, name, title, body, icon, badge, image, url, variables, is_active, created_at, updated_at
		FROM push_templates
		WHERE is_active = true
		ORDER BY name`

	rows, err := r.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var templates []*models.PushTemplate
	for rows.Next() {
		template := &models.PushTemplate{}
		var variablesJSON []byte

		err := rows.Scan(
			&template.ID,
			&template.Name,
			&template.Title,
			&template.Body,
			&template.Icon,
			&template.Badge,
			&template.Image,
			&template.URL,
			&variablesJSON,
			&template.IsActive,
			&template.CreatedAt,
			&template.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}

		if len(variablesJSON) > 0 {
			err = json.Unmarshal(variablesJSON, &template.Variables)
			if err != nil {
				return nil, err
			}
		}

		templates = append(templates, template)
	}

	return templates, rows.Err()
}

// Preference management
func (r *PushNotificationRepository) CreateOrUpdatePreferences(prefs *models.NotificationPreference) error {
	categoriesJSON, _ := json.Marshal(prefs.PreferredCategories)
	tagsJSON, _ := json.Marshal(prefs.PreferredTags)
	authorsJSON, _ := json.Marshal(prefs.PreferredAuthors)

	query := `
		INSERT INTO notification_preferences (user_id, subscription_id, breaking_news, category_updates, tag_updates, author_updates, preferred_categories, preferred_tags, preferred_authors, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, NOW(), NOW())
		ON CONFLICT (subscription_id) DO UPDATE SET
			user_id = EXCLUDED.user_id,
			breaking_news = EXCLUDED.breaking_news,
			category_updates = EXCLUDED.category_updates,
			tag_updates = EXCLUDED.tag_updates,
			author_updates = EXCLUDED.author_updates,
			preferred_categories = EXCLUDED.preferred_categories,
			preferred_tags = EXCLUDED.preferred_tags,
			preferred_authors = EXCLUDED.preferred_authors,
			updated_at = NOW()
		RETURNING id, created_at, updated_at`

	return r.db.QueryRow(query,
		prefs.UserID,
		prefs.SubscriptionID,
		prefs.BreakingNews,
		prefs.CategoryUpdates,
		prefs.TagUpdates,
		prefs.AuthorUpdates,
		categoriesJSON,
		tagsJSON,
		authorsJSON,
	).Scan(&prefs.ID, &prefs.CreatedAt, &prefs.UpdatedAt)
}

func (r *PushNotificationRepository) GetPreferencesBySubscription(subscriptionID uint64) (*models.NotificationPreference, error) {
	prefs := &models.NotificationPreference{}
	var categoriesJSON, tagsJSON, authorsJSON []byte

	query := `
		SELECT id, user_id, subscription_id, breaking_news, category_updates, tag_updates, author_updates, preferred_categories, preferred_tags, preferred_authors, created_at, updated_at
		FROM notification_preferences
		WHERE subscription_id = $1`

	err := r.db.QueryRow(query, subscriptionID).Scan(
		&prefs.ID,
		&prefs.UserID,
		&prefs.SubscriptionID,
		&prefs.BreakingNews,
		&prefs.CategoryUpdates,
		&prefs.TagUpdates,
		&prefs.AuthorUpdates,
		&categoriesJSON,
		&tagsJSON,
		&authorsJSON,
		&prefs.CreatedAt,
		&prefs.UpdatedAt,
	)

	if err != nil {
		return nil, err
	}

	if len(categoriesJSON) > 0 {
		json.Unmarshal(categoriesJSON, &prefs.PreferredCategories)
	}
	if len(tagsJSON) > 0 {
		json.Unmarshal(tagsJSON, &prefs.PreferredTags)
	}
	if len(authorsJSON) > 0 {
		json.Unmarshal(authorsJSON, &prefs.PreferredAuthors)
	}

	return prefs, nil
}