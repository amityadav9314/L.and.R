import React, { useState, useEffect } from 'react';
import { View, Text, StyleSheet, TouchableOpacity, ScrollView, TextInput, Switch, ActivityIndicator, Alert } from 'react-native';
import { useSafeAreaInsets } from 'react-native-safe-area-context';
import { useQueryClient } from '@tanstack/react-query';
import { useAuthStore } from '../store/authStore';
import { useTheme, ThemeColors } from '../utils/theme';
import { AppHeader } from '../components/AppHeader';
import { feedClient } from '../services/api';

export const SettingsScreen = () => {
    const { user, logout } = useAuthStore();
    const { colors, isDark, toggleTheme } = useTheme();
    const insets = useSafeAreaInsets();
    const queryClient = useQueryClient();
    const styles = createStyles(colors);

    // Feed Preferences State
    const [feedEnabled, setFeedEnabled] = useState(false);
    const [interestPrompt, setInterestPrompt] = useState('');
    const [evalPrompt, setEvalPrompt] = useState('');
    const [loading, setLoading] = useState(true);
    const [saving, setSaving] = useState(false);

    // Load feed preferences on mount
    useEffect(() => {
        const loadPreferences = async () => {
            try {
                const prefs = await feedClient.getFeedPreferences();
                setFeedEnabled(prefs.feedEnabled);
                setInterestPrompt(prefs.interestPrompt);
                setEvalPrompt(prefs.feedEvalPrompt);
            } catch (error) {
                console.log('Failed to load feed preferences:', error);
            } finally {
                setLoading(false);
            }
        };
        loadPreferences();
    }, []);

    // Save feed preferences
    const saveFeedPreferences = async () => {
        setSaving(true);
        try {
            await feedClient.updateFeedPreferences({
                feedEnabled,
                interestPrompt,
                feedEvalPrompt: evalPrompt,
            });
            Alert.alert('Success ✅', 'Your feed preferences have been saved!');
        } catch (error) {
            console.log('Failed to save feed preferences:', error);
            Alert.alert('Error', 'Failed to save preferences. Please try again.');
        } finally {
            setSaving(false);
        }
    };

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

                {/* Daily AI Feed Section */}
                <View style={styles.section}>
                    <Text style={styles.sectionTitle}>Daily AI Feed 📰</Text>
                    <View style={styles.card}>
                        {loading ? (
                            <ActivityIndicator color={colors.primary} />
                        ) : (
                            <>
                                <View style={styles.row}>
                                    <Text style={styles.label}>Enable Daily Feed</Text>
                                    <Switch
                                        value={feedEnabled}
                                        onValueChange={async (value) => {
                                            setFeedEnabled(value);
                                            // Auto-save when toggling
                                            try {
                                                await feedClient.updateFeedPreferences({
                                                    feedEnabled: value,
                                                    interestPrompt,
                                                    feedEvalPrompt: evalPrompt,
                                                });
                                                // Invalidate cache so DailyFeedScreen refetches
                                                queryClient.invalidateQueries({ queryKey: ['feedPreferences'] });
                                                Alert.alert(
                                                    value ? 'Feed Enabled ✅' : 'Feed Disabled',
                                                    value ? 'Daily articles will be generated based on your interests.' : 'Daily feed has been turned off.'
                                                );
                                            } catch (error) {
                                                console.log('Failed to toggle feed:', error);
                                                setFeedEnabled(!value); // Revert on error
                                                Alert.alert('Error', 'Failed to update feed setting.');
                                            }
                                        }}
                                        trackColor={{ false: colors.border, true: colors.primary }}
                                        thumbColor={feedEnabled ? colors.card : colors.textSecondary}
                                    />
                                </View>
                                {feedEnabled && (
                                    <>
                                        <View style={styles.divider} />
                                        <Text style={styles.inputLabel}>Your Interest Prompt</Text>
                                        <Text style={styles.inputHint}>
                                            Describe topics you want daily articles about. E.g., "Latest AI research papers, TypeScript best practices, React Native tips"
                                        </Text>
                                        <TextInput
                                            style={styles.textInput}
                                            placeholder="Enter your interests..."
                                            placeholderTextColor={colors.textSecondary}
                                            value={interestPrompt}
                                            onChangeText={setInterestPrompt}
                                            multiline
                                            numberOfLines={3}
                                        />

                                        <View style={[styles.divider, { marginVertical: 16 }]} />

                                        <Text style={styles.inputLabel}>Evaluation Criteria (Optional)</Text>
                                        <Text style={styles.inputHint}>
                                            Define how the AI should score articles. E.g., "Must be technical and detailed", "Avoid marketing fluff", "Score high for tutorials".
                                        </Text>
                                        <TextInput
                                            style={styles.textInput}
                                            placeholder="Enter evaluation criteria..."
                                            placeholderTextColor={colors.textSecondary}
                                            value={evalPrompt}
                                            onChangeText={setEvalPrompt}
                                            multiline
                                            numberOfLines={2}
                                        />

                                        <TouchableOpacity
                                            style={[styles.saveButton, saving && styles.saveButtonDisabled]}
                                            onPress={saveFeedPreferences}
                                            disabled={saving}
                                        >
                                            {saving ? (
                                                <ActivityIndicator color="#fff" size="small" />
                                            ) : (
                                                <Text style={styles.saveButtonText}>Save Preferences</Text>
                                            )}
                                        </TouchableOpacity>
                                    </>
                                )}
                            </>
                        )}
                    </View>
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
    divider: {
        height: 1,
        backgroundColor: colors.border,
        marginVertical: 12,
    },
    inputLabel: {
        fontSize: 14,
        fontWeight: '600',
        color: colors.textPrimary,
        marginBottom: 4,
    },
    inputHint: {
        fontSize: 12,
        color: colors.textSecondary,
        marginBottom: 8,
    },
    textInput: {
        backgroundColor: colors.background,
        borderRadius: 8,
        padding: 12,
        fontSize: 14,
        color: colors.textPrimary,
        borderWidth: 1,
        borderColor: colors.border,
        minHeight: 80,
        textAlignVertical: 'top',
    },
    saveButton: {
        backgroundColor: colors.primary,
        borderRadius: 8,
        padding: 12,
        alignItems: 'center',
        marginTop: 12,
    },
    saveButtonDisabled: {
        opacity: 0.6,
    },
    saveButtonText: {
        color: '#fff',
        fontWeight: '600',
        fontSize: 14,
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
