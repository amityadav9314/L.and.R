import React from 'react';
import { View, Text, StyleSheet, TouchableOpacity, Platform } from 'react-native';
// @ts-ignore
import { Ionicons } from '@expo/vector-icons';
import { useSafeAreaInsets } from 'react-native-safe-area-context';
import { useNavigation } from '../navigation/ManualRouter';
import { useTheme, ThemeColors } from '../utils/theme';
import { useFilterStore } from '../store/filterStore';

export const BottomNavBar = () => {
    const { currentScreen, navigate } = useNavigation();
    const { colors } = useTheme();
    const insets = useSafeAreaInsets();
    const { onlyDue, setOnlyDue } = useFilterStore();
    const styles = createStyles(colors, insets.bottom);

    // Don't show on login screen
    if (currentScreen === 'Login') {
        return null;
    }

    const navItems = [
        { name: 'Home', icon: '🏠', label: 'Home', filter: true, activeColor: '#FF6B6B' },
        { name: 'Vault', icon: '🏛️', label: 'Vault', filter: false, activeColor: '#4ECDC4' },
        { name: 'DailyFeed', icon: '📰', label: 'Feed', activeColor: '#9B59B6' },
        { name: 'AddMaterial', icon: '➕', label: 'Add', activeColor: '#45B7D1' },
        { name: 'Settings', icon: '⚙️', label: 'Settings', activeColor: '#FFD93D' },
    ];

    return (
        <View style={styles.container}>
            {navItems.map((item) => {
                const isActive = item.name === 'Vault'
                    ? (currentScreen === 'Home' && !onlyDue)
                    : (item.name === 'Home' ? (currentScreen === 'Home' && onlyDue) : currentScreen === item.name);

                const activeColor = item.activeColor || colors.primary;

                const handlePress = () => {
                    if (item.name === 'Home' || item.name === 'Vault') {
                        setOnlyDue(item.filter as boolean);
                        navigate('Home');
                    } else {
                        navigate(item.name as any);
                    }
                };

                return (
                    <TouchableOpacity
                        key={item.label}
                        style={styles.navItem}
                        onPress={handlePress}
                    >
                        <View style={[
                            styles.iconContainer,
                            isActive && { backgroundColor: activeColor + '15' } // Very light background for active state
                        ]}>
                            <Text style={[
                                styles.emojiIcon,
                                !isActive && { opacity: 0.6, grayscale: 1 } as any // Slight fade for inactive emojis
                            ]}>
                                {item.icon}
                            </Text>
                        </View>
                        <Text style={[
                            styles.navLabel,
                            { color: isActive ? activeColor : colors.textSecondary }
                        ]}>
                            {item.label}
                        </Text>
                    </TouchableOpacity>
                );
            })}
        </View>
    );
};

const createStyles = (colors: ThemeColors, bottomInset: number) => StyleSheet.create({
    container: {
        flexDirection: 'row',
        backgroundColor: colors.card,
        borderTopWidth: 1,
        borderTopColor: colors.border,
        paddingBottom: bottomInset > 0 ? bottomInset : 10,
        paddingTop: 10,
        position: 'absolute',
        bottom: 0,
        left: 0,
        right: 0,
        elevation: 8,
        shadowColor: '#000',
        shadowOffset: { width: 0, height: -2 },
        shadowOpacity: 0.1,
        shadowRadius: 4,
    },
    navItem: {
        flex: 1,
        alignItems: 'center',
        justifyContent: 'center',
    },
    iconContainer: {
        padding: 6,
        borderRadius: 12,
        marginBottom: 2,
    },
    emojiIcon: {
        fontSize: 22,
    },
    navLabel: {
        fontSize: 10,
        fontWeight: 'bold',
    },
});
