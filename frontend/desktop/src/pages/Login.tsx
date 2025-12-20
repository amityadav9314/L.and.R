import { GoogleLogin } from '@react-oauth/google';
import { useAuthStore } from '../store/authStore';
import { authClient } from '../services/api';
import { useNavigate } from 'react-router-dom';

const Login = () => {
    const { login } = useAuthStore();
    const navigate = useNavigate();

    const handleSuccess = async (credentialResponse: any) => {
        const idToken = credentialResponse.credential;
        if (!idToken) return;

        try {
            console.log('[Login] Verifying token with backend...');
            const response = await authClient.login({ googleIdToken: idToken });

            if (response.user && response.sessionToken) {
                await login(response.user, response.sessionToken);
                navigate('/');
            }
        } catch (error) {
            console.error('[Login] Backend verification failed:', error);
            alert('Failed to login. Please try again.');
        }
    };

    return (
        <div className="d-flex justify-content-center align-items-center vh-100 bg-light">
            <div className="card shadow-lg border-0 p-4 p-md-5 bg-white rounded-4" style={{ maxWidth: '450px', width: '90%' }}>
                <div className="text-center mb-4">
                    <h1 className="h2 fw-bold text-primary mb-2">L.and.R</h1>
                    <p className="text-muted">Learn and Revise smarter with AI</p>
                </div>

                <div className="text-center py-4 bg-light rounded-3 mb-4 border border-dashed">
                    <div className="d-flex justify-content-center mb-3">
                        <div className="bg-primary text-white p-3 rounded-circle shadow-sm">
                            <span className="fs-3">💡</span>
                        </div>
                    </div>
                    <h5>Welcome Back</h5>
                    <p className="small text-muted px-4">Sign in with your Google account to access your materials and flashcards.</p>
                </div>

                <div className="d-flex justify-content-center">
                    <GoogleLogin
                        onSuccess={handleSuccess}
                        onError={() => {
                            console.log('Login Failed');
                            alert('Google Login Failed');
                        }}
                        useOneTap
                        shape="pill"
                        text="signin_with"
                    />
                </div>

                <div className="text-center mt-4 pt-3 border-top">
                    <p className="small text-muted mb-0">&copy; 2025 L.and.R. All rights reserved.</p>
                </div>
            </div>
        </div>
    );
};

export default Login;
