package repositories

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/lib/pq"
	"high-performance-news-website/internal/models"
)

// AnalyticsRepository handles analytics data operations
type AnalyticsRepository struct {
	db *sql.DB
}

// NewAnalyticsRepository creates a new analytics repository
func NewAnalyticsRepository(db *sql.DB) *AnalyticsRepository {
	return &AnalyticsRepository{db: db}
}

// RecordArticleView records a new article view
func (r *AnalyticsRepository) RecordArticleView(view *models.ArticleView) error {
	query := `
		INSERT INTO article_views (article_id, ip_address, user_agent, referer, created_at)
		VALUES ($1, $2, $3, $4, $5)
	`
	
	_, err := r.db.Exec(query, view.ArticleID, view.IPAddress, view.UserAgent, view.Referer, view.CreatedAt)
	return err
}

// RecordArticleEngagement records a new engagement action
func (r *AnalyticsRepository) RecordArticleEngagement(engagement *models.ArticleEngagement) error {
	query := `
		INSERT INTO article_engagement (article_id, action, ip_address, created_at)
		VALUES ($1, $2, $3, $4)
	`
	
	_, err := r.db.Exec(query, engagement.ArticleID, engagement.Action, engagement.IPAddress, engagement.CreatedAt)
	return err
}

// RecordUserBehavior records user behavior data
func (r *AnalyticsRepository) RecordUserBehavior(behavior *models.UserBehavior) error {
	query := `
		INSERT INTO user_behavior (session_id, user_id, ip_address, user_agent, page_url, referer, 
		                          time_on_page, scroll_depth, behavior_data, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`
	
	_, err := r.db.Exec(query, behavior.SessionID, behavior.UserID, behavior.IPAddress, 
		behavior.UserAgent, behavior.PageURL, behavior.Referer, behavior.TimeOnPage, 
		behavior.ScrollDepth, behavior.BehaviorData, behavior.CreatedAt)
	return err
}

// RecordPerformanceMetric records a performance metric
func (r *AnalyticsRepository) RecordPerformanceMetric(metric *models.PerformanceMetric) error {
	query := `
		INSERT INTO performance_metrics (metric_type, name, value, unit, tags, created_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`
	
	_, err := r.db.Exec(query, metric.MetricType, metric.Name, metric.Value, 
		metric.Unit, metric.Tags, metric.CreatedAt)
	return err
}

// GetArticleAnalytics retrieves analytics for a specific article
func (r *AnalyticsRepository) GetArticleAnalytics(articleID uint64, startDate, endDate time.Time) (*models.ArticleAnalytics, error) {
	query := `
		WITH article_info AS (
			SELECT id, title, slug, published_at
			FROM articles
			WHERE id = $1
			LIMIT 1
		),
		view_stats AS (
			SELECT 
				COUNT(*) as view_count,
				COUNT(DISTINCT ip_address) as unique_views
			FROM article_views
			WHERE article_id = $1 
			AND created_at BETWEEN $2 AND $3
		),
		engagement_stats AS (
			SELECT 
				action,
				COUNT(*) as count
			FROM article_engagement
			WHERE article_id = $1 
			AND created_at BETWEEN $2 AND $3
			GROUP BY action
		),
		behavior_stats AS (
			SELECT 
				AVG(time_on_page) as avg_time_on_page,
				AVG(scroll_depth) as avg_scroll_depth,
				COUNT(CASE WHEN time_on_page < 30 THEN 1 END)::float / COUNT(*)::float as bounce_rate
			FROM user_behavior
			WHERE page_url LIKE '%' || (SELECT slug FROM article_info) || '%'
			AND created_at BETWEEN $2 AND $3
		)
		SELECT 
			ai.id,
			ai.title,
			ai.slug,
			ai.published_at,
			COALESCE(vs.view_count, 0) as view_count,
			COALESCE(vs.unique_views, 0) as unique_views,
			COALESCE(es_like.count, 0) as like_count,
			COALESCE(es_dislike.count, 0) as dislike_count,
			COALESCE(es_share.count, 0) as share_count,
			COALESCE(es_comment.count, 0) as comment_count,
			COALESCE(bs.avg_time_on_page, 0) as avg_time_on_page,
			COALESCE(bs.avg_scroll_depth, 0) as avg_scroll_depth,
			COALESCE(bs.bounce_rate, 0) as bounce_rate,
			CASE 
				WHEN vs.view_count > 0 THEN 
					(COALESCE(es_like.count, 0) + COALESCE(es_dislike.count, 0) + 
					 COALESCE(es_share.count, 0) + COALESCE(es_comment.count, 0))::float / vs.view_count::float
				ELSE 0
			END as engagement_rate
		FROM article_info ai
		CROSS JOIN view_stats vs
		CROSS JOIN behavior_stats bs
		LEFT JOIN engagement_stats es_like ON es_like.action = 'like'
		LEFT JOIN engagement_stats es_dislike ON es_dislike.action = 'dislike'
		LEFT JOIN engagement_stats es_share ON es_share.action = 'share'
		LEFT JOIN engagement_stats es_comment ON es_comment.action = 'comment'
	`

	analytics := &models.ArticleAnalytics{}
	err := r.db.QueryRow(query, articleID, startDate, endDate).Scan(
		&analytics.ArticleID,
		&analytics.Title,
		&analytics.Slug,
		&analytics.PublishedAt,
		&analytics.ViewCount,
		&analytics.UniqueViews,
		&analytics.LikeCount,
		&analytics.DislikeCount,
		&analytics.ShareCount,
		&analytics.CommentCount,
		&analytics.AvgTimeOnPage,
		&analytics.AvgScrollDepth,
		&analytics.BounceRate,
		&analytics.EngagementRate,
	)

	if err != nil {
		return nil, err
	}

	// Get top referers
	refererQuery := `
		SELECT referer
		FROM article_views
		WHERE article_id = $1 
		AND created_at BETWEEN $2 AND $3
		AND referer IS NOT NULL AND referer != ''
		GROUP BY referer
		ORDER BY COUNT(*) DESC
		LIMIT 5
	`
	
	rows, err := r.db.Query(refererQuery, articleID, startDate, endDate)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var referers []string
	for rows.Next() {
		var referer string
		if err := rows.Scan(&referer); err != nil {
			continue
		}
		referers = append(referers, referer)
	}
	analytics.TopReferers = referers

	// Get device breakdown
	deviceQuery := `
		SELECT 
			behavior_data->>'device' as device,
			COUNT(*) as count
		FROM user_behavior
		WHERE page_url LIKE '%' || $1 || '%'
		AND created_at BETWEEN $2 AND $3
		AND behavior_data->>'device' IS NOT NULL
		GROUP BY behavior_data->>'device'
	`
	
	rows, err = r.db.Query(deviceQuery, analytics.Slug, startDate, endDate)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	analytics.DeviceBreakdown = make(map[string]int64)
	for rows.Next() {
		var device string
		var count int64
		if err := rows.Scan(&device, &count); err != nil {
			continue
		}
		analytics.DeviceBreakdown[device] = count
	}

	return analytics, nil
}

// GetTopArticles retrieves top performing articles
func (r *AnalyticsRepository) GetTopArticles(startDate, endDate time.Time, limit int) ([]models.ArticleAnalytics, error) {
	query := `
		WITH article_views_agg AS (
			SELECT 
				av.article_id,
				COUNT(*) as view_count,
				COUNT(DISTINCT av.ip_address) as unique_views
			FROM article_views av
			WHERE av.created_at BETWEEN $1 AND $2
			GROUP BY av.article_id
		),
		engagement_agg AS (
			SELECT 
				ae.article_id,
				SUM(CASE WHEN ae.action = 'like' THEN 1 ELSE 0 END) as like_count,
				SUM(CASE WHEN ae.action = 'dislike' THEN 1 ELSE 0 END) as dislike_count,
				SUM(CASE WHEN ae.action = 'share' THEN 1 ELSE 0 END) as share_count,
				SUM(CASE WHEN ae.action = 'comment' THEN 1 ELSE 0 END) as comment_count
			FROM article_engagement ae
			WHERE ae.created_at BETWEEN $1 AND $2
			GROUP BY ae.article_id
		)
		SELECT 
			a.id,
			a.title,
			a.slug,
			a.published_at,
			COALESCE(ava.view_count, 0) as view_count,
			COALESCE(ava.unique_views, 0) as unique_views,
			COALESCE(ea.like_count, 0) as like_count,
			COALESCE(ea.dislike_count, 0) as dislike_count,
			COALESCE(ea.share_count, 0) as share_count,
			COALESCE(ea.comment_count, 0) as comment_count,
			0 as avg_time_on_page,
			0 as avg_scroll_depth,
			0 as bounce_rate,
			CASE 
				WHEN COALESCE(ava.view_count, 0) > 0 THEN 
					(COALESCE(ea.like_count, 0) + COALESCE(ea.dislike_count, 0) + 
					 COALESCE(ea.share_count, 0) + COALESCE(ea.comment_count, 0))::float / ava.view_count::float
				ELSE 0
			END as engagement_rate
		FROM articles a
		LEFT JOIN article_views_agg ava ON a.id = ava.article_id
		LEFT JOIN engagement_agg ea ON a.id = ea.article_id
		WHERE a.status = 'published'
		AND (ava.view_count > 0 OR ea.like_count > 0 OR ea.dislike_count > 0 OR ea.share_count > 0 OR ea.comment_count > 0)
		ORDER BY COALESCE(ava.view_count, 0) DESC, engagement_rate DESC
		LIMIT $3
	`

	rows, err := r.db.Query(query, startDate, endDate, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var articles []models.ArticleAnalytics
	for rows.Next() {
		var article models.ArticleAnalytics
		err := rows.Scan(
			&article.ArticleID,
			&article.Title,
			&article.Slug,
			&article.PublishedAt,
			&article.ViewCount,
			&article.UniqueViews,
			&article.LikeCount,
			&article.DislikeCount,
			&article.ShareCount,
			&article.CommentCount,
			&article.AvgTimeOnPage,
			&article.AvgScrollDepth,
			&article.BounceRate,
			&article.EngagementRate,
		)
		if err != nil {
			continue
		}
		articles = append(articles, article)
	}

	return articles, nil
}

// GetTrafficSources retrieves traffic source analytics
func (r *AnalyticsRepository) GetTrafficSources(startDate, endDate time.Time) ([]models.TrafficSource, error) {
	query := `
		SELECT 
			COALESCE(behavior_data->>'utm_source', 'direct') as source,
			COALESCE(behavior_data->>'utm_medium', 'none') as medium,
			COALESCE(behavior_data->>'utm_campaign', '') as campaign,
			COUNT(DISTINCT session_id) as sessions,
			COUNT(DISTINCT ip_address) as users,
			COUNT(*) as page_views,
			COUNT(CASE WHEN time_on_page < 30 THEN 1 END)::float / COUNT(*)::float as bounce_rate,
			AVG(time_on_page) as avg_duration
		FROM user_behavior
		WHERE created_at BETWEEN $1 AND $2
		GROUP BY 
			COALESCE(behavior_data->>'utm_source', 'direct'),
			COALESCE(behavior_data->>'utm_medium', 'none'),
			COALESCE(behavior_data->>'utm_campaign', '')
		ORDER BY sessions DESC
		LIMIT 20
	`

	rows, err := r.db.Query(query, startDate, endDate)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sources []models.TrafficSource
	for rows.Next() {
		var source models.TrafficSource
		err := rows.Scan(
			&source.Source,
			&source.Medium,
			&source.Campaign,
			&source.Sessions,
			&source.Users,
			&source.PageViews,
			&source.BounceRate,
			&source.AvgDuration,
		)
		if err != nil {
			continue
		}
		sources = append(sources, source)
	}

	return sources, nil
}

// GetDashboardMetrics retrieves key metrics for the dashboard
func (r *AnalyticsRepository) GetDashboardMetrics(startDate, endDate time.Time) (*models.DashboardMetrics, error) {
	metrics := &models.DashboardMetrics{}

	// Get total views and unique visitors
	viewQuery := `
		SELECT 
			COUNT(*) as total_views,
			COUNT(DISTINCT ip_address) as unique_visitors
		FROM article_views
		WHERE created_at BETWEEN $1 AND $2
	`
	
	err := r.db.QueryRow(viewQuery, startDate, endDate).Scan(&metrics.TotalViews, &metrics.UniqueVisitors)
	if err != nil {
		return nil, err
	}

	// Get total engagements
	engagementQuery := `
		SELECT COUNT(*) as total_engagements
		FROM article_engagement
		WHERE created_at BETWEEN $1 AND $2
	`
	
	err = r.db.QueryRow(engagementQuery, startDate, endDate).Scan(&metrics.TotalEngagements)
	if err != nil {
		return nil, err
	}

	// Get average time on site and bounce rate
	behaviorQuery := `
		SELECT 
			AVG(time_on_page) as avg_time_on_site,
			COUNT(CASE WHEN time_on_page < 30 THEN 1 END)::float / COUNT(*)::float as bounce_rate
		FROM user_behavior
		WHERE created_at BETWEEN $1 AND $2
	`
	
	err = r.db.QueryRow(behaviorQuery, startDate, endDate).Scan(&metrics.AvgTimeOnSite, &metrics.BounceRate)
	if err != nil {
		return nil, err
	}

	// Get top articles
	topArticles, err := r.GetTopArticles(startDate, endDate, 10)
	if err != nil {
		return nil, err
	}
	metrics.TopArticles = topArticles

	// Get traffic sources
	trafficSources, err := r.GetTrafficSources(startDate, endDate)
	if err != nil {
		return nil, err
	}
	metrics.TrafficSources = trafficSources

	// Get device breakdown
	deviceQuery := `
		SELECT 
			behavior_data->>'device' as device,
			COUNT(*) as count
		FROM user_behavior
		WHERE created_at BETWEEN $1 AND $2
		AND behavior_data->>'device' IS NOT NULL
		GROUP BY behavior_data->>'device'
		ORDER BY count DESC
	`
	
	rows, err := r.db.Query(deviceQuery, startDate, endDate)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	metrics.DeviceBreakdown = make(map[string]int64)
	for rows.Next() {
		var device string
		var count int64
		if err := rows.Scan(&device, &count); err != nil {
			continue
		}
		metrics.DeviceBreakdown[device] = count
	}

	// Get country breakdown
	countryQuery := `
		SELECT 
			behavior_data->>'country' as country,
			COUNT(*) as count
		FROM user_behavior
		WHERE created_at BETWEEN $1 AND $2
		AND behavior_data->>'country' IS NOT NULL
		GROUP BY behavior_data->>'country'
		ORDER BY count DESC
		LIMIT 10
	`
	
	rows, err = r.db.Query(countryQuery, startDate, endDate)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	metrics.CountryBreakdown = make(map[string]int64)
	for rows.Next() {
		var country string
		var count int64
		if err := rows.Scan(&country, &count); err != nil {
			continue
		}
		metrics.CountryBreakdown[country] = count
	}

	// Get hourly traffic
	hourlyQuery := `
		SELECT 
			EXTRACT(HOUR FROM created_at) as hour,
			COUNT(*) as views,
			COUNT(DISTINCT ip_address) as visitors
		FROM article_views
		WHERE created_at BETWEEN $1 AND $2
		GROUP BY EXTRACT(HOUR FROM created_at)
		ORDER BY hour
	`
	
	rows, err = r.db.Query(hourlyQuery, startDate, endDate)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var hourlyData models.HourlyTrafficData
		err := rows.Scan(&hourlyData.Hour, &hourlyData.Views, &hourlyData.Visitors)
		if err != nil {
			continue
		}
		metrics.HourlyTraffic = append(metrics.HourlyTraffic, hourlyData)
	}

	// Get performance metrics
	perfQuery := `
		SELECT 
			name,
			AVG(value) as avg_value
		FROM performance_metrics
		WHERE created_at BETWEEN $1 AND $2
		GROUP BY name
	`
	
	rows, err = r.db.Query(perfQuery, startDate, endDate)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	metrics.PerformanceMetrics = make(map[string]float64)
	for rows.Next() {
		var name string
		var avgValue float64
		if err := rows.Scan(&name, &avgValue); err != nil {
			continue
		}
		metrics.PerformanceMetrics[name] = avgValue
	}

	return metrics, nil
}

// SaveReport saves an analytics report
func (r *AnalyticsRepository) SaveReport(report *models.AnalyticsReport) error {
	query := `
		INSERT INTO analytics_reports (name, report_type, parameters, data, generated_by, generated_at, expires_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id
	`
	
	err := r.db.QueryRow(query, report.Name, report.ReportType, report.Parameters, 
		report.Data, report.GeneratedBy, report.GeneratedAt, report.ExpiresAt).Scan(&report.ID)
	return err
}

// GetReport retrieves a saved analytics report
func (r *AnalyticsRepository) GetReport(reportID uint64) (*models.AnalyticsReport, error) {
	query := `
		SELECT id, name, report_type, parameters, data, generated_by, generated_at, expires_at
		FROM analytics_reports
		WHERE id = $1
	`
	
	report := &models.AnalyticsReport{}
	err := r.db.QueryRow(query, reportID).Scan(
		&report.ID,
		&report.Name,
		&report.ReportType,
		&report.Parameters,
		&report.Data,
		&report.GeneratedBy,
		&report.GeneratedAt,
		&report.ExpiresAt,
	)
	
	return report, err
}

// BulkRecordViews records multiple article views efficiently
func (r *AnalyticsRepository) BulkRecordViews(views []models.ArticleView) error {
	if len(views) == 0 {
		return nil
	}

	// Use PostgreSQL COPY for maximum performance
	stmt, err := r.db.Prepare(pq.CopyIn("article_views", 
		"article_id", "ip_address", "user_agent", "referer", "created_at"))
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, view := range views {
		_, err = stmt.Exec(view.ArticleID, view.IPAddress, view.UserAgent, view.Referer, view.CreatedAt)
		if err != nil {
			return err
		}
	}

	return stmt.Close()
}

// BulkRecordEngagements records multiple engagements efficiently
func (r *AnalyticsRepository) BulkRecordEngagements(engagements []models.ArticleEngagement) error {
	if len(engagements) == 0 {
		return nil
	}

	stmt, err := r.db.Prepare(pq.CopyIn("article_engagement", 
		"article_id", "action", "ip_address", "created_at"))
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, engagement := range engagements {
		_, err = stmt.Exec(engagement.ArticleID, engagement.Action, engagement.IPAddress, engagement.CreatedAt)
		if err != nil {
			return err
		}
	}

	return stmt.Close()
}

// GetAnalyticsData retrieves analytics data with flexible filtering
func (r *AnalyticsRepository) GetAnalyticsData(params models.ReportParameters) (map[string]interface{}, error) {
	data := make(map[string]interface{})

	// Build dynamic query based on parameters
	var conditions []string
	var args []interface{}
	argIndex := 1

	// Add date range condition
	conditions = append(conditions, fmt.Sprintf("created_at BETWEEN $%d AND $%d", argIndex, argIndex+1))
	args = append(args, params.StartDate, params.EndDate)
	argIndex += 2

	// Add filters
	for key, value := range params.Filters {
		switch key {
		case "article_id":
			conditions = append(conditions, fmt.Sprintf("article_id = $%d", argIndex))
			args = append(args, value)
			argIndex++
		case "ip_address":
			conditions = append(conditions, fmt.Sprintf("ip_address = $%d", argIndex))
			args = append(args, value)
			argIndex++
		}
	}

	whereClause := ""
	if len(conditions) > 0 {
		whereClause = "WHERE " + strings.Join(conditions, " AND ")
	}

	// Build GROUP BY clause
	groupByClause := ""
	if len(params.GroupBy) > 0 {
		groupByClause = "GROUP BY " + strings.Join(params.GroupBy, ", ")
	}

	// Build LIMIT and OFFSET
	limitClause := ""
	if params.Limit > 0 {
		limitClause = fmt.Sprintf("LIMIT %d", params.Limit)
		if params.Offset > 0 {
			limitClause += fmt.Sprintf(" OFFSET %d", params.Offset)
		}
	}

	// Execute query based on metrics requested
	for _, metric := range params.Metrics {
		switch metric {
		case "views":
			query := fmt.Sprintf(`
				SELECT COUNT(*) as total_views
				FROM article_views
				%s
				%s
				%s
			`, whereClause, groupByClause, limitClause)
			
			var totalViews int64
			err := r.db.QueryRow(query, args...).Scan(&totalViews)
			if err != nil {
				return nil, err
			}
			data["total_views"] = totalViews

		case "engagements":
			query := fmt.Sprintf(`
				SELECT action, COUNT(*) as count
				FROM article_engagement
				%s
				GROUP BY action
				%s
			`, whereClause, limitClause)
			
			rows, err := r.db.Query(query, args...)
			if err != nil {
				return nil, err
			}
			defer rows.Close()

			engagements := make(map[string]int64)
			for rows.Next() {
				var action string
				var count int64
				if err := rows.Scan(&action, &count); err != nil {
					continue
				}
				engagements[action] = count
			}
			data["engagements"] = engagements
		}
	}

	return data, nil
}