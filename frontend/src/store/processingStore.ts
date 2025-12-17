import { create } from 'zustand';
import { Platform } from 'react-native';

const PROCESSING_KEY = 'material_processing';
const MAX_PROCESSING_MS = 5 * 60 * 1000; // 5 minutes max

interface ProcessingState {
    isProcessing: boolean;
    startedAt: number | null;
    startProcessing: () => Promise<void>;
    stopProcessing: () => Promise<void>;
    checkProcessing: () => Promise<boolean>;
}

// Storage helpers (same pattern as authStore)
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

export const useProcessingStore = create<ProcessingState>((set, get) => ({
    isProcessing: false,
    startedAt: null,

    startProcessing: async () => {
        const now = Date.now();
        await storage.setItem(PROCESSING_KEY, now.toString());
        set({ isProcessing: true, startedAt: now });
        console.log('[PROCESSING] Started processing at', new Date(now).toISOString());
    },

    stopProcessing: async () => {
        await storage.deleteItem(PROCESSING_KEY);
        set({ isProcessing: false, startedAt: null });
        console.log('[PROCESSING] Processing complete');
    },

    checkProcessing: async () => {
        const storedTime = await storage.getItem(PROCESSING_KEY);
        if (!storedTime) {
            set({ isProcessing: false, startedAt: null });
            return false;
        }

        const startedAt = parseInt(storedTime, 10);
        const elapsed = Date.now() - startedAt;

        // If more than 5 min, auto-clear
        if (elapsed > MAX_PROCESSING_MS) {
            console.log('[PROCESSING] Timeout exceeded, clearing stale processing state');
            await storage.deleteItem(PROCESSING_KEY);
            set({ isProcessing: false, startedAt: null });
            return false;
        }

        // Still processing
        set({ isProcessing: true, startedAt });
        console.log('[PROCESSING] Processing in progress, elapsed:', Math.round(elapsed / 1000), 's');
        return true;
    },
}));
