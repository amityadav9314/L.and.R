import React, { createContext, useContext, useState, useEffect, ReactNode, useCallback, useRef } from 'react';
import { Platform, BackHandler } from 'react-native';

type ScreenName = 'Login' | 'Home' | 'AddMaterial' | 'MaterialDetail' | 'Review' | 'Summary' | 'Settings';

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

    // Keep ref in sync with state
    useEffect(() => {
        stackRef.current = stack;
    }, [stack]);

    const current = stack[stack.length - 1];
    const canGoBack = stack.length > 1;

    // Handle Android Hardware Back Button
    useEffect(() => {
        if (Platform.OS === 'android') {
            const onBackPress = () => {
                // Use ref to get current stack value (avoids stale closure)
                if (stackRef.current.length > 1) {
                    // If we can go back in our navigation stack, do so
                    setStack(prev => prev.slice(0, -1));
                    return true; // Prevent default behavior (exit app)
                }
                return false; // Let default behavior happen (exit app)
            };

            const subscription = BackHandler.addEventListener('hardwareBackPress', onBackPress);
            return () => subscription.remove();
        }
    }, []); // Empty deps - only set up once

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
