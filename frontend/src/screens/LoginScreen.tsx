import React, { useEffect, useState } from 'react';
import { View, Text, StyleSheet, TouchableOpacity, ActivityIndicator, Platform, Image } from 'react-native';
import { useSafeAreaInsets } from 'react-native-safe-area-context';
import { GoogleSignin, statusCodes, GoogleSigninButton } from '@react-native-google-signin/google-signin';
import * as WebBrowser from 'expo-web-browser';
import Constants, { ExecutionEnvironment } from 'expo-constants';
import { makeRedirectUri } from 'expo-auth-session';
import { useAuthStore } from '../store/authStore';
import { authClient } from '../services/api';
import { GOOGLE_WEB_CLIENT_ID, GOOGLE_ANDROID_CLIENT_ID } from '../utils/config';
import { useTheme, ThemeColors } from '../utils/theme';

export const LoginScreen = () => {
    const { login } = useAuthStore();
    const { colors } = useTheme();
    const insets = useSafeAreaInsets();
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

    // Web Google Sign-In Flow
    const handleWebSignIn = async () => {
        setError(null);
        setIsLoggingIn(true);
        try {
            console.log('[LOGIN] Starting web Google Sign-In...');

            const redirectUri = makeRedirectUri({
                scheme: 'landr',
                path: 'redirect'
            });

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

            const result = await WebBrowser.openAuthSessionAsync(authUrl, redirectUri);

            if (result.type === 'success' && result.url) {
                const url = new URL(result.url);
                const fragment = url.hash.substring(1);
                const params = new URLSearchParams(fragment);
                const idToken = params.get('id_token');

                if (idToken) {
                    await handleLogin(idToken);
                } else {
                    setError('No ID token found in response');
                    setIsLoggingIn(false);
                }
            } else {
                setIsLoggingIn(false);
            }
        } catch (error) {
            console.error('[LOGIN] Web Sign-In error:', error);
            setError('Failed to open login window: ' + (error instanceof Error ? error.message : String(error)));
            setIsLoggingIn(false);
        }
    };

    // Native Google Sign-In Flow
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

    const styles = createStyles(colors);

    return (
        <View style={[styles.container, { paddingBottom: insets.bottom }]}>
            {/* Logo */}
            <View style={styles.logoContainer}>
                <Image
                    source={require('../../assets/icon.png')}
                    style={styles.logoIcon}
                    resizeMode="contain"
                />
                <Text style={[styles.logoText, { color: colors.primary }]}>LandR</Text>
            </View>

            <Text style={styles.title}>Learn & Revise</Text>
            <Text style={styles.subtitle}>Master any topic with AI and Spaced Repetition</Text>

            {error && (
                <View style={styles.errorContainer}>
                    <Text style={styles.errorText}>{typeof error === 'string' ? error : JSON.stringify(error)}</Text>
                </View>
            )}

            <GoogleSigninButton
                style={{ width: 280, height: 56 }}
                size={GoogleSigninButton.Size.Wide}
                color={GoogleSigninButton.Color.Dark}
                onPress={handleSignIn}
                disabled={isLoggingIn}
            />
        </View>
    );
};

const createStyles = (colors: ThemeColors) => StyleSheet.create({
    container: {
        flex: 1,
        justifyContent: 'center',
        alignItems: 'center',
        padding: 20,
        backgroundColor: colors.background,
    },
    logoContainer: {
        flexDirection: 'row',
        alignItems: 'center',
        marginBottom: 20,
        gap: 8,
    },
    logoIcon: {
        width: 64,
        height: 64,
        borderRadius: 12,
    },
    logoText: {
        fontSize: 42,
        fontWeight: 'bold',
        letterSpacing: 2,
    },
    title: {
        fontSize: 24,
        fontWeight: 'bold',
        marginBottom: 10,
        color: colors.textPrimary,
    },
    subtitle: {
        fontSize: 16,
        color: colors.textSecondary,
        marginBottom: 40,
        textAlign: 'center',
    },
    errorContainer: {
        backgroundColor: colors.errorBg,
        padding: 12,
        borderRadius: 8,
        marginBottom: 20,
        maxWidth: 400,
    },
    errorText: {
        color: colors.error,
        fontSize: 14,
        textAlign: 'center',
    },
    button: {
        backgroundColor: colors.primary,
        paddingVertical: 12,
        paddingHorizontal: 24,
        borderRadius: 8,
        elevation: 2,
        minWidth: 200,
        alignItems: 'center',
    },
    buttonDisabled: {
        backgroundColor: colors.paginationDisabledBg,
    },
    buttonText: {
        color: colors.textInverse,
        fontSize: 16,
        fontWeight: '600',
    },
});
