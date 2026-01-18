export const BYPASS_AUTH = false;
// Desktop always uses localhost (browser), even if EXPO_PUBLIC_API_URL is set for mobile
export const API_URL = import.meta.env.VITE_API_URL || import.meta.env.EXPO_PUBLIC_API_URL?.replace('10.0.2.2', 'localhost') || 'http://localhost:8080';
export const GOOGLE_WEB_CLIENT_ID = import.meta.env.VITE_GOOGLE_CLIENT_ID || "330800561912-pf7pdbfsfjicv9fe4lkkf1q130gg2952.apps.googleusercontent.com";

