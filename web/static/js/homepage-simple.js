// Simple homepage click handler
document.addEventListener('DOMContentLoaded', function() {
    console.log('Simple homepage script loaded');
    
    // Add click handler to entire document
    document.addEventListener('click', function(e) {
        console.log('Click detected on:', e.target);
        
        // Find the closest article card
        let card = e.target.closest('.article-card, .hero-main, .hero-secondary');
        if (card) {
            console.log('Card clicked:', card.className);
            
            // Find the link inside this card
            let link = card.querySelector('.article-link, .hero-link, .secondary-link');
            if (link) {
                console.log('Found link:', link.href);
                e.preventDefault();
                window.location.href = link.href;
                return;
            }
        }
        
        // If clicked directly on a link
        if (e.target.matches('.article-link, .hero-link, .secondary-link')) {
            console.log('Direct link clicked:', e.target.href);
            // Let the link work normally
        }
    });
    
    // Make cards look clickable
    const allCards = document.querySelectorAll('.article-card, .hero-main, .hero-secondary');
    allCards.forEach(card => {
        card.style.cursor = 'pointer';
    });
    
    console.log('Made', allCards.length, 'cards clickable');
});