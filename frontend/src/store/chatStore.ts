import { create } from 'zustand';

export type Mistake = {
    wrong_text: string;
    correct_text: string;
    rule_explanation: string;
};

export type Message = {
    id: string;
    sender: 'user' | 'ai';
    text: string;
    mistakes?: Mistake[];
};

interface ChatState {
    topic: string;
    language: string;
    level: string;
    messages: Message[];

    setTopic: (topic: string) => void;
    setLanguage: (lang: string) => void;
    setLevel: (level: string) => void;
    setMessages: (updater: Message[] | ((prev: Message[]) => Message[])) => void;
    resetChat: () => void;
}

export const useChatStore = create<ChatState>((set) => ({
    topic: 'Ordering food in a restaurant',
    language: 'English',
    level: 'A2',
    messages: [
        { id: '1', sender: 'ai', text: "Hello! I am your AI tutor. Let's practice. Send me a message when you're ready!" }
    ],

    setTopic: (topic) => set({ topic }),
    setLanguage: (language) => set({ language }),
    setLevel: (level) => set({ level }),

    setMessages: (updater) => set((state) => ({
        messages: typeof updater === 'function' ? updater(state.messages) : updater
    })),

    resetChat: () => set({
        topic: 'Ordering food in a restaurant',
        language: 'English',
        level: 'A2',
        messages: [{ id: '1', sender: 'ai', text: "Hello! I am your AI tutor. Let's practice. Send me a message when you're ready!" }]
    })
}));