import React, { useEffect } from 'react';
import { NavigationContainer } from '@react-navigation/native';
import * as Linking from 'expo-linking';
import { createNativeStackNavigator } from '@react-navigation/native-stack';
import { ActivityIndicator, View } from 'react-native';
import { LoginScreen } from '../screens/LoginScreen';
import { HomeScreen } from '../screens/HomeScreen';
import { AddMaterialScreen } from '../screens/AddMaterialScreen';
import { ReviewScreen } from '../screens/ReviewScreen';
import { MaterialDetailScreen } from '../screens/MaterialDetailScreen';
import { useAuthStore } from '../store/authStore';

const Stack = createNativeStackNavigator();

export const AppNavigator = () => {
    const { token, isLoading, restoreSession } = useAuthStore();

    useEffect(() => {
        restoreSession();
    }, []);

    if (isLoading) {
        return (
            <View style={{ flex: 1, justifyContent: 'center', alignItems: 'center' }}>
                <ActivityIndicator size="large" />
            </View>
        );
    }

    const linking = {
        prefixes: [Linking.createURL('/')],
        config: {
            screens: {
                Login: 'login',
                Home: '',
                AddMaterial: 'add-material',
                MaterialDetail: 'material/:materialId',
                Review: 'review/:flashcardId',
            },
        },
    };

    return (
        <NavigationContainer linking={linking}>
            <Stack.Navigator screenOptions={{ headerShown: false }}>
                {token ? (
                    <>
                        <Stack.Screen name="Home" component={HomeScreen} />
                        <Stack.Screen name="AddMaterial" component={AddMaterialScreen} options={{ headerShown: true, title: 'Add Material' }} />
                        <Stack.Screen name="MaterialDetail" component={MaterialDetailScreen} options={{ headerShown: true, title: 'Flashcards' }} />
                        <Stack.Screen name="Review" component={ReviewScreen} options={{ headerShown: true, title: 'Review Flashcard' }} />
                    </>
                ) : (
                    <Stack.Screen name="Login" component={LoginScreen} />
                )}
            </Stack.Navigator>
        </NavigationContainer>
    );
};
