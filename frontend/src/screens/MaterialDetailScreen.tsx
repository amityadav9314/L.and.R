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
} from 'react-native';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { useNavigation, useRoute } from '../navigation/ManualRouter';
import { learningClient } from '../services/api';
import { AppHeader } from '../components/AppHeader';

const { width: SCREEN_WIDTH } = Dimensions.get('window');
const CARD_WIDTH = Math.min(SCREEN_WIDTH - 48, 500); // Cap width for larger screens
const SWIPE_THRESHOLD = 50; // Minimum distance to trigger swipe

export const MaterialDetailScreen = () => {
    const navigation = useNavigation();
    const route = useRoute();
    const queryClient = useQueryClient();
    const { materialId, title } = route.params as { materialId: string; title: string };
    const displayTitle = title || 'Material Details';

    const [currentIndex, setCurrentIndex] = useState(0);
    const [isFlipped, setIsFlipped] = useState(false);
    const [isReviewing, setIsReviewing] = useState(false);

    // Animation values
    const position = useRef(new Animated.Value(0)).current;
    const flipAnimation = useRef(new Animated.Value(0)).current;

    // Touch tracking for swipe
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

    // Complete review mutation
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

    const currentCard = flashcards[currentIndex];

    // Flip animation interpolations
    const frontInterpolate = flipAnimation.interpolate({
        inputRange: [0, 180],
        outputRange: ['0deg', '180deg'],
    });

    const backInterpolate = flipAnimation.interpolate({
        inputRange: [0, 180],
        outputRange: ['180deg', '360deg'],
    });

    const frontAnimatedStyle = {
        transform: [{ rotateY: frontInterpolate }],
    };

    const backAnimatedStyle = {
        transform: [{ rotateY: backInterpolate }],
    };

    const flipCard = useCallback(() => {
        if (isFlipped) {
            Animated.spring(flipAnimation, {
                toValue: 0,
                friction: 8,
                tension: 10,
                useNativeDriver: true,
            }).start();
        } else {
            Animated.spring(flipAnimation, {
                toValue: 180,
                friction: 8,
                tension: 10,
                useNativeDriver: true,
            }).start();
        }
        setIsFlipped(!isFlipped);
    }, [isFlipped, flipAnimation]);

    const goToCard = useCallback((index: number) => {
        if (index < 0 || index >= flashcards.length) return;

        const direction = index > currentIndex ? -1 : 1;

        Animated.timing(position, {
            toValue: direction * SCREEN_WIDTH,
            duration: 200,
            useNativeDriver: true,
        }).start(() => {
            setCurrentIndex(index);
            setIsFlipped(false);
            flipAnimation.setValue(0);
            position.setValue(-direction * SCREEN_WIDTH);

            Animated.spring(position, {
                toValue: 0,
                friction: 8,
                useNativeDriver: true,
            }).start();
        });
    }, [currentIndex, flashcards.length, position, flipAnimation]);

    const goNext = useCallback(() => {
        if (currentIndex < flashcards.length - 1) {
            goToCard(currentIndex + 1);
        }
    }, [currentIndex, flashcards.length, goToCard]);

    const goPrev = useCallback(() => {
        if (currentIndex > 0) {
            goToCard(currentIndex - 1);
        }
    }, [currentIndex, goToCard]);

    // Touch event handlers for swipe (works on both web and native)
    const handleTouchStart = useCallback((e: GestureResponderEvent | React.TouchEvent | React.MouseEvent) => {
        let clientX: number, clientY: number;

        if ('nativeEvent' in e && 'pageX' in e.nativeEvent) {
            // React Native touch event
            clientX = e.nativeEvent.pageX;
            clientY = e.nativeEvent.pageY;
        } else if ('touches' in e && e.touches.length > 0) {
            // Web touch event
            clientX = e.touches[0].clientX;
            clientY = e.touches[0].clientY;
        } else if ('clientX' in e) {
            // Web mouse event
            clientX = e.clientX;
            clientY = e.clientY;
        } else {
            return;
        }

        touchStartX.current = clientX;
        touchStartY.current = clientY;
        touchCurrentX.current = clientX;
        isSwiping.current = false;
    }, []);

    const handleTouchMove = useCallback((e: GestureResponderEvent | React.TouchEvent | React.MouseEvent) => {
        let clientX: number, clientY: number;

        if ('nativeEvent' in e && 'pageX' in e.nativeEvent) {
            clientX = e.nativeEvent.pageX;
            clientY = e.nativeEvent.pageY;
        } else if ('touches' in e && e.touches.length > 0) {
            clientX = e.touches[0].clientX;
            clientY = e.touches[0].clientY;
        } else if ('clientX' in e) {
            clientX = e.clientX;
            clientY = e.clientY;
        } else {
            return;
        }

        const deltaX = clientX - touchStartX.current;
        const deltaY = clientY - touchStartY.current;

        // Only start swiping if horizontal movement is greater than vertical
        if (!isSwiping.current && Math.abs(deltaX) > 10 && Math.abs(deltaX) > Math.abs(deltaY)) {
            isSwiping.current = true;
        }

        if (isSwiping.current) {
            touchCurrentX.current = clientX;
            position.setValue(deltaX);

            // Prevent default to stop scrolling while swiping
            if ('preventDefault' in e) {
                e.preventDefault();
            }
        }
    }, [position]);

    const handleTouchEnd = useCallback(() => {
        const deltaX = touchCurrentX.current - touchStartX.current;

        if (isSwiping.current) {
            if (deltaX > SWIPE_THRESHOLD && currentIndex > 0) {
                // Swipe right - go to previous
                goPrev();
            } else if (deltaX < -SWIPE_THRESHOLD && currentIndex < flashcards.length - 1) {
                // Swipe left - go to next
                goNext();
            } else {
                // Reset position with spring animation
                Animated.spring(position, {
                    toValue: 0,
                    friction: 5,
                    useNativeDriver: true,
                }).start();
            }
        }

        isSwiping.current = false;
    }, [currentIndex, flashcards.length, goNext, goPrev, position]);

    // Handle card tap (only if not swiping)
    const handleCardPress = useCallback(() => {
        if (!isSwiping.current) {
            flipCard();
        }
    }, [flipCard]);

    const handleCompleteReview = async () => {
        if (!currentCard) return;

        setIsReviewing(true);
        try {
            // Complete review for current flashcard only
            await completeReviewMutation.mutateAsync(currentCard.id);

            // Check if there are more cards
            if (currentIndex < flashcards.length - 1) {
                // Move to next card
                goNext();
            } else if (flashcards.length === 1) {
                // This was the last card
                Alert.alert(
                    'All Done!',
                    'You\'ve completed all flashcards for this material.',
                    [{ text: 'OK', onPress: () => navigation.navigate('Home') }]
                );
            } else {
                // Was on last card but there were others - go back to first remaining
                Alert.alert(
                    'Card Reviewed!',
                    'Great job! Moving to the next card.',
                    [{ text: 'OK' }]
                );
            }
        } catch (err) {
            console.error('[MaterialDetail] Failed to complete review:', err);
            Alert.alert('Error', 'Failed to complete review. Please try again.');
        } finally {
            setIsReviewing(false);
        }
    };

    // Progress dots renderer
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
                            style={[
                                styles.progressDot,
                                index === currentIndex && styles.progressDotActive,
                            ]}
                        />
                    ))}
                </View>
            );
        }

        return (
            <View style={styles.progressTextContainer}>
                <Text style={styles.progressText}>
                    {currentIndex + 1} of {flashcards.length}
                </Text>
            </View>
        );
    };

    if (isLoading) {
        return (
            <View style={styles.container}>
                <AppHeader />
                <View style={styles.loadingContainer}>
                    <ActivityIndicator size="large" color="#4285F4" />
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
                    <TouchableOpacity
                        style={styles.backButton}
                        onPress={() => navigation.navigate('Home')}
                    >
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
                {/* Title */}
                <Text style={styles.headerTitle} numberOfLines={2}>{displayTitle}</Text>

                {/* Progress Indicator */}
                {renderProgressDots()}

                {/* Card Stack Container */}
                <View style={styles.cardStackContainer}>
                    {/* Background cards for stack effect */}
                    {currentIndex < flashcards.length - 1 && (
                        <View style={[styles.stackCard, styles.stackCard2]} />
                    )}
                    {currentIndex < flashcards.length - 2 && (
                        <View style={[styles.stackCard, styles.stackCard3]} />
                    )}

                    {/* Main Card */}
                    <Animated.View
                        style={[
                            styles.cardContainer,
                            { transform: [{ translateX: position }] },
                        ]}
                        onTouchStart={handleTouchStart as any}
                        onTouchMove={handleTouchMove as any}
                        onTouchEnd={handleTouchEnd}
                        // @ts-ignore - Web-specific props
                        onMouseDown={Platform.OS === 'web' ? handleTouchStart as any : undefined}
                        onMouseMove={Platform.OS === 'web' ? handleTouchMove as any : undefined}
                        onMouseUp={Platform.OS === 'web' ? handleTouchEnd : undefined}
                        onMouseLeave={Platform.OS === 'web' ? handleTouchEnd : undefined}
                    >
                        <TouchableOpacity
                            activeOpacity={0.95}
                            onPress={handleCardPress}
                            style={styles.cardTouchable}
                        >
                            {/* Front of card (Question) */}
                            <Animated.View
                                style={[styles.card, styles.cardFront, frontAnimatedStyle]}
                            >
                                <View style={styles.cardHeader}>
                                    <View style={styles.cardTypeIndicator}>
                                        <Text style={styles.cardTypeText}>Q</Text>
                                    </View>
                                    <View style={styles.stageBadge}>
                                        <Text style={styles.stageText}>Stage {currentCard?.stage || 0}</Text>
                                    </View>
                                </View>

                                <View style={styles.cardContent}>
                                    <Text style={styles.questionText}>{currentCard?.question}</Text>
                                </View>

                                <View style={styles.cardFooter}>
                                    <Text style={styles.tapHint}>👆 Tap to reveal answer</Text>
                                </View>
                            </Animated.View>

                            {/* Back of card (Answer) */}
                            <Animated.View
                                style={[styles.card, styles.cardBack, backAnimatedStyle]}
                            >
                                <View style={styles.cardHeader}>
                                    <View style={[styles.cardTypeIndicator, styles.cardTypeAnswer]}>
                                        <Text style={styles.cardTypeText}>A</Text>
                                    </View>
                                    <View style={styles.stageBadge}>
                                        <Text style={styles.stageText}>Stage {currentCard?.stage || 0}</Text>
                                    </View>
                                </View>

                                <View style={styles.cardContent}>
                                    <Text style={styles.answerText}>{currentCard?.answer}</Text>
                                </View>

                                <View style={styles.cardFooter}>
                                    <Text style={styles.tapHint}>👆 Tap for question</Text>
                                </View>
                            </Animated.View>
                        </TouchableOpacity>
                    </Animated.View>
                </View>

                {/* Swipe Hint */}
                <View style={styles.swipeHintContainer}>
                    <Text style={styles.swipeHint}>
                        {currentIndex > 0 && '← Swipe right for previous'}
                        {currentIndex > 0 && currentIndex < flashcards.length - 1 && '  •  '}
                        {currentIndex < flashcards.length - 1 && 'Swipe left for next →'}
                    </Text>
                </View>

                {/* Navigation Arrows */}
                <View style={styles.navigationContainer}>
                    <TouchableOpacity
                        style={[styles.navButton, currentIndex === 0 && styles.navButtonDisabled]}
                        onPress={goPrev}
                        disabled={currentIndex === 0}
                    >
                        <Text style={[styles.navButtonText, currentIndex === 0 && styles.navButtonTextDisabled]}>
                            ←
                        </Text>
                    </TouchableOpacity>

                    <TouchableOpacity
                        style={[styles.navButton, currentIndex === flashcards.length - 1 && styles.navButtonDisabled]}
                        onPress={goNext}
                        disabled={currentIndex === flashcards.length - 1}
                    >
                        <Text style={[styles.navButtonText, currentIndex === flashcards.length - 1 && styles.navButtonTextDisabled]}>
                            →
                        </Text>
                    </TouchableOpacity>
                </View>

                {/* Complete Review Button */}
                <TouchableOpacity
                    style={[styles.completeButton, isReviewing && styles.completeButtonDisabled]}
                    onPress={handleCompleteReview}
                    disabled={isReviewing}
                >
                    {isReviewing ? (
                        <ActivityIndicator size="small" color="#fff" />
                    ) : (
                        <Text style={styles.completeButtonText}>✓ Mark as Reviewed</Text>
                    )}
                </TouchableOpacity>
            </View>
        </View>
    );
};

const styles = StyleSheet.create({
    container: {
        flex: 1,
        backgroundColor: '#f0f4f8',
    },
    contentContainer: {
        flex: 1,
        padding: 20,
        paddingTop: 16,
    },
    headerTitle: {
        fontSize: 22,
        fontWeight: '700',
        color: '#1a1a2e',
        textAlign: 'center',
        marginBottom: 16,
    },

    // Progress indicators
    progressContainer: {
        flexDirection: 'row',
        justifyContent: 'center',
        alignItems: 'center',
        marginBottom: 20,
        gap: 8,
    },
    progressDot: {
        width: 10,
        height: 10,
        borderRadius: 5,
        backgroundColor: '#d0d5dd',
    },
    progressDotActive: {
        backgroundColor: '#4285F4',
        width: 12,
        height: 12,
        borderRadius: 6,
    },
    progressTextContainer: {
        alignItems: 'center',
        marginBottom: 20,
    },
    progressText: {
        fontSize: 16,
        fontWeight: '600',
        color: '#4285F4',
    },

    // Card stack
    cardStackContainer: {
        flex: 1,
        alignItems: 'center',
        justifyContent: 'center',
        maxHeight: 420,
    },
    stackCard: {
        position: 'absolute',
        width: CARD_WIDTH - 16,
        height: '92%',
        backgroundColor: '#fff',
        borderRadius: 20,
        shadowColor: '#000',
        shadowOffset: { width: 0, height: 2 },
        shadowOpacity: 0.1,
        shadowRadius: 4,
        elevation: 2,
    },
    stackCard2: {
        top: 8,
        transform: [{ scale: 0.96 }],
        opacity: 0.7,
    },
    stackCard3: {
        top: 16,
        transform: [{ scale: 0.92 }],
        opacity: 0.4,
    },

    // Card
    cardContainer: {
        width: CARD_WIDTH,
        height: '100%',
        maxHeight: 380,
    },
    cardTouchable: {
        flex: 1,
    },
    card: {
        position: 'absolute',
        width: '100%',
        height: '100%',
        backgroundColor: '#fff',
        borderRadius: 20,
        padding: 24,
        backfaceVisibility: 'hidden',
        shadowColor: '#4285F4',
        shadowOffset: { width: 0, height: 8 },
        shadowOpacity: 0.15,
        shadowRadius: 16,
        elevation: 8,
    },
    cardFront: {
        borderTopWidth: 4,
        borderTopColor: '#4285F4',
    },
    cardBack: {
        borderTopWidth: 4,
        borderTopColor: '#34A853',
    },
    cardHeader: {
        flexDirection: 'row',
        justifyContent: 'space-between',
        alignItems: 'center',
        marginBottom: 20,
    },
    cardTypeIndicator: {
        width: 36,
        height: 36,
        borderRadius: 18,
        backgroundColor: '#4285F4',
        justifyContent: 'center',
        alignItems: 'center',
    },
    cardTypeAnswer: {
        backgroundColor: '#34A853',
    },
    cardTypeText: {
        color: '#fff',
        fontSize: 18,
        fontWeight: '700',
    },
    stageBadge: {
        backgroundColor: '#f0f4f8',
        paddingHorizontal: 12,
        paddingVertical: 6,
        borderRadius: 12,
    },
    stageText: {
        fontSize: 12,
        fontWeight: '600',
        color: '#667085',
    },
    cardContent: {
        flex: 1,
        justifyContent: 'center',
    },
    questionText: {
        fontSize: 20,
        fontWeight: '600',
        color: '#1a1a2e',
        lineHeight: 30,
        textAlign: 'center',
    },
    answerText: {
        fontSize: 18,
        color: '#344054',
        lineHeight: 28,
        textAlign: 'center',
    },
    cardFooter: {
        alignItems: 'center',
        paddingTop: 16,
    },
    tapHint: {
        fontSize: 14,
        color: '#98a2b3',
        fontStyle: 'italic',
    },

    // Swipe hint
    swipeHintContainer: {
        alignItems: 'center',
        marginTop: 16,
        marginBottom: 8,
    },
    swipeHint: {
        fontSize: 13,
        color: '#98a2b3',
    },

    // Navigation
    navigationContainer: {
        flexDirection: 'row',
        justifyContent: 'center',
        gap: 20,
        marginBottom: 20,
    },
    navButton: {
        width: 50,
        height: 50,
        borderRadius: 25,
        backgroundColor: '#fff',
        justifyContent: 'center',
        alignItems: 'center',
        shadowColor: '#000',
        shadowOffset: { width: 0, height: 2 },
        shadowOpacity: 0.1,
        shadowRadius: 4,
        elevation: 3,
    },
    navButtonDisabled: {
        backgroundColor: '#f0f4f8',
        shadowOpacity: 0,
        elevation: 0,
    },
    navButtonText: {
        fontSize: 24,
        color: '#4285F4',
        fontWeight: '600',
    },
    navButtonTextDisabled: {
        color: '#d0d5dd',
    },

    // Complete button
    completeButton: {
        backgroundColor: '#34A853',
        paddingVertical: 16,
        borderRadius: 14,
        alignItems: 'center',
        shadowColor: '#34A853',
        shadowOffset: { width: 0, height: 4 },
        shadowOpacity: 0.3,
        shadowRadius: 8,
        elevation: 4,
    },
    completeButtonDisabled: {
        backgroundColor: '#98a2b3',
        shadowOpacity: 0,
    },
    completeButtonText: {
        color: '#fff',
        fontSize: 18,
        fontWeight: '700',
    },

    // Loading state
    loadingContainer: {
        flex: 1,
        justifyContent: 'center',
        alignItems: 'center',
    },
    loadingText: {
        marginTop: 16,
        fontSize: 16,
        color: '#667085',
    },

    // Error state
    errorContainer: {
        flex: 1,
        justifyContent: 'center',
        alignItems: 'center',
        padding: 20,
    },
    errorText: {
        fontSize: 18,
        color: '#d32f2f',
        marginBottom: 16,
    },
    retryButton: {
        backgroundColor: '#4285F4',
        paddingHorizontal: 24,
        paddingVertical: 12,
        borderRadius: 8,
    },
    retryButtonText: {
        color: '#fff',
        fontSize: 16,
        fontWeight: '600',
    },

    // Empty state
    emptyContainer: {
        flex: 1,
        justifyContent: 'center',
        alignItems: 'center',
        padding: 20,
    },
    emptyIcon: {
        fontSize: 64,
        marginBottom: 16,
    },
    emptyTitle: {
        fontSize: 24,
        fontWeight: '700',
        color: '#1a1a2e',
        marginBottom: 8,
    },
    emptyText: {
        fontSize: 16,
        color: '#667085',
        textAlign: 'center',
        marginBottom: 24,
    },
    backButton: {
        backgroundColor: '#4285F4',
        paddingHorizontal: 24,
        paddingVertical: 14,
        borderRadius: 12,
    },
    backButtonText: {
        color: '#fff',
        fontSize: 16,
        fontWeight: '600',
    },
});
