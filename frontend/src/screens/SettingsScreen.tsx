import React from 'react';
import { View, Text, StyleSheet, TouchableOpacity, ScrollView } from 'react-native';
import { useSafeAreaInsets } from 'react-native-safe-area-context';
import { useAuthStore } from '../store/authStore';
import { useTheme, ThemeColors } from '../utils/theme';
import { AppHeader } from '../components/AppHeader';

export const SettingsScreen = () => {
    const { user, logout } = useAuthStore();
    const { colors, isDark, toggleTheme } = useTheme();
    const insets = useSafeAreaInsets();
    const styles = createStyles(colors);

    return (
        <View style={styles.container}>
            <ScrollView contentContainerStyle={[styles.content, { paddingBottom: insets.bottom + 80 }]}>
                <AppHeader />
                <Text style={styles.title}>Settings</Text>

                <View style={styles.section}>
                    <Text style={styles.sectionTitle}>Account</Text>
                    <View style={styles.card}>
                        <View style={styles.row}>
                            <Text style={styles.label}>Email</Text>
                            <Text style={styles.value}>{user?.email || 'Not logged in'}</Text>
                        </View>
                    </View>
                </View>

                <View style={styles.section}>
                    <Text style={styles.sectionTitle}>Appearance</Text>
                    <TouchableOpacity style={styles.card} onPress={toggleTheme}>
                        <View style={styles.row}>
                            <Text style={styles.label}>Dark Mode</Text>
                            <Text style={styles.value}>{isDark ? 'On' : 'Off'}</Text>
                        </View>
                    </TouchableOpacity>
                </View>

                <View style={styles.section}>
                    <TouchableOpacity style={[styles.card, styles.logoutCard]} onPress={logout}>
                        <Text style={styles.logoutText}>Logout</Text>
                    </TouchableOpacity>
                </View>
            </ScrollView>
        </View>
    );
};

const createStyles = (colors: ThemeColors) => StyleSheet.create({
    container: {
        flex: 1,
        backgroundColor: colors.background,
    },
    content: {
        paddingHorizontal: 3,
    },
    title: {
        fontSize: 28,
        fontWeight: 'bold',
        color: colors.textPrimary,
        marginBottom: 24,
        paddingTop: 16,
    },
    section: {
        marginBottom: 24,
    },
    sectionTitle: {
        fontSize: 14,
        fontWeight: '600',
        color: colors.textSecondary,
        marginBottom: 8,
        textTransform: 'uppercase',
        letterSpacing: 1,
    },
    card: {
        backgroundColor: colors.card,
        borderRadius: 12,
        padding: 16,
        elevation: 2,
        shadowColor: '#000',
        shadowOffset: { width: 0, height: 1 },
        shadowOpacity: 0.1,
        shadowRadius: 2,
    },
    row: {
        flexDirection: 'row',
        justifyContent: 'space-between',
        alignItems: 'center',
    },
    label: {
        fontSize: 16,
        color: colors.textPrimary,
    },
    value: {
        fontSize: 16,
        color: colors.textSecondary,
    },
    logoutCard: {
        alignItems: 'center',
        borderWidth: 1,
        borderColor: colors.error,
    },
    logoutText: {
        color: colors.error,
        fontSize: 16,
        fontWeight: '600',
    },
});
