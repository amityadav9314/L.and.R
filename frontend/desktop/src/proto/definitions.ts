import { LoginRequest, LoginResponse } from './backend/proto/auth/auth';
import {
    AddMaterialRequest, AddMaterialResponse,
    FlashcardList,
    CompleteReviewRequest, FailReviewRequest, UpdateFlashcardRequest,
    DeleteMaterialRequest,
    GetDueMaterialsRequest, GetDueMaterialsResponse,
    GetDueFlashcardsRequest,
    GetAllTagsResponse,
    NotificationStatusResponse,
    GetMaterialSummaryRequest, GetMaterialSummaryResponse,
    RegisterPushTokenRequest,
} from './backend/proto/learning/learning';
import { Empty } from './backend/google/protobuf/empty';

export const AuthServiceDefinition = {
    name: 'AuthService',
    fullName: 'auth.AuthService',
    methods: {
        login: {
            name: 'Login',
            requestType: LoginRequest,
            requestStream: false,
            responseType: LoginResponse,
            responseStream: false,
            options: {},
        },
    },
} as const;

export const LearningServiceDefinition = {
    name: 'LearningService',
    fullName: 'learning.LearningService',
    methods: {
        addMaterial: {
            name: 'AddMaterial',
            requestType: AddMaterialRequest,
            requestStream: false,
            responseType: AddMaterialResponse,
            responseStream: false,
            options: {},
        },
        getDueMaterials: {
            name: 'GetDueMaterials',
            requestType: GetDueMaterialsRequest,
            requestStream: false,
            responseType: GetDueMaterialsResponse,
            responseStream: false,
            options: {},
        },
        getDueFlashcards: {
            name: 'GetDueFlashcards',
            requestType: GetDueFlashcardsRequest,
            requestStream: false,
            responseType: FlashcardList,
            responseStream: false,
            options: {},
        },
        completeReview: {
            name: 'CompleteReview',
            requestType: CompleteReviewRequest,
            requestStream: false,
            responseType: Empty,
            responseStream: false,
            options: {},
        },
        failReview: {
            name: 'FailReview',
            requestType: FailReviewRequest,
            requestStream: false,
            responseType: Empty,
            responseStream: false,
            options: {},
        },
        updateFlashcard: {
            name: 'UpdateFlashcard',
            requestType: UpdateFlashcardRequest,
            requestStream: false,
            responseType: Empty,
            responseStream: false,
            options: {},
        },
        deleteMaterial: {
            name: 'DeleteMaterial',
            requestType: DeleteMaterialRequest,
            requestStream: false,
            responseType: Empty,
            responseStream: false,
            options: {},
        },
        getAllTags: {
            name: 'GetAllTags',
            requestType: Empty,
            requestStream: false,
            responseType: GetAllTagsResponse,
            responseStream: false,
            options: {},
        },
        getNotificationStatus: {
            name: 'GetNotificationStatus',
            requestType: Empty,
            requestStream: false,
            responseType: NotificationStatusResponse,
            responseStream: false,
            options: {},
        },
        getMaterialSummary: {
            name: 'GetMaterialSummary',
            requestType: GetMaterialSummaryRequest,
            requestStream: false,
            responseType: GetMaterialSummaryResponse,
            responseStream: false,
            options: {},
        },
        registerPushToken: {
            name: 'RegisterPushToken',
            requestType: RegisterPushTokenRequest,
            requestStream: false,
            responseType: Empty,
            responseStream: false,
            options: {},
        },
    },
} as const;

// Feed Service (Daily AI Feed)
import {
    UpdateFeedPreferencesRequest,
    FeedPreferencesResponse,
    GetDailyFeedRequest, GetDailyFeedResponse,
    GetFeedCalendarStatusRequest, GetFeedCalendarStatusResponse
} from './backend/proto/feed/feed';

export const FeedServiceDefinition = {
    name: 'FeedService',
    fullName: 'feed.FeedService',
    methods: {
        updateFeedPreferences: {
            name: 'UpdateFeedPreferences',
            requestType: UpdateFeedPreferencesRequest,
            requestStream: false,
            responseType: Empty,
            responseStream: false,
            options: {},
        },
        getFeedPreferences: {
            name: 'GetFeedPreferences',
            requestType: Empty,
            requestStream: false,
            responseType: FeedPreferencesResponse,
            responseStream: false,
            options: {},
        },
        getDailyFeed: {
            name: 'GetDailyFeed',
            requestType: GetDailyFeedRequest,
            requestStream: false,
            responseType: GetDailyFeedResponse,
            responseStream: false,
            options: {},
        },
        getFeedCalendarStatus: {
            name: 'GetFeedCalendarStatus',
            requestType: GetFeedCalendarStatusRequest,
            requestStream: false,
            responseType: GetFeedCalendarStatusResponse,
            responseStream: false,
            options: {},
        },
    },
} as const;
