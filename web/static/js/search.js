/**
 * Search Page JavaScript
 * Handles search functionality, filters, pagination, and infinite scroll
 */

class SearchPage {
  constructor() {
    this.searchInput = document.getElementById('search-input');
    this.searchForm = document.getElementById('search-form');
    this.suggestionsContainer = document.getElementById('search-suggestions');
    this.suggestionsList = this.suggestionsContainer?.querySelector('.suggestions-list');
    this.clearSearchBtn = document.getElementById('clear-search');
    this.filtersPanel = document.getElementById('search-filters');
    this.resultsList = document.getElementById('results-list');
    this.loadingState = document.getElementById('loading-state');
    this.errorState = document.getElementById('error-state');
    this.emptyState = document.getElementById('empty-state');
    this.pagination = document.getElementById('pagination');
    this.infiniteLoader = document.getElementById('infinite-loader');
    this.searchStats = document.getElementById('search-stats');
    
    // State
    this.currentQuery = new URLSearchParams(window.location.search);
    this.isLoading = false;
    this.currentPage = parseInt(this.currentQuery.get('page')) || 1;
    this.hasMoreResults = true;
    this.debounceTimer = null;
    this.selectedSuggestionIndex = -1;
    this.infiniteScrollEnabled = false;
    this.observer = null;
    
    this.init();
  }

  init() {
    this.bindEvents();
    this.initFilters();
    this.initViewToggle();
    this.initInfiniteScroll();
    this.restoreState();
  }

  bindEvents() {
    // Search input with debounce
    this.searchInput?.addEventListener('input', (e) => {
      this.handleSearchInput(e.target.value);
    });

    // Search form submit
    this.searchForm?.addEventListener('submit', (e) => {
      e.preventDefault();
      this.performSearch();
    });

    // Clear search button
    this.clearSearchBtn?.addEventListener('click', () => {
      this.clearSearch();
    });

    // Keyboard navigation for suggestions
    this.searchInput?.addEventListener('keydown', (e) => {
      this.handleSuggestionKeyboard(e);
    });

    // Close suggestions on click outside
    document.addEventListener('click', (e) => {
      if (!this.searchInput?.contains(e.target) && !this.suggestionsContainer?.contains(e.target)) {
        this.hideSuggestions();
      }
    });

    // Sort select change
    document.getElementById('sort-select')?.addEventListener('change', (e) => {
      this.updateSort(e.target.value);
    });

    // Retry button
    document.getElementById('retry-btn')?.addEventListener('click', () => {
      this.performSearch();
    });

    // Mobile filter toggle
    document.getElementById('mobile-filter-toggle')?.addEventListener('click', () => {
      this.toggleMobileFilters();
    });

    // Infinite scroll toggle
    document.getElementById('infinite-scroll-toggle')?.addEventListener('change', (e) => {
      this.toggleInfiniteScroll(e.target.checked);
    });

    // Handle browser back/forward
    window.addEventListener('popstate', () => {
      this.handlePopState();
    });
  }

  // Search Input Handling
  handleSearchInput(value) {
    // Update clear button visibility
    if (this.clearSearchBtn) {
      this.clearSearchBtn.hidden = !value;
    }

    // Debounce suggestions
    clearTimeout(this.debounceTimer);
    this.debounceTimer = setTimeout(() => {
      if (value.length >= 2) {
        this.fetchSuggestions(value);
      } else {
        this.hideSuggestions();
      }
    }, 300);
  }

  async fetchSuggestions(query) {
    try {
      this.showSuggestionsLoading();
      
      const response = await fetch(`/api/v1/search/suggestions?q=${encodeURIComponent(query)}&limit=8`);
      
      if (!response.ok) throw new Error('Failed to fetch suggestions');
      
      const data = await response.json();
      
      if (data.success && data.suggestions?.length > 0) {
        this.displaySuggestions(data.suggestions, query);
      } else {
        this.hideSuggestions();
      }
    } catch (error) {
      console.warn('Suggestions fetch failed:', error);
      this.hideSuggestions();
    }
  }

  showSuggestionsLoading() {
    if (!this.suggestionsContainer) return;
    
    this.suggestionsContainer.hidden = false;
    this.suggestionsContainer.querySelector('.suggestions-loading').hidden = false;
    if (this.suggestionsList) {
      this.suggestionsList.innerHTML = '';
    }
  }

  displaySuggestions(suggestions, query) {
    if (!this.suggestionsContainer || !this.suggestionsList) return;

    this.suggestionsContainer.querySelector('.suggestions-loading').hidden = true;
    this.selectedSuggestionIndex = -1;

    const html = suggestions.map((suggestion, index) => {
      const highlighted = this.highlightMatch(suggestion, query);
      return `
        <li class="suggestion-item" role="option" data-index="${index}" data-value="${this.escapeHtml(suggestion)}">
          <svg width="16" height="16" viewBox="0 0 24 24" fill="none">
            <circle cx="11" cy="11" r="8" stroke="currentColor" stroke-width="2"/>
            <path d="m21 21-4.35-4.35" stroke="currentColor" stroke-width="2"/>
          </svg>
          <span>${highlighted}</span>
        </li>
      `;
    }).join('');

    this.suggestionsList.innerHTML = html;
    this.suggestionsContainer.hidden = false;

    // Add click handlers
    this.suggestionsList.querySelectorAll('.suggestion-item').forEach(item => {
      item.addEventListener('click', () => {
        this.selectSuggestion(item.dataset.value);
      });
    });
  }

  highlightMatch(text, query) {
    const regex = new RegExp(`(${this.escapeRegex(query)})`, 'gi');
    return this.escapeHtml(text).replace(regex, '<mark>$1</mark>');
  }

  hideSuggestions() {
    if (this.suggestionsContainer) {
      this.suggestionsContainer.hidden = true;
    }
    this.selectedSuggestionIndex = -1;
  }

  handleSuggestionKeyboard(e) {
    if (!this.suggestionsContainer || this.suggestionsContainer.hidden) return;

    const items = this.suggestionsList?.querySelectorAll('.suggestion-item');
    if (!items?.length) return;

    switch (e.key) {
      case 'ArrowDown':
        e.preventDefault();
        this.selectedSuggestionIndex = Math.min(this.selectedSuggestionIndex + 1, items.length - 1);
        this.updateSuggestionSelection(items);
        break;
      case 'ArrowUp':
        e.preventDefault();
        this.selectedSuggestionIndex = Math.max(this.selectedSuggestionIndex - 1, -1);
        this.updateSuggestionSelection(items);
        break;
      case 'Enter':
        if (this.selectedSuggestionIndex >= 0) {
          e.preventDefault();
          this.selectSuggestion(items[this.selectedSuggestionIndex].dataset.value);
        }
        break;
      case 'Escape':
        this.hideSuggestions();
        break;
    }
  }

  updateSuggestionSelection(items) {
    items.forEach((item, index) => {
      item.classList.toggle('active', index === this.selectedSuggestionIndex);
    });
  }

  selectSuggestion(value) {
    if (this.searchInput) {
      this.searchInput.value = value;
    }
    this.hideSuggestions();
    this.performSearch();
  }

  clearSearch() {
    if (this.searchInput) {
      this.searchInput.value = '';
      this.searchInput.focus();
    }
    if (this.clearSearchBtn) {
      this.clearSearchBtn.hidden = true;
    }
    this.hideSuggestions();
  }

  // Filter Handling
  initFilters() {
    // Filter toggle buttons
    document.querySelectorAll('.filter-toggle').forEach(toggle => {
      toggle.addEventListener('click', () => {
        const expanded = toggle.getAttribute('aria-expanded') === 'true';
        toggle.setAttribute('aria-expanded', !expanded);
        const content = toggle.nextElementSibling;
        if (content) {
          content.hidden = expanded;
        }
      });
    });

    // Category checkboxes
    document.querySelectorAll('#category-filters input[type="checkbox"]').forEach(checkbox => {
      checkbox.addEventListener('change', () => {
        this.updateFilters();
      });
    });

    // Tag checkboxes
    document.querySelectorAll('#tags-filters input[type="checkbox"]').forEach(checkbox => {
      checkbox.addEventListener('change', () => {
        this.updateFilters();
      });
    });

    // Date presets
    document.querySelectorAll('.date-preset').forEach(btn => {
      btn.addEventListener('click', () => {
        this.setDatePreset(btn.dataset.range);
      });
    });

    // Date inputs
    document.getElementById('date-from')?.addEventListener('change', () => {
      this.updateFilters();
    });
    document.getElementById('date-to')?.addEventListener('change', () => {
      this.updateFilters();
    });

    // Clear filters button
    document.getElementById('clear-filters')?.addEventListener('click', () => {
      this.clearFilters();
    });
  }

  setDatePreset(range) {
    const now = new Date();
    let dateFrom = new Date();

    switch (range) {
      case 'today':
        dateFrom.setHours(0, 0, 0, 0);
        break;
      case 'week':
        dateFrom.setDate(now.getDate() - 7);
        break;
      case 'month':
        dateFrom.setMonth(now.getMonth() - 1);
        break;
      case 'year':
        dateFrom.setFullYear(now.getFullYear() - 1);
        break;
    }

    const dateFromInput = document.getElementById('date-from');
    const dateToInput = document.getElementById('date-to');

    if (dateFromInput) {
      dateFromInput.value = dateFrom.toISOString().split('T')[0];
    }
    if (dateToInput) {
      dateToInput.value = now.toISOString().split('T')[0];
    }

    // Update active state
    document.querySelectorAll('.date-preset').forEach(btn => {
      btn.classList.toggle('active', btn.dataset.range === range);
    });

    this.updateFilters();
  }

  clearFilters() {
    // Clear category checkboxes
    document.querySelectorAll('#category-filters input[type="checkbox"]').forEach(cb => {
      cb.checked = false;
    });

    // Clear tag checkboxes
    document.querySelectorAll('#tags-filters input[type="checkbox"]').forEach(cb => {
      cb.checked = false;
    });

    // Clear date inputs
    const dateFrom = document.getElementById('date-from');
    const dateTo = document.getElementById('date-to');
    if (dateFrom) dateFrom.value = '';
    if (dateTo) dateTo.value = '';

    // Clear date preset active state
    document.querySelectorAll('.date-preset').forEach(btn => {
      btn.classList.remove('active');
    });

    this.updateFilters();
  }

  updateFilters() {
    this.currentPage = 1;
    this.performSearch();
  }

  toggleMobileFilters() {
    this.filtersPanel?.classList.toggle('active');
    document.body.classList.toggle('filters-open');
  }

  // View Toggle
  initViewToggle() {
    document.querySelectorAll('.view-btn').forEach(btn => {
      btn.addEventListener('click', () => {
        const view = btn.dataset.view;
        this.setView(view);
      });
    });

    // Restore saved view preference
    const savedView = localStorage.getItem('search-view') || 'list';
    this.setView(savedView);
  }

  setView(view) {
    if (this.resultsList) {
      this.resultsList.dataset.view = view;
    }

    document.querySelectorAll('.view-btn').forEach(btn => {
      btn.classList.toggle('active', btn.dataset.view === view);
    });

    localStorage.setItem('search-view', view);
  }

  // Sort
  updateSort(sortBy) {
    this.currentPage = 1;
    this.performSearch();
  }


  // Infinite Scroll
  initInfiniteScroll() {
    const toggle = document.getElementById('infinite-scroll-toggle');
    this.infiniteScrollEnabled = toggle?.checked || false;

    if (this.infiniteScrollEnabled) {
      this.setupInfiniteScrollObserver();
    }
  }

  toggleInfiniteScroll(enabled) {
    this.infiniteScrollEnabled = enabled;
    
    if (enabled) {
      this.setupInfiniteScrollObserver();
      if (this.pagination) this.pagination.hidden = true;
    } else {
      this.destroyInfiniteScrollObserver();
      if (this.pagination) this.pagination.hidden = false;
    }

    localStorage.setItem('infinite-scroll', enabled);
  }

  setupInfiniteScrollObserver() {
    if (this.observer) return;

    this.observer = new IntersectionObserver((entries) => {
      entries.forEach(entry => {
        if (entry.isIntersecting && !this.isLoading && this.hasMoreResults) {
          this.loadMoreResults();
        }
      });
    }, { rootMargin: '200px' });

    if (this.infiniteLoader) {
      this.observer.observe(this.infiniteLoader);
    }
  }

  destroyInfiniteScrollObserver() {
    if (this.observer) {
      this.observer.disconnect();
      this.observer = null;
    }
  }

  async loadMoreResults() {
    if (this.isLoading || !this.hasMoreResults) return;

    this.currentPage++;
    await this.performSearch(true);
  }

  // Search Execution
  async performSearch(append = false) {
    if (this.isLoading) return;

    this.isLoading = true;
    
    if (!append) {
      this.showLoading();
    } else if (this.infiniteLoader) {
      this.infiniteLoader.hidden = false;
    }

    try {
      const params = this.buildSearchParams();
      const response = await fetch(`/api/v1/search?${params.toString()}`);

      if (!response.ok) {
        throw new Error(`Search failed: ${response.status}`);
      }

      const data = await response.json();

      if (data.success) {
        if (append) {
          this.appendResults(data.data.articles);
        } else {
          this.displayResults(data);
        }

        // Update pagination state
        this.hasMoreResults = data.pagination?.has_next || false;
        
        // Update URL without reload
        this.updateURL(params);
      } else {
        throw new Error(data.error || 'Search failed');
      }
    } catch (error) {
      console.error('Search error:', error);
      if (!append) {
        this.showError(error.message);
      }
    } finally {
      this.isLoading = false;
      if (this.infiniteLoader) {
        this.infiniteLoader.hidden = true;
      }
    }
  }

  buildSearchParams() {
    const params = new URLSearchParams();

    // Query
    const query = this.searchInput?.value?.trim();
    if (query) {
      params.set('q', query);
    }

    // Categories
    const categories = [];
    document.querySelectorAll('#category-filters input:checked').forEach(cb => {
      categories.push(cb.value);
    });
    if (categories.length === 1) {
      params.set('category_id', categories[0]);
    }

    // Tags
    const tags = [];
    document.querySelectorAll('#tags-filters input:checked').forEach(cb => {
      tags.push(cb.value);
    });
    tags.forEach(tag => params.append('tags', tag));

    // Date range
    const dateFrom = document.getElementById('date-from')?.value;
    const dateTo = document.getElementById('date-to')?.value;
    if (dateFrom) {
      params.set('date_from', new Date(dateFrom).toISOString());
    }
    if (dateTo) {
      params.set('date_to', new Date(dateTo).toISOString());
    }

    // Sort
    const sortBy = document.getElementById('sort-select')?.value;
    if (sortBy && sortBy !== 'relevance') {
      params.set('sort_by', sortBy);
      params.set('sort_order', 'desc');
    }

    // Pagination
    params.set('page', this.currentPage.toString());
    params.set('limit', '20');

    return params;
  }

  // Results Display
  showLoading() {
    if (this.resultsList) this.resultsList.innerHTML = '';
    if (this.loadingState) this.loadingState.hidden = false;
    if (this.errorState) this.errorState.hidden = true;
    if (this.emptyState) this.emptyState.hidden = true;
    if (this.pagination) this.pagination.hidden = true;
  }

  hideLoading() {
    if (this.loadingState) this.loadingState.hidden = true;
  }

  showError(message) {
    this.hideLoading();
    if (this.resultsList) this.resultsList.innerHTML = '';
    if (this.errorState) {
      this.errorState.hidden = false;
      const errorMsg = document.getElementById('error-message');
      if (errorMsg) errorMsg.textContent = message;
    }
    if (this.emptyState) this.emptyState.hidden = true;
    if (this.pagination) this.pagination.hidden = true;
  }

  displayResults(data) {
    this.hideLoading();
    
    const articles = data.data?.articles || [];
    const total = data.pagination?.total || 0;
    const searchTime = data.data?.processing_time_ms || 0;

    // Update stats
    this.updateSearchStats(total, searchTime);

    if (articles.length === 0) {
      if (this.emptyState) this.emptyState.hidden = false;
      if (this.resultsList) this.resultsList.innerHTML = '';
      if (this.pagination) this.pagination.hidden = true;
      return;
    }

    if (this.emptyState) this.emptyState.hidden = true;
    
    // Render results
    const html = articles.map(article => this.renderArticle(article)).join('');
    if (this.resultsList) {
      this.resultsList.innerHTML = html;
    }

    // Update pagination
    if (!this.infiniteScrollEnabled) {
      this.updatePagination(data.pagination);
    }
  }

  appendResults(articles) {
    if (!articles?.length || !this.resultsList) return;

    const html = articles.map(article => this.renderArticle(article)).join('');
    this.resultsList.insertAdjacentHTML('beforeend', html);
  }

  renderArticle(article) {
    const query = this.searchInput?.value?.trim() || '';
    const title = query ? this.highlightMatch(article.title, query) : this.escapeHtml(article.title);
    const excerpt = article.excerpt ? 
      (query ? this.highlightMatch(article.excerpt, query) : this.escapeHtml(article.excerpt)) : '';

    return `
      <article class="search-result-item" itemscope itemtype="https://schema.org/Article">
        <div class="result-image">
          ${article.featured_image ? `
            <img src="${this.escapeHtml(article.featured_image)}" alt="${this.escapeHtml(article.title)}" 
                 class="result-img" loading="lazy" itemprop="image">
          ` : `
            <div class="result-img placeholder-img">
              <svg width="32" height="32" viewBox="0 0 24 24" fill="none">
                <rect x="3" y="3" width="18" height="18" rx="2" stroke="currentColor" stroke-width="2"/>
                <circle cx="8.5" cy="8.5" r="1.5" fill="currentColor"/>
                <path d="M21 15l-5-5L5 21" stroke="currentColor" stroke-width="2"/>
              </svg>
            </div>
          `}
        </div>
        <div class="result-content">
          <div class="result-meta-top">
            <a href="/category/${this.escapeHtml(article.category_slug || '')}" class="result-category" itemprop="articleSection">
              ${this.escapeHtml(article.category || 'Uncategorized')}
            </a>
            <time class="result-date" datetime="${article.published_at || ''}" itemprop="datePublished">
              ${this.formatTimeAgo(article.published_at)}
            </time>
          </div>
          <h2 class="result-title" itemprop="headline">
            <a href="/article/${this.escapeHtml(article.slug)}" class="result-link" itemprop="url">
              ${title}
            </a>
          </h2>
          <p class="result-excerpt" itemprop="description">${excerpt}</p>
          <div class="result-meta-bottom">
            <span class="result-author" itemprop="author">${this.escapeHtml(article.author || 'Unknown')}</span>
            <span class="result-views">
              <svg width="16" height="16" viewBox="0 0 24 24" fill="none">
                <path d="M1 12s4-8 11-8 11 8 11 8-4 8-11 8-11-8-11-8z" stroke="currentColor" stroke-width="2"/>
                <circle cx="12" cy="12" r="3" stroke="currentColor" stroke-width="2"/>
              </svg>
              ${this.formatNumber(article.view_count || 0)}
            </span>
          </div>
        </div>
      </article>
    `;
  }

  updateSearchStats(total, searchTime) {
    if (!this.searchStats) return;

    const query = this.searchInput?.value?.trim();
    if (!query) {
      this.searchStats.hidden = true;
      return;
    }

    this.searchStats.hidden = false;
    const countEl = this.searchStats.querySelector('.results-count strong:first-child');
    const queryEl = this.searchStats.querySelector('.results-count strong:last-child');
    const timeEl = this.searchStats.querySelector('.search-time');

    if (countEl) countEl.textContent = total.toLocaleString();
    if (queryEl) queryEl.textContent = query;
    if (timeEl) timeEl.textContent = `(${searchTime}ms)`;
  }

  updatePagination(paginationData) {
    if (!this.pagination || !paginationData) {
      if (this.pagination) this.pagination.hidden = true;
      return;
    }

    const { current_page, total_pages, has_prev, has_next } = paginationData;

    if (total_pages <= 1) {
      this.pagination.hidden = true;
      return;
    }

    this.pagination.hidden = false;

    // Generate page numbers
    const pages = this.generatePageNumbers(current_page, total_pages);
    const params = this.buildSearchParams();

    let html = '';

    // Previous button
    if (has_prev) {
      params.set('page', (current_page - 1).toString());
      html += `
        <a href="/search?${params.toString()}" class="pagination-btn prev" rel="prev">
          <svg width="20" height="20" viewBox="0 0 24 24" fill="none">
            <path d="M15 18l-6-6 6-6" stroke="currentColor" stroke-width="2"/>
          </svg>
          <span>Previous</span>
        </a>
      `;
    }

    // Page numbers
    html += '<div class="pagination-pages">';
    pages.forEach(page => {
      if (page === -1) {
        html += '<span class="pagination-ellipsis">...</span>';
      } else {
        params.set('page', page.toString());
        const isActive = page === current_page;
        html += `
          <a href="/search?${params.toString()}" 
             class="pagination-page ${isActive ? 'active' : ''}"
             ${isActive ? 'aria-current="page"' : ''}>
            ${page}
          </a>
        `;
      }
    });
    html += '</div>';

    // Next button
    if (has_next) {
      params.set('page', (current_page + 1).toString());
      html += `
        <a href="/search?${params.toString()}" class="pagination-btn next" rel="next">
          <span>Next</span>
          <svg width="20" height="20" viewBox="0 0 24 24" fill="none">
            <path d="M9 18l6-6-6-6" stroke="currentColor" stroke-width="2"/>
          </svg>
        </a>
      `;
    }

    this.pagination.innerHTML = html;

    // Add click handlers for AJAX pagination
    this.pagination.querySelectorAll('a').forEach(link => {
      link.addEventListener('click', (e) => {
        e.preventDefault();
        const url = new URL(link.href);
        this.currentPage = parseInt(url.searchParams.get('page')) || 1;
        this.performSearch();
        window.scrollTo({ top: 0, behavior: 'smooth' });
      });
    });
  }

  generatePageNumbers(current, total) {
    const pages = [];
    const delta = 2;

    for (let i = 1; i <= total; i++) {
      if (i === 1 || i === total || (i >= current - delta && i <= current + delta)) {
        pages.push(i);
      } else if (pages[pages.length - 1] !== -1) {
        pages.push(-1);
      }
    }

    return pages;
  }

  // URL Management
  updateURL(params) {
    const newURL = `${window.location.pathname}?${params.toString()}`;
    window.history.pushState({ page: this.currentPage }, '', newURL);
  }

  handlePopState() {
    this.currentQuery = new URLSearchParams(window.location.search);
    this.currentPage = parseInt(this.currentQuery.get('page')) || 1;
    this.restoreState();
    this.performSearch();
  }

  restoreState() {
    // Restore search query
    const query = this.currentQuery.get('q');
    if (this.searchInput && query) {
      this.searchInput.value = query;
      if (this.clearSearchBtn) {
        this.clearSearchBtn.hidden = false;
      }
    }

    // Restore sort
    const sortBy = this.currentQuery.get('sort_by') || 'relevance';
    const sortSelect = document.getElementById('sort-select');
    if (sortSelect) {
      sortSelect.value = sortBy;
    }

    // Restore infinite scroll preference
    const infiniteScroll = localStorage.getItem('infinite-scroll') === 'true';
    const toggle = document.getElementById('infinite-scroll-toggle');
    if (toggle) {
      toggle.checked = infiniteScroll;
      this.toggleInfiniteScroll(infiniteScroll);
    }
  }

  // Utility Methods
  escapeHtml(str) {
    if (!str) return '';
    const div = document.createElement('div');
    div.textContent = str;
    return div.innerHTML;
  }

  escapeRegex(str) {
    return str.replace(/[.*+?^${}()|[\]\\]/g, '\\$&');
  }

  formatTimeAgo(dateStr) {
    if (!dateStr) return '';
    
    const date = new Date(dateStr);
    const now = new Date();
    const diffMs = now - date;
    const diffSec = Math.floor(diffMs / 1000);
    const diffMin = Math.floor(diffSec / 60);
    const diffHour = Math.floor(diffMin / 60);
    const diffDay = Math.floor(diffHour / 24);

    if (diffDay > 30) {
      return date.toLocaleDateString();
    } else if (diffDay > 0) {
      return `${diffDay}d ago`;
    } else if (diffHour > 0) {
      return `${diffHour}h ago`;
    } else if (diffMin > 0) {
      return `${diffMin}m ago`;
    } else {
      return 'Just now';
    }
  }

  formatNumber(num) {
    if (num >= 1000000) {
      return (num / 1000000).toFixed(1) + 'M';
    } else if (num >= 1000) {
      return (num / 1000).toFixed(1) + 'K';
    }
    return num.toString();
  }
}

// Initialize when DOM is ready
document.addEventListener('DOMContentLoaded', () => {
  window.searchPage = new SearchPage();
});
