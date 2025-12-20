import React, { createContext, useContext, useState, useEffect, ReactNode, useCallback, useRef } from 'react';
import { Platform, BackHandler } from 'react-native';

type ScreenName = 'Login' | 'Home' | 'AddMaterial' | 'MaterialDetail' | 'Review' | 'Summary' | 'Settings' | 'DailyFeed';

interface NavigationContextType {
    currentScreen: ScreenName;
    params: any;
    navigate: (screen: ScreenName, params?: any) => void;
    goBack: () => void;
    canGoBack: boolean;
}

const NavigationContext = createContext<NavigationContextType | undefined>(undefined);

export const NavigationProvider = ({ children }: { children: ReactNode }) => {
    const [stack, setStack] = useState<{ screen: ScreenName; params: any }[]>([{ screen: 'Home', params: {} }]);
    const stackRef = useRef(stack);

    // Keep the Ref in sync with the latest stack state
    useEffect(() => {
        stackRef.current = stack;
    }, [stack]);

    const current = stack[stack.length - 1];
    const canGoBack = stack.length > 1;

    // Handle Android Hardware Back Button
    // We use a persistent listener with a Ref to avoid potential race conditions 
    // or missing events on specific hardware like iQOO/Funtouch OS.
    useEffect(() => {
        if (Platform.OS === 'android') {
            console.log('[ManualRouter] Registering persistent BackHandler listener.');

            const onBackPress = () => {
                const currentStack = stackRef.current;
                console.log('[ManualRouter] Hardware Back Event Caught. Stack state:', {
                    stackSize: currentStack.length,
                    currentScreen: currentStack[currentStack.length - 1].screen
                });

                if (currentStack.length > 1) {
                    setStack(prev => {
                        const newStack = prev.slice(0, -1);
                        console.log('[ManualRouter] Popping stack. New top:', newStack[newStack.length - 1].screen);
                        return newStack;
                    });
                    console.log('[ManualRouter] Returning true (intercepting back event)');
                    return true;
                }

                console.log('[ManualRouter] Stack is at root. Returning false (letting OS handle it)');
                return false;
            };

            const subscription = BackHandler.addEventListener('hardwareBackPress', onBackPress);
            return () => {
                console.log('[ManualRouter] Removing BackHandler listener.');
                subscription.remove();
            };
        }
    }, []); // Register only once

    // Handle Browser Back Button (Web only)
    useEffect(() => {
        if (Platform.OS === 'web') {
            const onPopState = (event: PopStateEvent) => {
                // When browser back is pressed, we pop our internal stack
                setStack(prev => (prev.length > 1 ? prev.slice(0, -1) : prev));
            };
            window.addEventListener('popstate', onPopState);
            return () => window.removeEventListener('popstate', onPopState);
        }
    }, []);

    const navigate = (screen: ScreenName, params: any = {}) => {
        if (screen === 'Home') {
            setStack([{ screen: 'Home', params: {} }]);
            if (Platform.OS === 'web') {
                // For web, we can't easily clear history, but we can push a new state that represents root
                window.history.pushState({ index: 0 }, '', '#Home');
            }
            return;
        }

        setStack(prev => {
            const newStack = [...prev, { screen, params }];
            if (Platform.OS === 'web') {
                window.history.pushState({ index: newStack.length - 1 }, '', `#${screen}`);
            }
            return newStack;
        });
    };

    const goBack = () => {
        setStack(prev => {
            if (prev.length > 1) {
                if (Platform.OS === 'web') {
                    window.history.back(); // This will trigger popstate, which updates stack
                    // But wait, calling history.back() triggers popstate, which calls setStack...
                    // We should avoid double update.
                    // Actually, if we use the in-app back button, we want to trigger browser back.
                    // So we should just call history.back() on web, and let the event listener handle the state update.
                    return prev; // Don't update state here, let popstate handle it
                }
                return prev.slice(0, -1);
            }
            return prev;
        });
    };

    return (
        <NavigationContext.Provider value={{ currentScreen: current.screen, params: current.params, navigate, goBack, canGoBack }}>
            {children}
        </NavigationContext.Provider>
    );
};

export const useNavigation = () => {
    const context = useContext(NavigationContext);
    if (!context) {
        throw new Error('useNavigation must be used within a NavigationProvider');
    }
    return context;
};

export const useRoute = () => {
    const context = useContext(NavigationContext);
    if (!context) {
        throw new Error('useRoute must be used within a NavigationProvider');
    }
    return { params: context.params };
};
