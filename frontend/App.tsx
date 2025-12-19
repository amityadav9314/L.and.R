import 'react-native-gesture-handler';
import { enableScreens } from 'react-native-screens';
enableScreens(false);
import React, { useEffect } from 'react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { AppNavigator } from './src/navigation/AppNavigator';
import { StatusBar } from 'expo-status-bar';
import { NotificationService } from './src/services/notificationService';
import { useAuthStore } from './src/store/authStore';
import { LoginScreen } from './src/screens/LoginScreen';
import { HomeScreen } from './src/screens/HomeScreen';
import { AddMaterialScreen } from './src/screens/AddMaterialScreen';
import { MaterialDetailScreen } from './src/screens/MaterialDetailScreen';
import { ReviewScreen } from './src/screens/ReviewScreen';
import { SummaryScreen } from './src/screens/SummaryScreen';
import { SettingsScreen } from './src/screens/SettingsScreen';
import { SafeAreaProvider } from 'react-native-safe-area-context';
import { GestureHandlerRootView } from 'react-native-gesture-handler';
import { NavigationProvider, useNavigation } from './src/navigation/ManualRouter';
import { ThemeProvider, useTheme } from './src/utils/theme';
import { BottomNavBar } from './src/components/BottomNavBar';

export const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      staleTime: 5 * 60 * 1000, // Data stays fresh for 5 minutes
      gcTime: 10 * 60 * 1000, // Cache for 10 minutes (formerly cacheTime)
      refetchOnWindowFocus: false, // Don't refetch when window regains focus
      refetchOnMount: false, // Don't refetch when component mounts if data exists
      refetchOnReconnect: false, // Don't refetch on reconnect
      retry: 1, // Only retry failed requests once
    },
  },
});

import { Gesture, GestureDetector } from 'react-native-gesture-handler';
import { View, StyleSheet } from 'react-native';

const ScreenRenderer = () => {
  const { currentScreen, goBack, canGoBack } = useNavigation();

  // Swipe Back Gesture - restrict to left edge and non-swiping screens
  const swipeBack = Gesture.Pan()
    .activeOffsetX(20) // Activate only if moved 20px right
    .failOffsetY([-20, 20]) // Fail if moved vertically (scrolling)
    .hitSlop({ left: 0, width: 40 }) // Only start from the very left edge
    .onEnd((e) => {
      const screensWithInternalSwipe = ['MaterialDetail', 'Review'];
      if (canGoBack && e.translationX > 50 && !screensWithInternalSwipe.includes(currentScreen)) {
        goBack();
      }
    });

  // Keep HomeScreen mounted to preserve scroll position and state
  return (
    <GestureDetector gesture={swipeBack}>
      <View style={{ flex: 1 }}>
        <View style={[StyleSheet.absoluteFill, { display: currentScreen === 'Home' ? 'flex' : 'none', zIndex: currentScreen === 'Home' ? 1 : 0 }]}>
          <HomeScreen />
        </View>

        {currentScreen !== 'Home' && (
          <View style={{ flex: 1, zIndex: 2 }}>
            {currentScreen === 'AddMaterial' && <AddMaterialScreen />}
            {currentScreen === 'MaterialDetail' && <MaterialDetailScreen />}
            {currentScreen === 'Review' && <ReviewScreen />}
            {currentScreen === 'Summary' && <SummaryScreen />}
            {currentScreen === 'Settings' && <SettingsScreen />}
          </View>
        )}
      </View>
    </GestureDetector>
  );
};

function AppContent() {
  const { user, restoreSession, isLoading } = useAuthStore();

  useEffect(() => {
    restoreSession();
  }, []);

  useEffect(() => {
    // Initialize notifications when user is logged in
    if (user) {
      NotificationService.initialize();
    }
  }, [user]);

  if (isLoading) {
    return null; // Or a splash screen
  }

  // ...

  return (
    <SafeAreaProvider>
      <GestureHandlerRootView style={{ flex: 1 }}>
        {user ? (
          <NavigationProvider>
            <View style={{ flex: 1 }}>
              <ScreenRenderer />
              <BottomNavBar />
            </View>
          </NavigationProvider>
        ) : (
          <LoginScreen />
        )}
      </GestureHandlerRootView>
      {/* <StatusBar style="auto" /> */}
    </SafeAreaProvider>
  );
}

export default function App() {
  return (
    <QueryClientProvider client={queryClient}>
      <ThemeProvider>
        <AppContent />
      </ThemeProvider>
    </QueryClientProvider>
  );
}
