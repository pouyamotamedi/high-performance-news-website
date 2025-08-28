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
            
            // Add search suggestions (placeholder)
            searchInput.addEventListener('input', function() {
                // TODO: Implement search suggestions
            });
        }
    }
    
    function initializeArticleCards() {
        const articleCards = document.querySelectorAll('.article-card');
        
        articleCards.forEach(card => {
            // Add hover effects
            card.addEventListener('mouseenter', function() {
                this.style.transform = 'translateY(-2px)';
            });
            
            card.addEventListener('mouseleave', function() {
                this.style.transform = 'translateY(0)';
            });
        });
    }
    
    function initializeCategoryCards() {
        const categoryCards = document.querySelectorAll('.category-card');
        
        categoryCards.forEach(card => {
            // Add click analytics (placeholder)
            card.addEventListener('click', function() {
                const categoryName = this.querySelector('.category-name')?.textContent;
                console.log('Category clicked:', categoryName);
                // TODO: Send analytics event
            });
        });
    }
})();