// Development Admin Login - Bypasses API authentication
document.addEventListener('DOMContentLoaded', function() {
    // Check if already logged in
    const token = localStorage.getItem('auth_token');
    const role = localStorage.getItem('user_role');
    
    if (token && (role === 'admin' || role === 'editor')) {
        window.location.href = '/admin/dashboard';
        return;
    }

    // Initialize login form
    const loginForm = document.getElementById('loginForm');
    const emailInput = document.getElementById('email');
    const passwordInput = document.getElementById('password');
    const loginButton = document.getElementById('loginButton');
    const errorDiv = document.getElementById('errorMessage');
    const loadingSpan = document.getElementById('loadingSpan');
    const buttonText = document.getElementById('buttonText');

    // Add demo credentials hint
    const demoHint = document.createElement('div');
    demoHint.innerHTML = `
        <div style="background: #e0f2fe; border: 1px solid #0288d1; border-radius: 6px; padding: 1rem; margin-bottom: 1rem;">
            <h4 style="margin: 0 0 0.5rem 0; color: #01579b;">Development Mode</h4>
            <p style="margin: 0; color: #0277bd;">Use any email/password combination to login (e.g., admin@demo.com / demo123)</p>
        </div>
    `;
    loginForm.parentNode.insertBefore(demoHint, loginForm);

    loginForm.addEventListener('submit', async function(e) {
        e.preventDefault();
        
        const email = emailInput.value.trim();
        const password = passwordInput.value.trim();
        
        if (!email || !password) {
            showError('Please enter both email and password');
            return;
        }

        setLoading(true);
        hideError();

        // Simulate API delay
        setTimeout(() => {
            // Mock successful authentication
            const mockToken = 'dev-token-' + Date.now();
            const mockRole = 'admin';
            
            // Store the token
            localStorage.setItem('auth_token', mockToken);
            localStorage.setItem('user_role', mockRole);
            
            // Redirect to admin dashboard
            window.location.href = '/admin/dashboard';
        }, 500);
    });

    function setLoading(loading) {
        loginButton.disabled = loading;
        if (loading) {
            loadingSpan.classList.remove('hidden');
            buttonText.textContent = 'Signing in...';
        } else {
            loadingSpan.classList.add('hidden');
            buttonText.textContent = 'Sign in';
        }
    }

    function showError(message) {
        errorDiv.textContent = message;
        errorDiv.classList.remove('hidden');
    }

    function hideError() {
        errorDiv.classList.add('hidden');
    }
});