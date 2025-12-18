import React, { useState } from 'react';
import { View, Text, StyleSheet, TouchableOpacity, ActivityIndicator, ScrollView, Alert } from 'react-native';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { useNavigation, useRoute } from '../navigation/ManualRouter';
import { useSafeAreaInsets } from 'react-native-safe-area-context';
import { learningClient } from '../services/api';
import { Flashcard } from '../../proto/backend/proto/learning/learning';
import { AppHeader } from '../components/AppHeader';
import { useTheme, ThemeColors } from '../utils/theme';

export const ReviewScreen = () => {
    const navigation = useNavigation();
    const route = useRoute();
    const { flashcardId } = route.params as { flashcardId: string };
    const queryClient = useQueryClient();
    const { colors } = useTheme();
    const insets = useSafeAreaInsets();
    const [showAnswer, setShowAnswer] = useState(false);

    // Find the flashcard from any material's cache
    const flashcard = React.useMemo(() => {
        const queryCache = queryClient.getQueryCache();
        const queries = queryCache.findAll({ queryKey: ['dueFlashcards'] });

        for (const query of queries) {
            const data = query.state.data as any;
            if (Array.isArray(data)) {
                const found = data.find((f: any) => f.id === flashcardId);
                if (found) return found;
            }
        }
        return null;
    }, [flashcardId, queryClient]);

    const mutation = useMutation({
        mutationFn: async () => {
            return await learningClient.completeReview({ flashcardId });
        },
        onSuccess: () => {
            Alert.alert('Great job!', 'Review completed.');
            queryClient.invalidateQueries({ queryKey: ['dueFlashcards'] });
            navigation.goBack();
        },
        onError: (error) => {
            console.error(error);
            Alert.alert('Error', 'Failed to complete review.');
        },
    });

    const styles = createStyles(colors);

    if (!flashcard) {
        return (
            <View style={styles.container}>
                <AppHeader />
                <View style={styles.center}>
                    <Text style={styles.notFoundText}>Flashcard not found</Text>
                </View>
            </View>
        );
    }

    return (
        <View style={[styles.container, { paddingBottom: insets.bottom }]}>
            <AppHeader />
            <View style={styles.contentContainer}>
                <View style={styles.card}>
                    <Text style={styles.label}>Question</Text>
                    <Text style={styles.text}>{flashcard.question}</Text>

                    {showAnswer && (
                        <View style={styles.answerContainer}>
                            <View style={styles.divider} />
                            <Text style={styles.label}>Answer</Text>
                            <Text style={styles.text}>{flashcard.answer}</Text>
                        </View>
                    )}
                </View>

                {!showAnswer ? (
                    <TouchableOpacity
                        style={styles.button}
                        onPress={() => setShowAnswer(true)}
                    >
                        <Text style={styles.buttonText}>Show Answer</Text>
                    </TouchableOpacity>
                ) : (
                    <TouchableOpacity
                        style={[styles.button, styles.completeButton]}
                        onPress={() => mutation.mutate()}
                        disabled={mutation.isPending}
                    >
                        {mutation.isPending ? (
                            <ActivityIndicator color={colors.textInverse} />
                        ) : (
                            <Text style={styles.buttonText}>Complete Review</Text>
                        )}
                    </TouchableOpacity>
                )}
            </View>
        </View>
    );
};

const createStyles = (colors: ThemeColors) => StyleSheet.create({
    container: {
        flex: 1,
        backgroundColor: colors.background,
    },
    contentContainer: {
        flex: 1,
        padding: 20,
        justifyContent: 'center',
    },
    center: {
        flex: 1,
        justifyContent: 'center',
        alignItems: 'center',
    },
    notFoundText: {
        color: colors.textSecondary,
        fontSize: 16,
    },
    card: {
        backgroundColor: colors.card,
        padding: 30,
        borderRadius: 15,
        elevation: 4,
        shadowColor: '#000',
        shadowOffset: { width: 0, height: 2 },
        shadowOpacity: 0.1,
        shadowRadius: 4,
        marginBottom: 30,
        minHeight: 200,
        justifyContent: 'center',
    },
    label: {
        fontSize: 14,
        color: colors.textSecondary,
        marginBottom: 10,
        textTransform: 'uppercase',
        fontWeight: 'bold',
    },
    text: {
        fontSize: 22,
        color: colors.textPrimary,
        textAlign: 'center',
        lineHeight: 30,
    },
    answerContainer: {
        marginTop: 20,
    },
    divider: {
        height: 1,
        backgroundColor: colors.divider,
        marginVertical: 20,
    },
    button: {
        backgroundColor: colors.primary,
        paddingVertical: 15,
        borderRadius: 30,
        alignItems: 'center',
        elevation: 2,
    },
    completeButton: {
        backgroundColor: colors.success,
    },
    buttonText: {
        color: colors.textInverse,
        fontSize: 18,
        fontWeight: 'bold',
    },
});
