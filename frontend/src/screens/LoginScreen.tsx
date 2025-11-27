import React, { useEffect, useState } from 'react';
import { View, Text, StyleSheet, TouchableOpacity, ActivityIndicator } from 'react-native';
import * as WebBrowser from 'expo-web-browser';
import * as Google from 'expo-auth-session/providers/google';
import { ResponseType } from 'expo-auth-session';
import { useAuthStore } from '../store/authStore';
import { authClient } from '../services/api';
import { GOOGLE_WEB_CLIENT_ID, GOOGLE_IOS_CLIENT_ID, GOOGLE_ANDROID_CLIENT_ID } from '../utils/config';

WebBrowser.maybeCompleteAuthSession();

export const LoginScreen = () => {
    const { login } = useAuthStore();
    const [isLoggingIn, setIsLoggingIn] = useState(false);
    const [error, setError] = useState<string | null>(null);
    
    const [request, response, promptAsync] = Google.useAuthRequest({
        clientId: GOOGLE_WEB_CLIENT_ID,
        iosClientId: GOOGLE_IOS_CLIENT_ID,
        androidClientId: GOOGLE_ANDROID_CLIENT_ID,
        responseType: ResponseType.IdToken,
    });

    useEffect(() => {
        if (response?.type === 'success') {
            console.log('[LOGIN] Google OAuth success');
            console.log('[LOGIN] Full response:', JSON.stringify(response, null, 2));
            console.log('[LOGIN] Response params:', response.params);
            
            // Try different possible locations for the ID token
            const idToken = response.params?.id_token || 
                           response.authentication?.idToken ||
                           response.params?.idToken;
            
            if (idToken) {
                console.log('[LOGIN] Found ID token, length:', idToken.length);
                handleLogin(idToken);
            } else {
                console.error('[LOGIN] No ID token in response');
                console.error('[LOGIN] Available params keys:', Object.keys(response.params || {}));
                setError('No ID token received from Google');
                setIsLoggingIn(false);
            }
        } else if (response?.type === 'error') {
            console.error('[LOGIN] OAuth error:', response.error);
            setError(`Login failed: ${response.error?.message || 'Unknown error'}`);
            setIsLoggingIn(false);
        } else if (response?.type === 'cancel') {
            console.log('[LOGIN] User cancelled OAuth');
            setIsLoggingIn(false);
        }
    }, [response]);

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

    const handleSignIn = async () => {
        setError(null);
        setIsLoggingIn(true);
        try {
            await promptAsync();
        } catch (error) {
            console.error('[LOGIN] Error opening OAuth:', error);
            setError('Failed to open login window');
            setIsLoggingIn(false);
        }
    };

    return (
        <View style={styles.container}>
            <Text style={styles.title}>Learn & Revise</Text>
            <Text style={styles.subtitle}>Master any topic with AI and Spaced Repetition</Text>

            {error && (
                <View style={styles.errorContainer}>
                    <Text style={styles.errorText}>{error}</Text>
                </View>
            )}

            <TouchableOpacity
                style={[styles.button, (!request || isLoggingIn) && styles.buttonDisabled]}
                onPress={handleSignIn}
                disabled={!request || isLoggingIn}
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
        backgroundColor: '#fff',
        padding: 20,
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
