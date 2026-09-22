import { useState, useRef, useEffect } from 'react';
import { useNavigate } from 'react-router-dom';
import { Send, Bot, User, Loader2, Save, CheckCircle2, Mic, MicOff, BookPlus, BookOpen, X, RefreshCw } from 'lucide-react';
import { apiFetch } from '../api';
import { useChatStore } from '../store/chatStore';
import type { Mistake, Message } from '../store/chatStore';

export default function AIChat() {
    const navigate = useNavigate();

    // --- ГЛОБАЛЬНИЙ СТАН ЧАТУ З ZUSTAND ---
    const {
        topic, setTopic,
        language, setLanguage,
        level, setLevel,
        messages, setMessages,
        resetChat
    } = useChatStore();

    // --- ЛОКАЛЬНІ СТАНИ ---
    const [input, setInput] = useState('');
    const [isLoading, setIsLoading] = useState(false);

    const [savedMistakes, setSavedMistakes] = useState<Set<string>>(new Set());

    const [isVocabModalOpen, setIsVocabModalOpen] = useState(false);
    const [vocabForm, setVocabForm] = useState({ word: '', translation: '', context_sentence: '' });
    const [isSavingVocab, setIsSavingVocab] = useState(false);

    const [selectionPopup, setSelectionPopup] = useState<{ text: string, context: string, x: number, y: number } | null>(null);

    const [isListening, setIsListening] = useState(false);
    const recognitionRef = useRef<any>(null);
    const messagesEndRef = useRef<HTMLDivElement>(null);

    useEffect(() => {
        messagesEndRef.current?.scrollIntoView({ behavior: 'smooth' });
    }, [messages]);

    useEffect(() => {
        const handleSelectionChange = () => {
            const selection = window.getSelection();
            if (!selection || selection.toString().trim() === '') {
                setSelectionPopup(null);
            }
        };
        document.addEventListener('selectionchange', handleSelectionChange);
        return () => document.removeEventListener('selectionchange', handleSelectionChange);
    }, []);

    useEffect(() => {
        const SpeechRecognition = (window as any).SpeechRecognition || (window as any).webkitSpeechRecognition;
        if (SpeechRecognition) {
            recognitionRef.current = new SpeechRecognition();
            recognitionRef.current.lang = 'en-US';
            recognitionRef.current.continuous = false;
            recognitionRef.current.interimResults = false;

            recognitionRef.current.onresult = (event: any) => {
                const transcript = event.results[0][0].transcript;
                setInput((prev) => prev + (prev ? ' ' : '') + transcript);
                setIsListening(false);
            };

            recognitionRef.current.onerror = (event: any) => {
                console.error("Помилка розпізнавання:", event.error);
                setIsListening(false);
            };

            recognitionRef.current.onend = () => {
                setIsListening(false);
            };
        }
    }, []);

    const toggleListening = () => {
        if (isListening) {
            recognitionRef.current?.stop();
            setIsListening(false);
        } else {
            recognitionRef.current?.start();
            setIsListening(true);
        }
    };

    const handleSendMessage = async (e: React.FormEvent) => {
        e.preventDefault();
        if (!input.trim() || isLoading) return;

        const userMessageText = input.trim();
        setInput('');

        if (isListening) {
            recognitionRef.current?.stop();
            setIsListening(false);
        }

        const newUserMsg: Message = { id: Date.now().toString(), sender: 'user', text: userMessageText };
        setMessages((prev: Message[]) => [...prev, newUserMsg]);
        setIsLoading(true);

        try {
            const data = await apiFetch('/practice/chat', {
                method: 'POST',
                body: JSON.stringify({ topic, language, level, message: userMessageText })
            });

            const newAiMsg: Message = {
                id: (Date.now() + 1).toString(),
                sender: 'ai',
                text: data.reply,
                mistakes: data.mistakes
            };
            setMessages((prev: Message[]) => [...prev, newAiMsg]);

        } catch (error: any) {
            setMessages((prev: Message[]) => [...prev, {
                id: Date.now().toString(),
                sender: 'ai',
                text: `Помилка: ${error.message || 'Не вдалося зв\'язатися з ШІ.'}`
            }]);
        } finally {
            setIsLoading(false);
        }
    };

    const handleSaveMistake = async (mistake: Mistake, mistakeKey: string) => {
        try {
            await apiFetch('/practice/mistakes', {
                method: 'POST',
                body: JSON.stringify(mistake)
            });
            setSavedMistakes(prev => new Set(prev).add(mistakeKey));
        } catch (error) {
            alert('Помилка при збереженні. Спробуйте ще раз.');
        }
    };

    const handleTextSelection = () => {
        setTimeout(() => {
            const selection = window.getSelection();
            const selectedText = selection?.toString().trim() || '';

            if (selectedText && selectedText.length > 1 && selection && selection.rangeCount > 0) {
                const range = selection.getRangeAt(0);
                const rect = range.getBoundingClientRect();

                let contextText = '';
                const messageElement = selection.anchorNode?.parentElement?.closest('.message-content');
                if (messageElement) {
                    contextText = (messageElement as HTMLElement).innerText;
                }

                setSelectionPopup({
                    text: selectedText,
                    context: contextText,
                    x: rect.left + rect.width / 2,
                    y: rect.top
                });
            } else {
                setSelectionPopup(null);
            }
        }, 10);
    };

    const handleSaveVocabulary = async (e: React.FormEvent) => {
        e.preventDefault();
        if (!vocabForm.word || !vocabForm.translation) return;

        setIsSavingVocab(true);
        try {
            await apiFetch('/practice/vocab', {
                method: 'POST',
                body: JSON.stringify(vocabForm)
            });
            setIsVocabModalOpen(false);
            setVocabForm({ word: '', translation: '', context_sentence: '' });
        } catch (error) {
            alert('Помилка збереження слова. Спробуйте ще раз.');
        } finally {
            setIsSavingVocab(false);
        }
    };

    return (
        <div className="flex flex-col h-[calc(100vh-100px)] max-h-[800px] bg-surface border border-surfaceBorder rounded-2xl overflow-hidden shadow-lg relative">

            {/* ПЛАВАЮЧА КНОПКА (TOOLTIP) */}
            {selectionPopup && (
                <div
                    className="fixed z-40 transform -translate-x-1/2 -translate-y-full pb-2 animate-in fade-in zoom-in duration-200"
                    style={{ left: selectionPopup.x, top: selectionPopup.y }}
                >
                    <button
                        onClick={(e) => {
                            e.preventDefault();
                            e.stopPropagation();
                            setVocabForm({
                                word: selectionPopup.text,
                                translation: '',
                                context_sentence: selectionPopup.context
                            });
                            setIsVocabModalOpen(true);
                            setSelectionPopup(null);
                        }}
                        className="bg-primary text-white shadow-[0_10px_25px_-5px_rgba(0,0,0,0.5)] flex items-center gap-2 px-3 py-2 rounded-xl text-sm font-medium hover:bg-primaryHover transition-colors border border-white/20"
                    >
                        <BookPlus className="w-4 h-4" /> В словник
                    </button>
                    <div className="absolute bottom-0 left-1/2 -translate-x-1/2 translate-y-[4px] border-[6px] border-transparent border-t-primary"></div>
                </div>
            )}

            {/* МОДАЛЬНЕ ВІКНО ДОДАВАННЯ СЛОВА */}
            {isVocabModalOpen && (
                <div className="absolute inset-0 z-50 bg-black/60 flex items-center justify-center p-4 backdrop-blur-sm">
                    <div className="bg-mainBg border border-surfaceBorder w-full max-w-md rounded-2xl p-6 shadow-2xl relative">
                        <button onClick={() => setIsVocabModalOpen(false)} className="absolute top-4 right-4 text-textMuted hover:text-textMain transition-colors">
                            <X className="w-5 h-5" />
                        </button>

                        <h3 className="text-lg font-bold text-textMain mb-4 flex items-center gap-2">
                            <BookPlus className="w-5 h-5 text-primary" /> Додати в словник
                        </h3>

                        <form onSubmit={handleSaveVocabulary} className="space-y-4">
                            <div>
                                <label className="block text-xs text-textMuted mb-1">Виділене слово або фраза *</label>
                                <input
                                    type="text" required value={vocabForm.word}
                                    onChange={e => setVocabForm(prev => ({ ...prev, word: e.target.value }))}
                                    className="w-full bg-surface border border-surfaceBorder text-textMain rounded-xl px-3 py-2 outline-none focus:border-primary"
                                />
                            </div>
                            <div>
                                <label className="block text-xs text-textMuted mb-1">Введіть переклад *</label>
                                <input
                                    type="text" required value={vocabForm.translation} autoFocus
                                    onChange={e => setVocabForm(prev => ({ ...prev, translation: e.target.value }))}
                                    className="w-full bg-surface border border-surfaceBorder text-textMain rounded-xl px-3 py-2 outline-none focus:border-primary"
                                />
                            </div>
                            <div>
                                <label className="block text-xs text-textMuted mb-1">Контекст (речення з чату)</label>
                                <textarea
                                    value={vocabForm.context_sentence}
                                    onChange={e => setVocabForm(prev => ({ ...prev, context_sentence: e.target.value }))}
                                    rows={2}
                                    className="w-full bg-surface border border-surfaceBorder text-textMain rounded-xl px-3 py-2 outline-none focus:border-primary resize-none"
                                />
                            </div>
                            <button
                                type="submit" disabled={isSavingVocab || !vocabForm.word || !vocabForm.translation}
                                className="w-full bg-primary hover:bg-primaryHover disabled:bg-primary/50 text-white font-medium py-2.5 rounded-xl transition-colors flex items-center justify-center gap-2"
                            >
                                {isSavingVocab ? <Loader2 className="w-4 h-4 animate-spin" /> : <Save className="w-4 h-4" />} Зберегти
                            </button>
                        </form>
                    </div>
                </div>
            )}

            {/* ВЕРХНЯ ПАНЕЛЬ */}
            <div className="bg-mainBg border-b border-surfaceBorder p-4 flex flex-wrap gap-4 items-center justify-between">
                <div className="flex items-center gap-2 text-purple-400 font-bold">
                    <Bot className="w-6 h-6" /><span>AI Tutor</span>
                </div>

                <div className="flex gap-2 items-center flex-wrap">
                    <input type="text" value={language} onChange={(e) => setLanguage(e.target.value)} className="bg-surface border border-surfaceBorder text-textMain text-sm rounded-lg px-3 py-1.5 w-24 outline-none focus:border-primary hidden sm:block" placeholder="Мова" />
                    <input type="text" value={level} onChange={(e) => setLevel(e.target.value)} className="bg-surface border border-surfaceBorder text-textMain text-sm rounded-lg px-3 py-1.5 w-16 outline-none focus:border-primary hidden sm:block" placeholder="Рівень" />
                    <input type="text" value={topic} onChange={(e) => setTopic(e.target.value)} className="bg-surface border border-surfaceBorder text-textMain text-sm rounded-lg px-3 py-1.5 w-32 md:w-48 outline-none focus:border-primary" placeholder="Тема" />

                    <button
                        onClick={resetChat}
                        className="ml-2 bg-surface hover:bg-surfaceBorder border border-surfaceBorder text-textMuted p-1.5 rounded-lg transition-colors flex items-center gap-1.5 px-3"
                        title="Очистити чат і почати нову розмову"
                    >
                        <RefreshCw className="w-4 h-4" />
                    </button>

                    <button
                        onClick={() => navigate('/notebook')}
                        className="bg-purple-500/10 hover:bg-purple-500/20 border border-purple-500/20 text-purple-400 p-1.5 rounded-lg transition-colors flex items-center gap-1.5 px-3"
                        title="Відкрити зошит з помилками та словником"
                    >
                        <BookOpen className="w-4 h-4" />
                        <span className="text-sm font-medium hidden md:block">Мій зошит</span>
                    </button>
                </div>
            </div>

            {/* СПИСОК ПОВІДОМЛЕНЬ */}
            <div
                className="flex-1 overflow-y-auto p-4 space-y-6"
                onMouseUp={handleTextSelection}
                onTouchEnd={handleTextSelection}
            >
                {messages.map((msg: Message) => (
                    <div key={msg.id} className={`flex ${msg.sender === 'user' ? 'justify-end' : 'justify-start'}`}>
                        <div className={`max-w-[85%] md:max-w-[70%] flex gap-3 ${msg.sender === 'user' ? 'flex-row-reverse' : 'flex-row'}`}>
                            <div className="mt-auto shrink-0">
                                {msg.sender === 'user' ? (
                                    <div className="bg-primary/20 p-2 rounded-full"><User className="w-5 h-5 text-primary" /></div>
                                ) : (
                                    <div className="bg-purple-500/20 p-2 rounded-full"><Bot className="w-5 h-5 text-purple-400" /></div>
                                )}
                            </div>

                            <div className="space-y-2">
                                <div className={`message-content p-4 rounded-2xl ${msg.sender === 'user' ? 'bg-primary text-white rounded-br-sm' : 'bg-mainBg border border-surfaceBorder text-textMain rounded-bl-sm'}`}>
                                    {msg.text}
                                </div>

                                {msg.mistakes && msg.mistakes.length > 0 && (
                                    <div className="bg-red-500/5 border border-red-500/20 rounded-xl p-4 mt-2 space-y-4">
                                        <p className="text-xs font-bold text-red-400 uppercase tracking-wider">Аналіз помилок:</p>
                                        {msg.mistakes.map((mistake: Mistake, idx: number) => {
                                            const mistakeKey = `${msg.id}-${idx}`;
                                            const isSaved = savedMistakes.has(mistakeKey);

                                            return (
                                                <div key={idx} className="bg-mainBg rounded-lg p-3 border border-surfaceBorder text-sm">
                                                    <div className="flex items-center gap-2 text-red-400 mb-1">
                                                        <span className="line-through">{mistake.wrong_text}</span>
                                                    </div>
                                                    <div className="flex items-center gap-2 text-green-400 font-medium mb-2">
                                                        <span>{mistake.correct_text}</span>
                                                    </div>
                                                    <p className="text-textMuted text-xs mb-3">
                                                        <span className="text-blue-400">Правило:</span> {mistake.rule_explanation}
                                                    </p>
                                                    <button
                                                        onClick={() => handleSaveMistake(mistake, mistakeKey)}
                                                        disabled={isSaved}
                                                        className={`text-xs px-3 py-1.5 rounded-lg flex items-center gap-1.5 transition-colors ${isSaved ? 'bg-green-500/20 text-green-400 cursor-default' : 'bg-surface border border-surfaceBorder hover:border-primary text-textMain'}`}
                                                    >
                                                        {isSaved ? <><CheckCircle2 className="w-3.5 h-3.5" /> Збережено</> : <><Save className="w-3.5 h-3.5" /> В зошит</>}
                                                    </button>
                                                </div>
                                            );
                                        })}
                                    </div>
                                )}
                            </div>
                        </div>
                    </div>
                ))}
                {isLoading && (
                    <div className="flex justify-start">
                        <div className="bg-mainBg border border-surfaceBorder p-4 rounded-2xl rounded-bl-sm flex gap-2 items-center">
                            <Loader2 className="w-4 h-4 animate-spin text-purple-400" />
                            <span className="text-textMuted text-sm">AI друкує...</span>
                        </div>
                    </div>
                )}
                <div ref={messagesEndRef} />
            </div>

            {/* ПАНЕЛЬ ВВОДУ */}
            <div className="bg-mainBg border-t border-surfaceBorder p-4 relative z-10">
                <form onSubmit={handleSendMessage} className="flex gap-2">
                    <button
                        type="button" onClick={toggleListening}
                        className={`p-3 rounded-xl transition-colors flex items-center justify-center min-w-[50px] ${isListening ? 'bg-red-500 text-white animate-pulse' : 'bg-surface border border-surfaceBorder text-textMain hover:border-primary'}`}
                    >
                        {isListening ? <MicOff className="w-5 h-5" /> : <Mic className="w-5 h-5" />}
                    </button>
                    <input
                        type="text" value={input} onChange={(e) => setInput(e.target.value)}
                        placeholder="Напишіть повідомлення..." disabled={isLoading}
                        className="flex-1 bg-surface border border-surfaceBorder text-textMain rounded-xl px-4 py-3 outline-none focus:border-primary transition-colors disabled:opacity-50"
                    />
                    <button
                        type="submit" disabled={isLoading || !input.trim()}
                        className="bg-primary hover:bg-primaryHover disabled:bg-primary/50 text-white p-3 rounded-xl transition-colors flex items-center justify-center min-w-[50px]"
                    >
                        <Send className="w-5 h-5" />
                    </button>
                </form>
            </div>
        </div>
    );
}