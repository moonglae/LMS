import { useState, useRef, useEffect } from 'react';
import { Send, Bot, User, Loader2, Save, CheckCircle2 } from 'lucide-react';
import { apiFetch } from '../api';

type Mistake = {
    wrong_text: string;
    correct_text: string;
    rule_explanation: string;
};

type Message = {
    id: string;
    sender: 'user' | 'ai';
    text: string;
    mistakes?: Mistake[];
};

export default function AIChat() {
    // Налаштування чату
    const [topic, setTopic] = useState('Ordering food in a restaurant');
    const [language, setLanguage] = useState('English');
    const [level, setLevel] = useState('A2');

    const [messages, setMessages] = useState<Message[]>([
        { id: '1', sender: 'ai', text: "Hello! I am your AI tutor. Let's practice. Send me a message when you're ready!" }
    ]);
    const [input, setInput] = useState('');
    const [isLoading, setIsLoading] = useState(false);
    const [savedMistakes, setSavedMistakes] = useState<Set<string>>(new Set());

    const messagesEndRef = useRef<HTMLDivElement>(null);

    // Автоматична прокрутка вниз при новому повідомленні
    useEffect(() => {
        messagesEndRef.current?.scrollIntoView({ behavior: 'smooth' });
    }, [messages]);

    const handleSendMessage = async (e: React.FormEvent) => {
        e.preventDefault();
        if (!input.trim() || isLoading) return;

        const userMessageText = input.trim();
        setInput('');

        // Додаємо повідомлення користувача в UI
        const newUserMsg: Message = { id: Date.now().toString(), sender: 'user', text: userMessageText };
        setMessages(prev => [...prev, newUserMsg]);
        setIsLoading(true);

        try {
            // Відправляємо на наш Go-сервер
            const data = await apiFetch('/practice/chat', {
                method: 'POST',
                body: JSON.stringify({
                    topic,
                    language,
                    level,
                    message: userMessageText
                })
            });

            // Додаємо відповідь ШІ в UI
            const newAiMsg: Message = {
                id: (Date.now() + 1).toString(),
                sender: 'ai',
                text: data.reply,
                mistakes: data.mistakes
            };
            setMessages(prev => [...prev, newAiMsg]);

        } catch (error: any) {
            // У разі помилки виводимо системне повідомлення
            setMessages(prev => [...prev, {
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
            await apiFetch('/practice/mistakes/save', {
                method: 'POST',
                body: JSON.stringify(mistake)
            });
            // Зберігаємо ключ, щоб показати галочку "Збережено"
            setSavedMistakes(prev => new Set(prev).add(mistakeKey));
        } catch (error) {
            alert('Помилка при збереженні. Спробуйте ще раз.');
        }
    };

    return (
        <div className="flex flex-col h-[calc(100vh-100px)] max-h-[800px] bg-surface border border-surfaceBorder rounded-2xl overflow-hidden shadow-lg">

            {/* Шапка з налаштуваннями */}
            <div className="bg-mainBg border-b border-surfaceBorder p-4 flex flex-wrap gap-4 items-center">
                <div className="flex items-center gap-2 text-purple-400 font-bold mr-auto">
                    <Bot className="w-6 h-6" />
                    <span>AI Tutor</span>
                </div>

                <div className="flex gap-2">
                    <input
                        type="text"
                        value={language}
                        onChange={(e) => setLanguage(e.target.value)}
                        className="bg-surface border border-surfaceBorder text-textMain text-sm rounded-lg px-3 py-1.5 w-24 outline-none focus:border-primary"
                        placeholder="Мова"
                    />
                    <input
                        type="text"
                        value={level}
                        onChange={(e) => setLevel(e.target.value)}
                        className="bg-surface border border-surfaceBorder text-textMain text-sm rounded-lg px-3 py-1.5 w-16 outline-none focus:border-primary"
                        placeholder="Рівень"
                    />
                    <input
                        type="text"
                        value={topic}
                        onChange={(e) => setTopic(e.target.value)}
                        className="bg-surface border border-surfaceBorder text-textMain text-sm rounded-lg px-3 py-1.5 w-48 outline-none focus:border-primary"
                        placeholder="Тема"
                    />
                </div>
            </div>

            {/* Зона чату */}
            <div className="flex-1 overflow-y-auto p-4 space-y-6">
                {messages.map((msg) => (
                    <div key={msg.id} className={`flex ${msg.sender === 'user' ? 'justify-end' : 'justify-start'}`}>
                        <div className={`max-w-[80%] md:max-w-[70%] flex gap-3 ${msg.sender === 'user' ? 'flex-row-reverse' : 'flex-row'}`}>

                            {/* Аватарка */}
                            <div className="mt-auto shrink-0">
                                {msg.sender === 'user' ? (
                                    <div className="bg-primary/20 p-2 rounded-full"><User className="w-5 h-5 text-primary" /></div>
                                ) : (
                                    <div className="bg-purple-500/20 p-2 rounded-full"><Bot className="w-5 h-5 text-purple-400" /></div>
                                )}
                            </div>

                            {/* Контент повідомлення */}
                            <div className="space-y-2">
                                <div className={`p-4 rounded-2xl ${msg.sender === 'user' ? 'bg-primary text-white rounded-br-sm' : 'bg-mainBg border border-surfaceBorder text-textMain rounded-bl-sm'}`}>
                                    {msg.text}
                                </div>

                                {/* Блок з помилками (якщо є) */}
                                {msg.mistakes && msg.mistakes.length > 0 && (
                                    <div className="bg-red-500/5 border border-red-500/20 rounded-xl p-4 mt-2 space-y-4">
                                        <p className="text-xs font-bold text-red-400 uppercase tracking-wider">Аналіз помилок:</p>
                                        {msg.mistakes.map((mistake, idx) => {
                                            // Унікальний ключ для перевірки чи збережено
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

            {/* Зона вводу */}
            <div className="bg-mainBg border-t border-surfaceBorder p-4">
                <form onSubmit={handleSendMessage} className="flex gap-2">
                    <input
                        type="text"
                        value={input}
                        onChange={(e) => setInput(e.target.value)}
                        placeholder="Напишіть повідомлення..."
                        disabled={isLoading}
                        className="flex-1 bg-surface border border-surfaceBorder text-textMain rounded-xl px-4 py-3 outline-none focus:border-primary transition-colors disabled:opacity-50"
                    />
                    <button
                        type="submit"
                        disabled={isLoading || !input.trim()}
                        className="bg-primary hover:bg-primaryHover disabled:bg-primary/50 text-white p-3 rounded-xl transition-colors flex items-center justify-center min-w-[50px]"
                    >
                        <Send className="w-5 h-5" />
                    </button>
                </form>
            </div>
        </div>
    );
}