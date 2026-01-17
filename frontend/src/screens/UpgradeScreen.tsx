import React, { useState } from 'react';
import { View, Text, StyleSheet, TouchableOpacity, ScrollView, Alert, ActivityIndicator } from 'react-native';
import RazorpayCheckout from 'react-native-razorpay';
import { paymentClient } from '../services/directApi';
import { API_URL } from '../utils/config';
import { useAuthStore } from '../store/authStore';
import { Ionicons } from '@expo/vector-icons';
import { CommonActions, useNavigation } from '@react-navigation/native';

export const UpgradeScreen = () => {
    const { user } = useAuthStore();
    const navigation = useNavigation();
    const [loading, setLoading] = useState(false);

    const handleUpgrade = async () => {
        setLoading(true);
        try {
            // 1. Create Order on Backend
            const order = await paymentClient.createSubscriptionOrder({ planId: 'PRO' });

            // 2. Open Razorpay Checkout
            const options = {
                description: 'Upgrade to Pro Plan',
                image: `${API_URL}/logo.png`, // Optional: Add app logo URL
                currency: order.currency,
                key: order.keyId, // Key ID from backend
                amount: order.amount * 100, // Amount in paise (Backend returns INR float, so * 100)
                name: 'L.and.R Pro',
                order_id: order.orderId,
                prefill: {
                    email: user?.email || '',
                    contact: '', // Can be populated if available
                    name: user?.name || ''
                },
                theme: { color: '#0d6efd' }
            };

            RazorpayCheckout.open(options).then((data: any) => {
                // handle success
                console.log(`Success: ${data.razorpay_payment_id}`);
                Alert.alert('Payment Successful', 'Your plan has been upgraded to Pro!', [
                    { text: 'OK', onPress: () => navigation.dispatch(CommonActions.goBack()) }
                ]);
            }).catch((error: any) => {
                // handle failure
                console.log(`Error: ${error.code} | ${error.description}`);
                Alert.alert('Payment Failed', error.description || 'Something went wrong');
            });

        } catch (err) {
            console.error(err);
            Alert.alert('Error', 'Failed to initiate payment.');
        } finally {
            setLoading(false);
        }
    };

    const FeatureItem = ({ text }: { text: string }) => (
        <View style={styles.featureItem}>
            <Ionicons name="checkmark-circle" size={24} color="#28a745" />
            <Text style={styles.featureText}>{text}</Text>
        </View>
    );

    return (
        <ScrollView contentContainerStyle={styles.container}>
            <View style={styles.card}>
                <Text style={styles.title}>Upgrade to Pro</Text>
                <View style={styles.priceContainer}>
                    <Text style={styles.price}>₹199</Text>
                    <Text style={styles.period}>/ month</Text>
                </View>

                <View style={styles.features}>
                    <FeatureItem text="Unlimited AI Flashcards" />
                    <FeatureItem text="Personalized Daily Feed" />
                    <FeatureItem text="Detailed Analytics" />
                    <FeatureItem text="Priority Support" />
                </View>

                <TouchableOpacity
                    style={[styles.button, loading && styles.buttonDisabled]}
                    onPress={handleUpgrade}
                    disabled={loading}
                >
                    {loading ? (
                        <ActivityIndicator color="#fff" />
                    ) : (
                        <Text style={styles.buttonText}>Upgrade Now</Text>
                    )}
                </TouchableOpacity>
            </View>
        </ScrollView>
    );
};

const styles = StyleSheet.create({
    container: {
        flexGrow: 1,
        backgroundColor: '#f8f9fa',
        justifyContent: 'center',
        padding: 20,
    },
    card: {
        backgroundColor: 'white',
        borderRadius: 16,
        padding: 24,
        alignItems: 'center',
        shadowColor: '#000',
        shadowOffset: { width: 0, height: 2 },
        shadowOpacity: 0.1,
        shadowRadius: 8,
        elevation: 5,
    },
    title: {
        fontSize: 24,
        fontWeight: 'bold',
        marginBottom: 20,
        color: '#212529',
    },
    priceContainer: {
        flexDirection: 'row',
        alignItems: 'baseline',
        marginBottom: 30,
    },
    price: {
        fontSize: 48,
        fontWeight: 'bold',
        color: '#0d6efd',
    },
    period: {
        fontSize: 16,
        color: '#6c757d',
        marginLeft: 5,
    },
    features: {
        width: '100%',
        marginBottom: 30,
    },
    featureItem: {
        flexDirection: 'row',
        alignItems: 'center',
        marginBottom: 16,
    },
    featureText: {
        marginLeft: 10,
        fontSize: 16,
        color: '#495057',
    },
    button: {
        backgroundColor: '#0d6efd',
        width: '100%',
        paddingVertical: 16,
        borderRadius: 8,
        alignItems: 'center',
    },
    buttonDisabled: {
        opacity: 0.7,
    },
    buttonText: {
        color: 'white',
        fontSize: 18,
        fontWeight: 'bold',
    },
});
