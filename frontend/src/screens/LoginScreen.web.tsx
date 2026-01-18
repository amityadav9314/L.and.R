import React, { useEffect, useState } from 'react';
import { View, Text, StyleSheet, TouchableOpacity, ActivityIndicator } from 'react-native';
import * as WebBrowser from 'expo-web-browser';
import * as Google from 'expo-auth-session/providers/google';
import { ResponseType, makeRedirectUri } from 'expo-auth-session';
import { useAuthStore } from '../store/authStore';
import { authClient } from '../services/api';
import { GOOGLE_WEB_CLIENT_ID, GOOGLE_IOS_CLIENT_ID, GOOGLE_ANDROID_CLIENT_ID } from '../utils/config';

WebBrowser.maybeCompleteAuthSession();

export const LoginScreen = () => {
    console.log('[LoginScreen Web] Rendering...');
    const { login } = useAuthStore();
    const [isLoggingIn, setIsLoggingIn] = useState(false);
    const [error, setError] = useState<string | null>(null);

    const [request, response, promptAsync] = Google.useAuthRequest({
        clientId: GOOGLE_WEB_CLIENT_ID,
        iosClientId: GOOGLE_IOS_CLIENT_ID,
        androidClientId: GOOGLE_ANDROID_CLIENT_ID,
        responseType: ResponseType.IdToken,
        redirectUri: makeRedirectUri(),
    });

    useEffect(() => {
        if (response?.type === 'success') {
            const idToken = response.params?.id_token ||
                response.authentication?.idToken ||
                response.params?.idToken;

            if (idToken) {
                handleLogin(idToken);
            } else {
                setError('No ID token received from Google');
                setIsLoggingIn(false);
            }
        } else if (response?.type === 'error') {
            setError(`Login failed: ${response.error?.message || 'Unknown error'}`);
            setIsLoggingIn(false);
        } else if (response?.type === 'cancel') {
            setIsLoggingIn(false);
        }
    }, [response]);

    const handleLogin = async (idToken: string) => {
        setIsLoggingIn(true);
        setError(null);
        try {
            const result = await authClient.login({ googleIdToken: idToken });
            if (result.user && result.sessionToken) {
                await login(result.user, result.sessionToken);
            } else {
                setError('Invalid response from server');
                setIsLoggingIn(false);
            }
        } catch (error: any) {
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
                style={[styles.button, (!request || isLoggingIn) && styles.buttonDisabled]}
                onPress={async () => {
                    // Call promptAsync FIRST while we still have user activation
                    const result = promptAsync();
                    // THEN set loading state
                    setIsLoggingIn(true);
                    // Wait for the result
                    await result;
                }}
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
        textAlign: 'center',
        marginBottom: 40,
    },
    button: {
        backgroundColor: '#4285F4',
        paddingVertical: 12,
        paddingHorizontal: 30,
        borderRadius: 25,
        elevation: 3,
        shadowColor: '#000',
        shadowOffset: { width: 0, height: 2 },
        shadowOpacity: 0.2,
        shadowRadius: 2,
        minWidth: 200,
        alignItems: 'center',
    },
    buttonDisabled: {
        backgroundColor: '#a0cfff',
    },
    buttonText: {
        color: '#fff',
        fontSize: 16,
        fontWeight: 'bold',
    },
    errorContainer: {
        backgroundColor: '#ffebee',
        padding: 10,
        borderRadius: 5,
        marginBottom: 20,
        maxWidth: '80%',
    },
    errorText: {
        color: '#c62828',
        textAlign: 'center',
    },
});
