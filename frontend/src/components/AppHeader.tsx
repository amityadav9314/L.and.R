import React from 'react';
import { View, Text, TouchableOpacity, StyleSheet } from 'react-native';
// @ts-ignore
import { Ionicons } from '@expo/vector-icons';
import { useSafeAreaInsets } from 'react-native-safe-area-context';
import { useNavigation } from '../navigation/ManualRouter';
import { useAuthStore } from '../store/authStore';
import { useTheme } from '../utils/theme';
import { APP_NAME } from '../utils/constants';

export const AppHeader = () => {
    const { navigate, goBack, canGoBack } = useNavigation();
    const { logout } = useAuthStore();
    const { isDark, colors, toggleTheme } = useTheme();
    const insets = useSafeAreaInsets();

    const handleHomePress = () => {
        navigate('Home');
    };

    return (
        <View style={[
            styles.headerContainer,
            {
                backgroundColor: colors.headerBg,
                borderBottomColor: colors.headerBorder,
                paddingTop: insets.top + 10, // Add safe area inset + extra padding
            }
        ]}>
            <View style={styles.leftContainer}>
                {canGoBack && (
                    <TouchableOpacity onPress={goBack} style={styles.iconButton}>
                        <Ionicons name="arrow-back" size={24} color={colors.textPrimary} />
                    </TouchableOpacity>
                )}
                <TouchableOpacity onPress={handleHomePress} style={styles.iconButton}>
                    <Text style={{ fontSize: 24 }}>🏠</Text>
                </TouchableOpacity>
            </View>

            <TouchableOpacity onPress={handleHomePress} style={styles.centerContainer}>
                <Text style={[styles.logo, { color: colors.primary }]}>{APP_NAME}</Text>
            </TouchableOpacity>

            <View style={styles.rightContainer}>
                <TouchableOpacity onPress={toggleTheme} style={styles.iconButton}>
                    <Ionicons
                        name={isDark ? "sunny" : "moon"}
                        size={22}
                        color={colors.textPrimary}
                    />
                </TouchableOpacity>
                <TouchableOpacity onPress={logout} style={styles.logoutButton}>
                    <Text style={[styles.logout, { color: colors.error }]}>Logout</Text>
                </TouchableOpacity>
            </View>
        </View>
    );
};

const styles = StyleSheet.create({
    headerContainer: {
        flexDirection: 'row',
        justifyContent: 'space-between',
        alignItems: 'center',
        paddingTop: 10,
        paddingBottom: 15,
        paddingHorizontal: 15,
        borderBottomWidth: 1,
    },
    leftContainer: {
        flexDirection: 'row',
        alignItems: 'center',
        flex: 1,
    },
    centerContainer: {
        flex: 2,
        alignItems: 'center',
    },
    rightContainer: {
        flex: 1,
        flexDirection: 'row',
        alignItems: 'center',
        justifyContent: 'flex-end',
    },
    iconButton: {
        padding: 8,
        marginRight: 5,
    },
    logo: {
        fontSize: 24,
        fontWeight: 'bold',
        letterSpacing: -0.5,
    },
    logoutButton: {
        padding: 8,
    },
    logout: {
        fontWeight: '600',
        fontSize: 14,
    },
});
