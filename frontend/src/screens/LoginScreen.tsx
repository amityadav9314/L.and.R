import React, { useEffect, useState } from 'react';
import { View, Text, StyleSheet, TouchableOpacity, ActivityIndicator, Platform } from 'react-native';
import { GoogleSignin, statusCodes, GoogleSigninButton } from '@react-native-google-signin/google-signin';
import * as WebBrowser from 'expo-web-browser';
import Constants, { ExecutionEnvironment } from 'expo-constants';
import { makeRedirectUri } from 'expo-auth-session';
import { useAuthStore } from '../store/authStore';
import { authClient } from '../services/api';
import { GOOGLE_WEB_CLIENT_ID, GOOGLE_ANDROID_CLIENT_ID } from '../utils/config';

export const LoginScreen = () => {
    const { login } = useAuthStore();
    const [isLoggingIn, setIsLoggingIn] = useState(false);
    const [error, setError] = useState<string | null>(null);

    // Configure Google Sign-In for native platforms
    useEffect(() => {
        if (Platform.OS !== 'web') {
            GoogleSignin.configure({
                webClientId: GOOGLE_WEB_CLIENT_ID,
                offlineAccess: false,
                forceCodeForRefreshToken: false,
            });
        }
    }, []);

    // Web Google Sign-In Flow (using WebBrowser)
    const handleWebSignIn = async () => {
        setError(null);
        setIsLoggingIn(true);
        try {
            console.log('[LOGIN] Starting web Google Sign-In...');
            
            const redirectUri = makeRedirectUri({
                scheme: 'landr',
                path: 'redirect'
            });
            
            console.log('[LOGIN] Redirect URI:', redirectUri);
            
            const clientId = GOOGLE_WEB_CLIENT_ID;
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
            const result = await WebBrowser.openAuthSessionAsync(authUrl, redirectUri);
            console.log('[LOGIN] Auth Session Result:', result);

            if (result.type === 'success' && result.url) {
                const url = new URL(result.url);
                const fragment = url.hash.substring(1);
                const params = new URLSearchParams(fragment);
                const idToken = params.get('id_token');

                if (idToken) {
                    console.log('[LOGIN] Found ID token');
                    await handleLogin(idToken);
                } else {
                    setError('No ID token found in response');
                    setIsLoggingIn(false);
                }
            } else {
                console.log('[LOGIN] Auth cancelled or failed');
                setIsLoggingIn(false);
            }
        } catch (error) {
            console.error('[LOGIN] Web Sign-In error:', error);
            setError('Failed to open login window: ' + (error instanceof Error ? error.message : String(error)));
            setIsLoggingIn(false);
        }
    };

    // Native Google Sign-In Flow (using GoogleSignin SDK)
    const handleNativeSignIn = async () => {
        setError(null);
        setIsLoggingIn(true);
        try {
            console.log('[LOGIN] Starting native Google Sign-In...');
            
            await GoogleSignin.hasPlayServices();
            const userInfo = await GoogleSignin.signIn();
            console.log('[LOGIN] Sign-in successful, user:', userInfo.data?.user.email);
            
            const tokens = await GoogleSignin.getTokens();
            const idToken = tokens.idToken;
            
            if (idToken) {
                console.log('[LOGIN] Got ID token, sending to backend...');
                await handleLogin(idToken);
            } else {
                setError('No ID token received from Google');
                setIsLoggingIn(false);
            }
        } catch (error: any) {
            console.error('[LOGIN] Native Sign-In error:', error);
            
            if (error.code === statusCodes.SIGN_IN_CANCELLED) {
                setError('Sign-in was cancelled');
            } else if (error.code === statusCodes.IN_PROGRESS) {
                setError('Sign-in is already in progress');
            } else if (error.code === statusCodes.PLAY_SERVICES_NOT_AVAILABLE) {
                setError('Google Play Services not available');
            } else {
                setError('Sign-in failed: ' + (error.message || 'Unknown error'));
            }
            
            setIsLoggingIn(false);
        }
    };

    // Main sign-in handler - routes to web or native based on platform
    const handleSignIn = async () => {
        if (Platform.OS === 'web') {
            await handleWebSignIn();
        } else {
            await handleNativeSignIn();
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

            <GoogleSigninButton
                style={{ width: 192, height: 48 }}
                size={GoogleSigninButton.Size.Wide}
                color={GoogleSigninButton.Color.Dark}
                onPress={handleSignIn}
                disabled={isLoggingIn}
            />
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
