import { create } from 'zustand';
import { UserProfile } from '../proto/backend/proto/auth/auth.ts';

interface AuthState {
    user: UserProfile | null;
    token: string | null;
    isLoading: boolean;
    login: (user: UserProfile, token: string) => Promise<void>;
    logout: () => Promise<void>;
    restoreSession: () => Promise<void>;
}

export const useAuthStore = create<AuthState>((set) => ({
    user: null,
    token: null,
    isLoading: true,
    login: async (user, token) => {
        console.log('[AUTH] Login - setting user');
        localStorage.setItem('auth_token', token);
        localStorage.setItem('user_profile', JSON.stringify(user));
        set({ user, token, isLoading: false });
    },
    logout: async () => {
        console.log('[AUTH] Logout - clearing storage');
        localStorage.removeItem('auth_token');
        localStorage.removeItem('user_profile');
        set({ user: null, token: null, isLoading: false });
    },
    restoreSession: async () => {
        console.log('[AUTH] Starting session restore...');
        try {
            const token = localStorage.getItem('auth_token');
            const userStr = localStorage.getItem('user_profile');

            if (token && userStr) {
                const user = JSON.parse(userStr) as UserProfile;
                console.log('[AUTH] Restoring existing session for user:', user.email);
                set({ user, token, isLoading: false });
            } else {
                set({ isLoading: false });
            }
        } catch (e) {
            console.error("[AUTH] Failed to restore session", e);
            set({ isLoading: false });
        }
    },
}));
