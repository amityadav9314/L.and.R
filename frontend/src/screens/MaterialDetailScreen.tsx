import React, { useState, useRef, useCallback } from 'react';
import {
    View,
    Text,
    StyleSheet,
    TouchableOpacity,
    Animated,
    Dimensions,
    ActivityIndicator,
    Alert,
    Platform,
    GestureResponderEvent,
    Modal,
    TextInput,
    KeyboardAvoidingView,
} from 'react-native';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { useNavigation, useRoute } from '../navigation/ManualRouter';
import { learningClient } from '../services/api';
import { AppHeader } from '../components/AppHeader';
import { useTheme, ThemeColors } from '../utils/theme';

const { width: SCREEN_WIDTH } = Dimensions.get('window');
const CARD_WIDTH = Math.min(SCREEN_WIDTH - 48, 500);
const SWIPE_THRESHOLD = 50;

export const MaterialDetailScreen = () => {
    const navigation = useNavigation();
    const route = useRoute();
    const queryClient = useQueryClient();
    const { colors } = useTheme();
    const { materialId, title } = route.params as { materialId: string; title: string };
    const displayTitle = title || 'Material Details';

    const [currentIndex, setCurrentIndex] = useState(0);
    const [isFlipped, setIsFlipped] = useState(false);
    const [isReviewing, setIsReviewing] = useState(false);
    const [isEditing, setIsEditing] = useState(false);
    const [editQuestion, setEditQuestion] = useState('');
    const [editAnswer, setEditAnswer] = useState('');

    const position = useRef(new Animated.Value(0)).current;
    const flipAnimation = useRef(new Animated.Value(0)).current;

    const touchStartX = useRef(0);
    const touchStartY = useRef(0);
    const touchCurrentX = useRef(0);
    const isSwiping = useRef(false);

    const { data: flashcards = [], isLoading, error, refetch } = useQuery({
        queryKey: ['dueFlashcards', materialId],
        queryFn: async () => {
            console.log(`[MaterialDetail] Fetching flashcards for material: ${materialId}`);
            try {
                const response = await learningClient.getDueFlashcards({ materialId });
                console.log('[MaterialDetail] Got flashcards:', response.flashcards?.length || 0);
                return response.flashcards || [];
            } catch (err) {
                console.error('[MaterialDetail] Failed to fetch flashcards:', err);
                return [];
            }
        },
    });

    const completeReviewMutation = useMutation({
        mutationFn: async (flashcardId: string) => {
            return await learningClient.completeReview({ flashcardId });
        },
        onSuccess: () => {
            queryClient.invalidateQueries({ queryKey: ['dueFlashcards', materialId] });
            queryClient.invalidateQueries({ queryKey: ['dueMaterials'] });
            queryClient.invalidateQueries({ queryKey: ['notificationStatus'] });
        },
    });

    const failReviewMutation = useMutation({
        mutationFn: async (flashcardId: string) => {
            return await learningClient.failReview({ flashcardId });
        },
        onSuccess: () => {
            queryClient.invalidateQueries({ queryKey: ['dueFlashcards', materialId] });
            queryClient.invalidateQueries({ queryKey: ['dueMaterials'] });
        },
    });

    const updateFlashcardMutation = useMutation({
        mutationFn: async (data: { flashcardId: string; question: string; answer: string }) => {
            return await learningClient.updateFlashcard(data);
        },
        onSuccess: () => {
            queryClient.invalidateQueries({ queryKey: ['dueFlashcards', materialId] });
            setIsEditing(false);
            Alert.alert('Success', 'Flashcard updated successfully');
        },
        onError: (err) => {
            console.error('[MaterialDetail] Failed to update flashcard:', err);
            Alert.alert('Error', 'Failed to update flashcard');
        },
    });

    const currentCard = flashcards[currentIndex];
    const styles = createStyles(colors);

    const frontInterpolate = flipAnimation.interpolate({
        inputRange: [0, 180],
        outputRange: ['0deg', '180deg'],
    });

    const backInterpolate = flipAnimation.interpolate({
        inputRange: [0, 180],
        outputRange: ['180deg', '360deg'],
    });

    const frontAnimatedStyle = { transform: [{ rotateY: frontInterpolate }] };
    const backAnimatedStyle = { transform: [{ rotateY: backInterpolate }] };

    const flipCard = useCallback(() => {
        if (isFlipped) {
            Animated.spring(flipAnimation, { toValue: 0, friction: 8, tension: 10, useNativeDriver: true }).start();
        } else {
            Animated.spring(flipAnimation, { toValue: 180, friction: 8, tension: 10, useNativeDriver: true }).start();
        }
        setIsFlipped(!isFlipped);
    }, [isFlipped, flipAnimation]);

    const goToCard = useCallback((index: number) => {
        if (index < 0 || index >= flashcards.length) return;
        const direction = index > currentIndex ? -1 : 1;
        Animated.timing(position, { toValue: direction * SCREEN_WIDTH, duration: 200, useNativeDriver: true }).start(() => {
            setCurrentIndex(index);
            setIsFlipped(false);
            flipAnimation.setValue(0);
            position.setValue(-direction * SCREEN_WIDTH);
            Animated.spring(position, { toValue: 0, friction: 8, useNativeDriver: true }).start();
        });
    }, [currentIndex, flashcards.length, position, flipAnimation]);

    const goNext = useCallback(() => {
        if (currentIndex < flashcards.length - 1) goToCard(currentIndex + 1);
    }, [currentIndex, flashcards.length, goToCard]);

    const goPrev = useCallback(() => {
        if (currentIndex > 0) goToCard(currentIndex - 1);
    }, [currentIndex, goToCard]);

    const handleTouchStart = useCallback((e: GestureResponderEvent | React.TouchEvent | React.MouseEvent) => {
        let clientX: number, clientY: number;
        if ('nativeEvent' in e && 'pageX' in e.nativeEvent) {
            clientX = e.nativeEvent.pageX; clientY = e.nativeEvent.pageY;
        } else if ('touches' in e && e.touches.length > 0) {
            clientX = e.touches[0].clientX; clientY = e.touches[0].clientY;
        } else if ('clientX' in e) {
            clientX = e.clientX; clientY = e.clientY;
        } else { return; }
        touchStartX.current = clientX; touchStartY.current = clientY; touchCurrentX.current = clientX; isSwiping.current = false;
    }, []);

    const handleTouchMove = useCallback((e: GestureResponderEvent | React.TouchEvent | React.MouseEvent) => {
        let clientX: number, clientY: number;
        if ('nativeEvent' in e && 'pageX' in e.nativeEvent) {
            clientX = e.nativeEvent.pageX; clientY = e.nativeEvent.pageY;
        } else if ('touches' in e && e.touches.length > 0) {
            clientX = e.touches[0].clientX; clientY = e.touches[0].clientY;
        } else if ('clientX' in e) {
            clientX = e.clientX; clientY = e.clientY;
        } else { return; }
        const deltaX = clientX - touchStartX.current;
        const deltaY = clientY - touchStartY.current;
        if (!isSwiping.current && Math.abs(deltaX) > 10 && Math.abs(deltaX) > Math.abs(deltaY)) {
            isSwiping.current = true;
        }
        if (isSwiping.current) {
            touchCurrentX.current = clientX;
            position.setValue(deltaX);
            if ('preventDefault' in e) e.preventDefault();
        }
    }, [position]);

    const handleTouchEnd = useCallback(() => {
        const deltaX = touchCurrentX.current - touchStartX.current;
        if (isSwiping.current) {
            if (deltaX > SWIPE_THRESHOLD && currentIndex > 0) goPrev();
            else if (deltaX < -SWIPE_THRESHOLD && currentIndex < flashcards.length - 1) goNext();
            else Animated.spring(position, { toValue: 0, friction: 5, useNativeDriver: true }).start();
        }
        isSwiping.current = false;
    }, [currentIndex, flashcards.length, goNext, goPrev, position]);

    const handleCardPress = useCallback(() => {
        if (!isSwiping.current) flipCard();
    }, [flipCard]);

    const handleCompleteReview = async () => {
        if (!currentCard) return;
        setIsReviewing(true);
        try {
            await completeReviewMutation.mutateAsync(currentCard.id);
            if (currentIndex < flashcards.length - 1) {
                goNext();
            } else if (flashcards.length === 1) {
                Alert.alert('All Done!', 'You\'ve completed all flashcards for this material.', [{ text: 'OK', onPress: () => navigation.navigate('Home') }]);
            } else {
                Alert.alert('Card Reviewed!', 'Great job! Moving to the next card.', [{ text: 'OK' }]);
            }
        } catch (err) {
            console.error('[MaterialDetail] Failed to complete review:', err);
            Alert.alert('Error', 'Failed to complete review. Please try again.');
        } finally {
            setIsReviewing(false);
        }
    };

    const handleFailReview = async () => {
        if (!currentCard) return;
        setIsReviewing(true);
        try {
            await failReviewMutation.mutateAsync(currentCard.id);
            if (currentIndex < flashcards.length - 1) {
                goNext();
            } else {
                Alert.alert('Keep Learning!', 'Card will be reviewed again tomorrow.', [{ text: 'OK' }]);
            }
        } catch (err) {
            console.error('[MaterialDetail] Failed to fail review:', err);
            Alert.alert('Error', 'Failed to update card. Please try again.');
        } finally {
            setIsReviewing(false);
        }
    };

    const handleEdit = () => {
        if (!currentCard) return;
        setEditQuestion(currentCard.question);
        setEditAnswer(currentCard.answer);
        setIsEditing(true);
    };

    const handleSaveEdit = () => {
        if (!currentCard) return;
        updateFlashcardMutation.mutate({
            flashcardId: currentCard.id,
            question: editQuestion,
            answer: editAnswer,
        });
    };

    const renderProgressDots = () => {
        if (flashcards.length === 0) return null;
        const maxDots = 7;
        const showDots = flashcards.length <= maxDots;

        if (showDots) {
            return (
                <View style={styles.progressContainer}>
                    {flashcards.map((_: unknown, index: number) => (
                        <TouchableOpacity
                            key={index}
                            onPress={() => goToCard(index)}
                            style={[styles.progressDot, index === currentIndex && styles.progressDotActive]}
                        />
                    ))}
                </View>
            );
        }
        return (
            <View style={styles.progressTextContainer}>
                <Text style={styles.progressText}>{currentIndex + 1} of {flashcards.length}</Text>
            </View>
        );
    };

    if (isLoading) {
        return (
            <View style={styles.container}>
                <AppHeader />
                <View style={styles.loadingContainer}>
                    <ActivityIndicator size="large" color={colors.primary} />
                    <Text style={styles.loadingText}>Loading flashcards...</Text>
                </View>
            </View>
        );
    }

    if (error) {
        return (
            <View style={styles.container}>
                <AppHeader />
                <View style={styles.errorContainer}>
                    <Text style={styles.errorText}>Failed to load flashcards</Text>
                    <TouchableOpacity style={styles.retryButton} onPress={() => refetch()}>
                        <Text style={styles.retryButtonText}>Retry</Text>
                    </TouchableOpacity>
                </View>
            </View>
        );
    }

    if (flashcards.length === 0) {
        return (
            <View style={styles.container}>
                <AppHeader />
                <View style={styles.emptyContainer}>
                    <Text style={styles.emptyIcon}>🎉</Text>
                    <Text style={styles.emptyTitle}>All caught up!</Text>
                    <Text style={styles.emptyText}>No flashcards due for this material.</Text>
                    <TouchableOpacity style={styles.backButton} onPress={() => navigation.navigate('Home')}>
                        <Text style={styles.backButtonText}>Back to Home</Text>
                    </TouchableOpacity>
                </View>
            </View>
        );
    }

    return (
        <View style={styles.container}>
            <AppHeader />
            <View style={styles.contentContainer}>
                <View style={styles.headerRow}>
                    <Text style={styles.headerTitle} numberOfLines={1}>{displayTitle}</Text>
                    <TouchableOpacity onPress={handleEdit} style={styles.editButton}>
                        <Text style={styles.editButtonText}>✎</Text>
                    </TouchableOpacity>
                </View>
                {renderProgressDots()}

                <Modal
                    visible={isEditing}
                    animationType="slide"
                    transparent={true}
                    onRequestClose={() => setIsEditing(false)}
                >
                    <KeyboardAvoidingView
                        behavior={Platform.OS === "ios" ? "padding" : "height"}
                        style={styles.modalContainer}
                    >
                        <View style={styles.modalContent}>
                            <Text style={styles.modalTitle}>Edit Flashcard</Text>

                            <Text style={styles.inputLabel}>Question</Text>
                            <TextInput
                                style={styles.input}
                                value={editQuestion}
                                onChangeText={setEditQuestion}
                                multiline
                            />

                            <Text style={styles.inputLabel}>Answer</Text>
                            <TextInput
                                style={styles.input}
                                value={editAnswer}
                                onChangeText={setEditAnswer}
                                multiline
                            />

                            <View style={styles.modalButtons}>
                                <TouchableOpacity
                                    style={[styles.modalButton, styles.cancelButton]}
                                    onPress={() => setIsEditing(false)}
                                >
                                    <Text style={styles.modalButtonText}>Cancel</Text>
                                </TouchableOpacity>
                                <TouchableOpacity
                                    style={[styles.modalButton, styles.saveButton]}
                                    onPress={handleSaveEdit}
                                >
                                    <Text style={styles.modalButtonText}>Save</Text>
                                </TouchableOpacity>
                            </View>
                        </View>
                    </KeyboardAvoidingView>
                </Modal>

                <View style={styles.cardStackContainer}>
                    {currentIndex < flashcards.length - 1 && <View style={[styles.stackCard, styles.stackCard2]} />}
                    {currentIndex < flashcards.length - 2 && <View style={[styles.stackCard, styles.stackCard3]} />}

                    <Animated.View
                        style={[styles.cardContainer, { transform: [{ translateX: position }] }]}
                        onTouchStart={handleTouchStart as any}
                        onTouchMove={handleTouchMove as any}
                        onTouchEnd={handleTouchEnd}
                        // @ts-ignore
                        onMouseDown={Platform.OS === 'web' ? handleTouchStart as any : undefined}
                        onMouseMove={Platform.OS === 'web' ? handleTouchMove as any : undefined}
                        onMouseUp={Platform.OS === 'web' ? handleTouchEnd : undefined}
                        onMouseLeave={Platform.OS === 'web' ? handleTouchEnd : undefined}
                    >
                        <TouchableOpacity activeOpacity={0.95} onPress={handleCardPress} style={styles.cardTouchable}>
                            <Animated.View style={[styles.card, styles.cardFront, frontAnimatedStyle]}>
                                <View style={styles.cardHeader}>
                                    <View style={styles.cardTypeIndicator}><Text style={styles.cardTypeText}>Q</Text></View>
                                    <View style={styles.stageBadge}><Text style={styles.stageText}>Stage {currentCard?.stage || 0}</Text></View>
                                </View>
                                <View style={styles.cardContent}><Text style={styles.questionText}>{currentCard?.question}</Text></View>
                                <View style={styles.cardFooter}><Text style={styles.tapHint}>👆 Tap to reveal answer</Text></View>
                            </Animated.View>

                            <Animated.View style={[styles.card, styles.cardBack, backAnimatedStyle]}>
                                <View style={styles.cardHeader}>
                                    <View style={[styles.cardTypeIndicator, styles.cardTypeAnswer]}><Text style={styles.cardTypeText}>A</Text></View>
                                    <View style={styles.stageBadge}><Text style={styles.stageText}>Stage {currentCard?.stage || 0}</Text></View>
                                </View>
                                <View style={styles.cardContent}><Text style={styles.answerText}>{currentCard?.answer}</Text></View>
                                <View style={styles.cardFooter}><Text style={styles.tapHint}>👆 Tap for question</Text></View>
                            </Animated.View>
                        </TouchableOpacity>
                    </Animated.View>
                </View>

                <View style={styles.swipeHintContainer}>
                    <Text style={styles.swipeHint}>
                        {currentIndex > 0 && '← Swipe right for previous'}
                        {currentIndex > 0 && currentIndex < flashcards.length - 1 && '  •  '}
                        {currentIndex < flashcards.length - 1 && 'Swipe left for next →'}
                    </Text>
                </View>

                <View style={styles.navigationContainer}>
                    <TouchableOpacity style={[styles.navButton, currentIndex === 0 && styles.navButtonDisabled]} onPress={goPrev} disabled={currentIndex === 0}>
                        <Text style={[styles.navButtonText, currentIndex === 0 && styles.navButtonTextDisabled]}>←</Text>
                    </TouchableOpacity>
                    <TouchableOpacity style={[styles.navButton, currentIndex === flashcards.length - 1 && styles.navButtonDisabled]} onPress={goNext} disabled={currentIndex === flashcards.length - 1}>
                        <Text style={[styles.navButtonText, currentIndex === flashcards.length - 1 && styles.navButtonTextDisabled]}>→</Text>
                    </TouchableOpacity>
                </View>

                <View style={styles.actionButtonsContainer}>
                    <TouchableOpacity style={[styles.wrongButton, isReviewing && styles.buttonDisabled]} onPress={handleFailReview} disabled={isReviewing}>
                        <Text style={styles.wrongButtonText}>✗ Wrong</Text>
                    </TouchableOpacity>
                    <TouchableOpacity style={[styles.correctButton, isReviewing && styles.buttonDisabled]} onPress={handleCompleteReview} disabled={isReviewing}>
                        {isReviewing ? <ActivityIndicator size="small" color={colors.textInverse} /> : <Text style={styles.correctButtonText}>✓ Got It</Text>}
                    </TouchableOpacity>
                </View>
            </View>
        </View>
    );
};

const createStyles = (colors: ThemeColors) => StyleSheet.create({
    container: { flex: 1, backgroundColor: colors.background },
    contentContainer: { flex: 1, padding: 20, paddingTop: 16 },
    headerTitle: { fontSize: 22, fontWeight: '700', color: colors.textPrimary, textAlign: 'center', marginBottom: 16 },
    progressContainer: { flexDirection: 'row', justifyContent: 'center', alignItems: 'center', marginBottom: 20, gap: 8 },
    progressDot: { width: 10, height: 10, borderRadius: 5, backgroundColor: colors.border },
    progressDotActive: { backgroundColor: colors.primary, width: 12, height: 12, borderRadius: 6 },
    progressTextContainer: { alignItems: 'center', marginBottom: 20 },
    progressText: { fontSize: 16, fontWeight: '600', color: colors.primary },
    cardStackContainer: { flex: 1, alignItems: 'center', justifyContent: 'center', maxHeight: 420 },
    stackCard: { position: 'absolute', width: CARD_WIDTH - 16, height: '92%', backgroundColor: colors.card, borderRadius: 20, shadowColor: '#000', shadowOffset: { width: 0, height: 2 }, shadowOpacity: 0.1, shadowRadius: 4, elevation: 2 },
    stackCard2: { top: 8, transform: [{ scale: 0.96 }], opacity: 0.7 },
    stackCard3: { top: 16, transform: [{ scale: 0.92 }], opacity: 0.4 },
    cardContainer: { width: CARD_WIDTH, height: '100%', maxHeight: 380 },
    cardTouchable: { flex: 1 },
    card: { position: 'absolute', width: '100%', height: '100%', backgroundColor: colors.card, borderRadius: 20, padding: 24, backfaceVisibility: 'hidden', shadowColor: colors.primary, shadowOffset: { width: 0, height: 8 }, shadowOpacity: 0.15, shadowRadius: 16, elevation: 8 },
    cardFront: { borderTopWidth: 4, borderTopColor: colors.primary },
    cardBack: { borderTopWidth: 4, borderTopColor: colors.success },
    cardHeader: { flexDirection: 'row', justifyContent: 'space-between', alignItems: 'center', marginBottom: 20 },
    cardTypeIndicator: { width: 36, height: 36, borderRadius: 18, backgroundColor: colors.primary, justifyContent: 'center', alignItems: 'center' },
    cardTypeAnswer: { backgroundColor: colors.success },
    cardTypeText: { color: colors.textInverse, fontSize: 18, fontWeight: '700' },
    stageBadge: { backgroundColor: colors.cardAlt, paddingHorizontal: 12, paddingVertical: 6, borderRadius: 12 },
    stageText: { fontSize: 12, fontWeight: '600', color: colors.textSecondary },
    cardContent: { flex: 1, justifyContent: 'center' },
    questionText: { fontSize: 20, fontWeight: '600', color: colors.textPrimary, lineHeight: 30, textAlign: 'center' },
    answerText: { fontSize: 18, color: colors.textPrimary, lineHeight: 28, textAlign: 'center' },
    cardFooter: { alignItems: 'center', paddingTop: 16 },
    tapHint: { fontSize: 14, color: colors.textSecondary, fontStyle: 'italic' },
    swipeHintContainer: { alignItems: 'center', marginTop: 16, marginBottom: 8 },
    swipeHint: { fontSize: 13, color: colors.textSecondary },
    navigationContainer: { flexDirection: 'row', justifyContent: 'center', gap: 20, marginBottom: 20 },
    navButton: { width: 50, height: 50, borderRadius: 25, backgroundColor: colors.card, justifyContent: 'center', alignItems: 'center', shadowColor: '#000', shadowOffset: { width: 0, height: 2 }, shadowOpacity: 0.1, shadowRadius: 4, elevation: 3 },
    navButtonDisabled: { backgroundColor: colors.cardAlt, shadowOpacity: 0, elevation: 0 },
    navButtonText: { fontSize: 24, color: colors.primary, fontWeight: '600' },
    navButtonTextDisabled: { color: colors.border },
    completeButton: { backgroundColor: colors.success, paddingVertical: 16, borderRadius: 14, alignItems: 'center', shadowColor: colors.success, shadowOffset: { width: 0, height: 4 }, shadowOpacity: 0.3, shadowRadius: 8, elevation: 4 },
    completeButtonDisabled: { backgroundColor: colors.paginationDisabledBg, shadowOpacity: 0 },
    completeButtonText: { color: colors.textInverse, fontSize: 18, fontWeight: '700' },
    loadingContainer: { flex: 1, justifyContent: 'center', alignItems: 'center' },
    loadingText: { marginTop: 16, fontSize: 16, color: colors.textSecondary },
    errorContainer: { flex: 1, justifyContent: 'center', alignItems: 'center', padding: 20 },
    errorText: { fontSize: 18, color: colors.error, marginBottom: 16 },
    retryButton: { backgroundColor: colors.primary, paddingHorizontal: 24, paddingVertical: 12, borderRadius: 8 },
    retryButtonText: { color: colors.textInverse, fontSize: 16, fontWeight: '600' },
    emptyContainer: { flex: 1, justifyContent: 'center', alignItems: 'center', padding: 20 },
    emptyIcon: { fontSize: 64, marginBottom: 16 },
    emptyTitle: { fontSize: 24, fontWeight: '700', color: colors.textPrimary, marginBottom: 8 },
    emptyText: { fontSize: 16, color: colors.textSecondary, textAlign: 'center', marginBottom: 24 },
    backButton: { backgroundColor: colors.primary, paddingHorizontal: 24, paddingVertical: 14, borderRadius: 12 },
    backButtonText: { color: colors.textInverse, fontSize: 16, fontWeight: '600' },
    actionButtonsContainer: { flexDirection: 'row', gap: 12 },
    wrongButton: { flex: 1, backgroundColor: colors.error, paddingVertical: 16, borderRadius: 14, alignItems: 'center', shadowColor: colors.error, shadowOffset: { width: 0, height: 4 }, shadowOpacity: 0.3, shadowRadius: 8, elevation: 4 },
    wrongButtonText: { color: colors.textInverse, fontSize: 18, fontWeight: '700' },
    correctButton: { flex: 2, backgroundColor: colors.success, paddingVertical: 16, borderRadius: 14, alignItems: 'center', shadowColor: colors.success, shadowOffset: { width: 0, height: 4 }, shadowOpacity: 0.3, shadowRadius: 8, elevation: 4 },
    correctButtonText: { color: colors.textInverse, fontSize: 18, fontWeight: '700' },
    buttonDisabled: { opacity: 0.6, shadowOpacity: 0 },
    headerRow: { flexDirection: 'row', alignItems: 'center', justifyContent: 'center', marginBottom: 20 },
    editButton: { position: 'absolute', right: 0, padding: 8 },
    editButtonText: { fontSize: 24, color: colors.primary },
    modalContainer: { flex: 1, justifyContent: 'center', alignItems: 'center', backgroundColor: 'rgba(0,0,0,0.5)' },
    modalContent: { width: '90%', backgroundColor: colors.card, borderRadius: 20, padding: 20, shadowColor: '#000', shadowOffset: { width: 0, height: 2 }, shadowOpacity: 0.25, shadowRadius: 4, elevation: 5 },
    modalTitle: { fontSize: 20, fontWeight: 'bold', marginBottom: 20, color: colors.textPrimary, textAlign: 'center' },
    inputLabel: { fontSize: 16, fontWeight: '600', marginBottom: 8, color: colors.textSecondary },
    input: { backgroundColor: colors.background, borderRadius: 12, padding: 12, marginBottom: 16, color: colors.textPrimary, minHeight: 80, textAlignVertical: 'top' },
    modalButtons: { flexDirection: 'row', justifyContent: 'space-between', gap: 12 },
    modalButton: { flex: 1, padding: 14, borderRadius: 12, alignItems: 'center' },
    cancelButton: { backgroundColor: colors.error },
    saveButton: { backgroundColor: colors.primary },
    modalButtonText: { color: colors.textInverse, fontWeight: 'bold', fontSize: 16 },
});
