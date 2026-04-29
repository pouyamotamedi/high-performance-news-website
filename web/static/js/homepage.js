// Homepage specific JavaScript
(function() {
    'use strict';
    
    // Initialize homepage functionality
    document.addEventListener('DOMContentLoaded', function() {
        console.log('Homepage loaded');
        
        // Initialize search functionality
        initializeSearch();
        
        // Initialize article interactions
        initializeArticleCards();
        
        // Initialize category cards
        initializeCategoryCards();
    });
    
    function initializeSearch() {
        const searchForm = document.querySelector('.search-form');
        const searchInput = document.querySelector('.search-input');
        
        if (searchForm && searchInput) {
            searchForm.addEventListener('submit', function(e) {
                const query = searchInput.value.trim();
                if (!query) {
                    e.preventDefault();
                    searchInput.focus();
                }
            });
        }
    }
    
    function initializeArticleCards() {
        // Make all article cards clickable
        const articleCards = document.querySelectorAll('.article-card');
        articleCards.forEach(card => {
            const articleLink = card.querySelector('.article-link');
            if (articleLink) {
                card.style.cursor = 'pointer';
                card.addEventListener('click', function() {
                    window.location.href = articleLink.href;
                });
            }
        });

        // Make hero main card clickable
        const heroMainCard = document.querySelector('.hero-main');
        if (heroMainCard) {
            const heroLink = heroMainCard.querySelector('.hero-link');
            if (heroLink) {
                heroMainCard.style.cursor = 'pointer';
                heroMainCard.addEventListener('click', function() {
                    window.location.href = heroLink.href;
                });
            }
        }
        
        // Make hero secondary cards clickable
        const heroSecondaryCards = document.querySelectorAll('.hero-secondary');
        heroSecondaryCards.forEach(card => {
            const articleLink = card.querySelector('.secondary-link');
            if (articleLink) {
                card.style.cursor = 'pointer';
                card.addEventListener('click', function() {
                    window.location.href = articleLink.href;
                });
            }
        });
    }
    
    function initializeCategoryCards() {
        const categoryCards = document.querySelectorAll('.category-card');
        
        categoryCards.forEach(card => {
            // Add click analytics (placeholder)
            card.addEventListener('click', function() {
                const categoryName = this.querySelector('.category-name')?.textContent;
                console.log('Category clicked:', categoryName);
            });
        });
    }
})();