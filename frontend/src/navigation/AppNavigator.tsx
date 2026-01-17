import React, { useEffect } from 'react';
import { NavigationContainer } from '@react-navigation/native';
import * as Linking from 'expo-linking';
import { createStackNavigator } from '@react-navigation/stack';
import { ActivityIndicator, View } from 'react-native';
import { LoginScreen } from '../screens/LoginScreen';
import { HomeScreen } from '../screens/HomeScreen';
import { AddMaterialScreen } from '../screens/AddMaterialScreen';
import { ReviewScreen } from '../screens/ReviewScreen';
import { MaterialDetailScreen } from '../screens/MaterialDetailScreen';
import { UpgradeScreen } from '../screens/UpgradeScreen';
import { useAuthStore } from '../store/authStore';

const Stack = createStackNavigator();

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

    return (
        <NavigationContainer>
            <Stack.Navigator>
                {token ? (
                    <>
                        <Stack.Screen name="Home" component={HomeScreen} />
                        <Stack.Screen name="AddMaterial" component={AddMaterialScreen} />
                        <Stack.Screen name="Review" component={ReviewScreen} />
                        <Stack.Screen name="MaterialDetail" component={MaterialDetailScreen} />
                        <Stack.Screen name="Upgrade" component={UpgradeScreen} />
                    </>
                ) : (
                    <Stack.Screen name="Login" component={LoginScreen} />
                )}
            </Stack.Navigator>
        </NavigationContainer>
    );
};
