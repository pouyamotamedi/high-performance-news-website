// Article page JavaScript functionality
document.addEventListener('DOMContentLoaded', function() {
    // Article page enhancements
    console.log('Article page loaded');
    
    // Add any article-specific functionality here
    // For example: reading progress, social sharing, etc.
    
    // Reading progress indicator
    const article = document.querySelector('.article-main');
    if (article) {
        let ticking = false;
        
        function updateReadingProgress() {
            const scrollTop = window.pageYOffset;
            const docHeight = document.documentElement.scrollHeight - window.innerHeight;
            const scrollPercent = (scrollTop / docHeight) * 100;
            
            // You could add a progress bar here
            // const progressBar = document.querySelector('.reading-progress');
            // if (progressBar) {
            //     progressBar.style.width = scrollPercent + '%';
            // }
            
            ticking = false;
        }
        
        function requestTick() {
            if (!ticking) {
                requestAnimationFrame(updateReadingProgress);
                ticking = true;
            }
        }
        
        window.addEventListener('scroll', requestTick);
    }
    
    // Social sharing functionality
    const shareButtons = document.querySelectorAll('.share-button');
    shareButtons.forEach(button => {
        button.addEventListener('click', function(e) {
            e.preventDefault();
            const url = window.location.href;
            const title = document.title;
            
            if (navigator.share) {
                navigator.share({
                    title: title,
                    url: url
                });
            } else {
                // Fallback for browsers without Web Share API
                navigator.clipboard.writeText(url).then(() => {
                    console.log('URL copied to clipboard');
                });
            }
        });
    });
});