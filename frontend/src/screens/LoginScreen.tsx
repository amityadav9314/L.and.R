import React, { useEffect, useState } from 'react';
import { View, Text, StyleSheet, TouchableOpacity, ActivityIndicator, Button, Platform } from 'react-native';
import * as WebBrowser from 'expo-web-browser';
import * as Google from 'expo-auth-session/providers/google';
import Constants, { ExecutionEnvironment } from 'expo-constants';
import { ResponseType, makeRedirectUri } from 'expo-auth-session';
import { useAuthStore } from '../store/authStore';
import { authClient } from '../services/api';
import { GOOGLE_WEB_CLIENT_ID, GOOGLE_IOS_CLIENT_ID, GOOGLE_ANDROID_CLIENT_ID } from '../utils/config';

// WebBrowser.maybeCompleteAuthSession();

export const LoginScreen = () => {
    const { login } = useAuthStore();
    const [isLoggingIn, setIsLoggingIn] = useState(false);
    const [error, setError] = useState<string | null>(null);

    // Manual Auth Flow
    const handleSignIn = async () => {
        setError(null);
        setIsLoggingIn(true);
        try {
            const isExpoGo = Constants.executionEnvironment === ExecutionEnvironment.StoreClient;
            // Wait, StoreClient is Expo Go. Standalone is Standalone.
            // Actually, let's just check if hostUri exists. If hostUri exists, we are likely in Expo Go or Dev Client.
            // But Dev Client supports custom schemes.
            // Let's stick to appOwnership if it works, or just assume Expo Go if hostUri is present?

            // Better check:
            const isStoreClient = Constants.executionEnvironment === 'storeClient'; // Expo Go

            let redirectUri;
            let clientId;

            if (isStoreClient) {
                // Expo Go: Use Tunnel HTTPS
                const hostUri = Constants.expoConfig?.hostUri;
                redirectUri = hostUri ? `https://${hostUri}` : makeRedirectUri();
                console.log('[LOGIN] Expo Go detected. Using Tunnel URL.');
            } else {
                // Production/EAS Build: Use Expo's Auth Proxy
                // This works with Web Client ID and doesn't require custom scheme registration
                redirectUri = makeRedirectUri({
                    scheme: 'landr',
                    path: 'redirect'
                });
                console.log('[LOGIN] Standalone/EAS Build detected. Using Expo Auth Proxy.');
            }

            console.log('[LOGIN] Redirect URI:', redirectUri);

            // Use appropriate Client ID based on platform
            if (Platform.OS === 'android') {
                clientId = GOOGLE_ANDROID_CLIENT_ID;
                console.log('[LOGIN] Using Android Client ID');
            } else if (Platform.OS === 'ios') {
                clientId = GOOGLE_IOS_CLIENT_ID;
                console.log('[LOGIN] Using iOS Client ID');
            } else {
                clientId = GOOGLE_WEB_CLIENT_ID;
                console.log('[LOGIN] Using Web Client ID');
            }
            const scope = encodeURIComponent('openid profile email');
            const responseType = 'id_token';
            const nonce = Math.random().toString(36).substring(7);

            const authUrl = `https://accounts.google.com/o/oauth2/v2/auth` +
                `?client_id=${clientId}` +
                `&redirect_uri=${encodeURIComponent(redirectUri)}` +
                `&response_type=${responseType}` +
                `&scope=${scope}` +
                `&nonce=${nonce}`;

            console.log('[LOGIN] Opening Auth Session...');
            // Force Chrome to avoid "disallowed user agent" error
            const result = await WebBrowser.openAuthSessionAsync(authUrl, redirectUri, {
                browserPackage: 'com.android.chrome'
            });
            console.log('[LOGIN] Auth Session Result:', result);

            if (result.type === 'success' && result.url) {
                // Extract id_token from URL fragment
                const url = new URL(result.url);
                const fragment = url.hash.substring(1); // remove #
                const params = new URLSearchParams(fragment);
                const idToken = params.get('id_token');

                if (idToken) {
                    console.log('[LOGIN] Found ID token');
                    handleLogin(idToken);
                } else {
                    setError('No ID token found in response');
                    setIsLoggingIn(false);
                }
            } else {
                console.log('[LOGIN] Auth cancelled or failed');
                setIsLoggingIn(false);
            }
        } catch (error) {
            console.error('[LOGIN] Error opening OAuth:', error);
            setError('Failed to open login window: ' + (error instanceof Error ? error.message : String(error)));
            setIsLoggingIn(false);
        }
    };

    const handleLogin = async (idToken: string) => {
        setIsLoggingIn(true);
        setError(null);

        try {
            console.log('[LOGIN] Sending ID token to backend...');
            const result = await authClient.login({ googleIdToken: idToken });

            console.log('[LOGIN] Backend response received');
            if (result.user && result.sessionToken) {
                console.log('[LOGIN] Login successful, user:', result.user.email);
                await login(result.user, result.sessionToken);
            } else {
                console.error('[LOGIN] Invalid response from backend');
                setError('Invalid response from server');
                setIsLoggingIn(false);
            }
        } catch (error: any) {
            console.error('[LOGIN] Backend error:', error);
            setError(`Login failed: ${error.message || 'Server error'}`);
            setIsLoggingIn(false);
        }
    };

    return (
        <View style={styles.container}>
            <Text style={styles.title}>Learn & Revise</Text>
            <Text style={styles.subtitle}>Master any topic with AI and Spaced Repetition</Text>

            {error && (
                <View style={styles.errorContainer}>
                    <Text style={styles.errorText}>{typeof error === 'string' ? error : JSON.stringify(error)}</Text>
                </View>
            )}

            <TouchableOpacity
                style={[styles.button, isLoggingIn && styles.buttonDisabled]}
                onPress={handleSignIn}
                disabled={isLoggingIn}
            >
                {isLoggingIn ? (
                    <ActivityIndicator color="#fff" />
                ) : (
                    <Text style={styles.buttonText}>Sign in with Google</Text>
                )}
            </TouchableOpacity>
        </View>
    );
};

const styles = StyleSheet.create({
    container: {
        flex: 1,
        justifyContent: 'center',
        alignItems: 'center',
        padding: 20,
        backgroundColor: '#f5f5f5',
    },
    title: {
        fontSize: 32,
        fontWeight: 'bold',
        marginBottom: 10,
        color: '#333',
    },
    subtitle: {
        fontSize: 16,
        color: '#666',
        marginBottom: 40,
        textAlign: 'center',
    },
    errorContainer: {
        backgroundColor: '#fee',
        padding: 12,
        borderRadius: 8,
        marginBottom: 20,
        maxWidth: 400,
    },
    errorText: {
        color: '#c00',
        fontSize: 14,
        textAlign: 'center',
    },
    button: {
        backgroundColor: '#4285F4',
        paddingVertical: 12,
        paddingHorizontal: 24,
        borderRadius: 8,
        elevation: 2,
        minWidth: 200,
        alignItems: 'center',
    },
    buttonDisabled: {
        backgroundColor: '#aaa',
    },
    buttonText: {
        color: '#fff',
        fontSize: 16,
        fontWeight: '600',
    },
});
