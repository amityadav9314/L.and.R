import React from 'react';
import { View, Text, TouchableOpacity, StyleSheet } from 'react-native';
// @ts-ignore
import { Ionicons } from '@expo/vector-icons';
import { useSafeAreaInsets } from 'react-native-safe-area-context';
import { useNavigation } from '../navigation/ManualRouter';
import { useAuthStore } from '../store/authStore';
import { useTheme } from '../utils/theme';
import { useFilterStore } from '../store/filterStore';
import { APP_NAME } from '../utils/constants';

export const AppHeader = () => {
    const { goBack, canGoBack, navigate, currentScreen } = useNavigation();
    const { colors, toggleTheme, isDark } = useTheme();
    const { user } = useAuthStore();
    const { resetAll } = useFilterStore();
    const insets = useSafeAreaInsets();

    // Screens that should not show a back button (top-level tabs)
    const isMainScreen = ['Home', 'DailyFeed', 'AddMaterial', 'Settings', 'Review'].includes(currentScreen);
    const showBack = canGoBack && !isMainScreen;

    const handleLogoPress = () => {
        resetAll();
        navigate('Home');
    };

    return (
        <View style={[
            styles.headerContainer,
            {
                backgroundColor: colors.headerBg,
                borderBottomColor: colors.headerBorder,
                paddingTop: insets.top + 5,
            }
        ]}>
            <View style={styles.content}>
                <TouchableOpacity
                    style={styles.leftSection}
                    onPress={handleLogoPress}
                    activeOpacity={0.7}
                >
                    {showBack && (
                        <TouchableOpacity onPress={goBack} style={styles.backButton}>
                            <Ionicons name="arrow-back" size={20} color={colors.textPrimary} />
                        </TouchableOpacity>
                    )}
                    <View style={styles.logoContainer}>
                        <Text style={[styles.logo, { color: colors.primary }]}>{APP_NAME}</Text>
                        {user?.isPro && (
                            <View style={styles.proBadge}>
                                <Text style={styles.proText}>PRO</Text>
                            </View>
                        )}
                    </View>
                    {user && (
                        <Text style={[styles.welcomeText, { color: colors.textSecondary }]}>
                            Welcome, {user.name.split(' ')[0]}
                        </Text>
                    )}
                </TouchableOpacity>

                <TouchableOpacity onPress={toggleTheme} style={styles.themeToggle}>
                    <Ionicons
                        name={isDark ? "sunny" : "moon"}
                        size={20}
                        color={colors.textPrimary}
                    />
                </TouchableOpacity>
            </View>
        </View>
    );
};

const styles = StyleSheet.create({
    headerContainer: {
        paddingBottom: 4,
        paddingHorizontal: 3,
        borderBottomWidth: 1,
    },
    content: {
        flexDirection: 'row',
        alignItems: 'center',
        justifyContent: 'space-between',
        height: 36,
    },
    leftSection: {
        flexDirection: 'row',
        alignItems: 'center',
        gap: 8,
    },
    backButton: {
        padding: 4,
        marginRight: 2,
    },
    logoContainer: {
        flexDirection: 'row',
        alignItems: 'center',
        gap: 6,
    },
    logo: {
        fontSize: 26,
        fontWeight: 'bold',
        letterSpacing: -0.5,
    },
    proBadge: {
        backgroundColor: '#FFB800',
        borderRadius: 8,
        paddingHorizontal: 6,
        paddingVertical: 2,
        borderWidth: 1,
        borderColor: '#FFA000',
    },
    proText: {
        color: '#000000',
        fontSize: 10,
        fontWeight: '700',
        letterSpacing: 0.5,
    },
    welcomeText: {
        fontSize: 14,
        fontWeight: '500',
        marginLeft: 4,
        marginTop: 6, // Align slightly with large logo
    },
    themeToggle: {
        padding: 6,
    },
});
