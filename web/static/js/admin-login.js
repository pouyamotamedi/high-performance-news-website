// Admin Login JavaScript
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

        try {
            const response = await fetch('/api/v1/auth/login', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json'
                },
                body: JSON.stringify({
                    username: email,  // API expects 'username' field but we're sending email value
                    password: password
                })
            });

            const data = await response.json();

            if (response.ok) {
                // Store the token in localStorage for client-side compatibility
                localStorage.setItem('auth_token', data.data.tokens.access_token);
                localStorage.setItem('user_role', data.data.user.role);
                
                // Note: Server now sets secure HttpOnly cookie for admin panel API access
                
                // Check if user is admin or editor
                if (data.data.user.role === 'admin' || data.data.user.role === 'editor') {
                    // Redirect to admin dashboard
                    window.location.href = '/admin/dashboard';
                } else {
                    showError('Access denied. Admin or Editor role required.');
                }
            } else {
                showError(data.message || 'Invalid credentials');
            }
        } catch (error) {
            console.error('Login error:', error);
            showError('Network error. Please try again.');
        } finally {
            setLoading(false);
        }
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