import { create } from 'zustand';

interface FilterState {
    onlyDue: boolean;
    searchQuery: string;
    inputValue: string;
    selectedTags: string[];
    setOnlyDue: (onlyDue: boolean) => void;
    setSearchQuery: (query: string) => void;
    setInputValue: (value: string) => void;
    setSelectedTags: (tags: string[] | ((prev: string[]) => string[])) => void;
    toggleOnlyDue: () => void;
    resetAll: () => void;
}

export const useFilterStore = create<FilterState>((set) => ({
    onlyDue: true,
    searchQuery: '',
    inputValue: '',
    selectedTags: [],
    setOnlyDue: (onlyDue) => set({ onlyDue }),
    setSearchQuery: (searchQuery) => set({ searchQuery }),
    setInputValue: (inputValue) => set({ inputValue }),
    setSelectedTags: (tags) => set((state) => ({
        selectedTags: typeof tags === 'function' ? tags(state.selectedTags) : tags
    })),
    toggleOnlyDue: () => set((state) => ({ onlyDue: !state.onlyDue })),
    resetAll: () => set({
        onlyDue: true,
        searchQuery: '',
        inputValue: '',
        selectedTags: [],
    }),
}));
