import { useEffect, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { ArrowLeft, BookOpen, AlertCircle, Loader2, Trash2 } from 'lucide-react';
import { apiFetch } from '../api';

type Mistake = {
    id: number;
    wrong_text: string;
    correct_text: string;
    rule_explanation: string;
    created_at: string;
};

type Vocab = {
    id: number;
    word: string;
    translation: string;
    context_sentence: string;
    created_at: string;
};

export default function ChatNotebook() {
    const navigate = useNavigate();
    const [activeTab, setActiveTab] = useState<'mistakes' | 'vocab'>('vocab');

    const [mistakes, setMistakes] = useState<Mistake[]>([]);
    const [vocab, setVocab] = useState<Vocab[]>([]);
    const [isLoading, setIsLoading] = useState(true);

    useEffect(() => {
        fetchData();
    }, []);

    const fetchData = async () => {
        setIsLoading(true);
        try {
            const [mistakesData, vocabData] = await Promise.all([
                apiFetch('/practice/mistakes'),
                apiFetch('/practice/vocab')
            ]);
            setMistakes(Array.isArray(mistakesData) ? mistakesData : []);
            setVocab(Array.isArray(vocabData) ? vocabData : []);
        } catch (error) {
            console.error("Помилка завантаження зошита", error);
        } finally {
            setIsLoading(false);
        }
    };

    // --- ФУНКЦІЇ ВИДАЛЕННЯ ---
    const handleDeleteVocab = async (id: number) => {
        try {
            await apiFetch(`/practice/vocab?id=${id}`, { method: 'DELETE' });
            setVocab(prev => prev.filter(v => v.id !== id));
        } catch (error) {
            alert('Не вдалося видалити слово');
        }
    };

    const handleDeleteMistake = async (id: number) => {
        try {
            await apiFetch(`/practice/mistakes?id=${id}`, { method: 'DELETE' });
            setMistakes(prev => prev.filter(m => m.id !== id));
        } catch (error) {
            alert('Не вдалося видалити помилку');
        }
    };

    return (
        <div className="max-w-4xl mx-auto p-6 mt-8 space-y-6">
            <div className="flex items-center gap-4 mb-8">
                <button onClick={() => navigate(-1)} className="p-2 bg-surface border border-surfaceBorder rounded-xl hover:bg-surfaceBorder transition-colors">
                    <ArrowLeft className="w-5 h-5 text-textMain" />
                </button>
                <div>
                    <h1 className="text-3xl font-bold text-textMain tracking-tight">Мій зошит (AI Чат)</h1>
                    <p className="text-textMuted mt-1">Збережені слова та граматичні виправлення з бесід</p>
                </div>
            </div>

            {/* Вкладки */}
            <div className="flex gap-2 border-b border-surfaceBorder pb-4">
                <button
                    onClick={() => setActiveTab('vocab')}
                    className={`px-5 py-2.5 rounded-xl font-medium flex items-center gap-2 transition-colors ${activeTab === 'vocab' ? 'bg-primary text-white' : 'bg-surface text-textMuted hover:text-textMain border border-surfaceBorder'}`}
                >
                    <BookOpen className="w-4 h-4" /> Словник ({vocab.length})
                </button>
                <button
                    onClick={() => setActiveTab('mistakes')}
                    className={`px-5 py-2.5 rounded-xl font-medium flex items-center gap-2 transition-colors ${activeTab === 'mistakes' ? 'bg-primary text-white' : 'bg-surface text-textMuted hover:text-textMain border border-surfaceBorder'}`}
                >
                    <AlertCircle className="w-4 h-4" /> Помилки ({mistakes.length})
                </button>
            </div>

            {isLoading ? (
                <div className="flex justify-center py-20"><Loader2 className="w-8 h-8 animate-spin text-primary" /></div>
            ) : (
                <div className="space-y-4">
                    {activeTab === 'vocab' && (
                        vocab.length === 0 ? (
                            <div className="text-center py-12 text-textMuted bg-surface rounded-2xl border border-surfaceBorder">Словник порожній. Виділяйте слова в чаті, щоб додати їх сюди.</div>
                        ) : (
                            <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                                {vocab.map(v => (
                                    <div key={v.id} className="bg-surface border border-surfaceBorder p-5 rounded-2xl relative group flex flex-col justify-between">
                                        <div>
                                            <div className="flex justify-between items-start mb-1">
                                                <h3 className="text-xl font-bold text-primary">{v.word}</h3>
                                                {/* КНОПКА ВИДАЛЕННЯ СЛОВА */}
                                                <button
                                                    onClick={() => handleDeleteVocab(v.id)}
                                                    className="text-textMuted hover:text-red-400 p-1.5 rounded-lg transition-colors"
                                                    title="Видалити зі словника"
                                                >
                                                    <Trash2 className="w-4 h-4" />
                                                </button>
                                            </div>
                                            <p className="text-textMain font-medium mb-3">{v.translation}</p>
                                        </div>
                                        {v.context_sentence && (
                                            <p className="text-sm text-textMuted bg-mainBg p-3 rounded-xl border border-surfaceBorder italic mt-2">"{v.context_sentence}"</p>
                                        )}
                                    </div>
                                ))}
                            </div>
                        )
                    )}

                    {activeTab === 'mistakes' && (
                        mistakes.length === 0 ? (
                            <div className="text-center py-12 text-textMuted bg-surface rounded-2xl border border-surfaceBorder">У вас немає збережених помилок.</div>
                        ) : (
                            <div className="space-y-4">
                                {mistakes.map(m => (
                                    <div key={m.id} className="bg-surface border border-surfaceBorder p-5 rounded-2xl flex flex-col gap-3 relative">
                                        <div className="flex justify-between items-start">
                                            <div className="flex flex-col sm:flex-row sm:items-center gap-2 sm:gap-6">
                                                <span className="text-red-400 line-through">✕ {m.wrong_text}</span>
                                                <span className="text-green-400 font-bold">✓ {m.correct_text}</span>
                                            </div>
                                            {/* КНОПКА ВИДАЛЕННЯ ПОМИЛКИ */}
                                            <button
                                                onClick={() => handleDeleteMistake(m.id)}
                                                className="text-textMuted hover:text-red-400 p-1.5 rounded-lg transition-colors"
                                                title="Видалити помилку"
                                            >
                                                <Trash2 className="w-4 h-4" />
                                            </button>
                                        </div>
                                        <div className="text-sm text-blue-300 bg-blue-500/10 border border-blue-500/20 p-3 rounded-xl">
                                            <span className="font-bold mr-1">Правило:</span>{m.rule_explanation}
                                        </div>
                                    </div>
                                ))}
                            </div>
                        )
                    )}
                </div>
            )}
        </div>
    );
}