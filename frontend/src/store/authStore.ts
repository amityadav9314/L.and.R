import { create } from 'zustand';
import { Platform } from 'react-native';
import { UserProfile } from '../../proto/backend/proto/auth/auth';
import { queryClient } from '../../App';

interface AuthState {
    user: UserProfile | null;
    token: string | null;
    isLoading: boolean;
    login: (user: UserProfile, token: string) => Promise<void>;
    logout: () => Promise<void>;
    restoreSession: () => Promise<void>;
}

// Storage helpers that work on both web and native
const storage = {
    setItem: async (key: string, value: string) => {
        if (Platform.OS === 'web') {
            localStorage.setItem(key, value);
        } else {
            const SecureStore = await import('expo-secure-store');
            await SecureStore.setItemAsync(key, value);
        }
    },
    getItem: async (key: string): Promise<string | null> => {
        if (Platform.OS === 'web') {
            return localStorage.getItem(key);
        } else {
            const SecureStore = await import('expo-secure-store');
            return await SecureStore.getItemAsync(key);
        }
    },
    deleteItem: async (key: string) => {
        if (Platform.OS === 'web') {
            localStorage.removeItem(key);
        } else {
            const SecureStore = await import('expo-secure-store');
            await SecureStore.deleteItemAsync(key);
        }
    }
};

export const useAuthStore = create<AuthState>((set) => ({
    user: null,
    token: null,
    isLoading: true,
    login: async (user, token) => {
        console.log('[AUTH] Login - clearing cache and setting new user');
        // Clear all cached data when logging in with a new account
        queryClient.clear();
        await storage.setItem('auth_token', token);
        await storage.setItem('user_profile', JSON.stringify(user));
        set({ user, token, isLoading: false });
    },
    logout: async () => {
        console.log('[AUTH] Logout - clearing cache and storage');
        // Clear all cached data when logging out
        queryClient.clear();
        await storage.deleteItem('auth_token');
        await storage.deleteItem('user_profile');
        set({ user: null, token: null, isLoading: false });
    },
    restoreSession: async () => {
        console.log('[AUTH] Starting session restore...');
        try {
            const token = await storage.getItem('auth_token');
            const userStr = await storage.getItem('user_profile');
            console.log('[AUTH] Retrieved from storage - token:', !!token, 'user:', !!userStr);

            if (token && userStr) {
                const user = JSON.parse(userStr) as UserProfile;
                console.log('[AUTH] Restoring existing session for user:', user.email);
                set({ user, token, isLoading: false });
            } else {
                console.log('[AUTH] No session found, no bypass, showing login');
                set({ isLoading: false });
            }
        } catch (e) {
            console.error("[AUTH] Failed to restore session", e);
            set({ isLoading: false });
        }
    },
}));
