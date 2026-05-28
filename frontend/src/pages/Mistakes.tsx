import { useEffect, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { AlertTriangle, ArrowLeft, CheckCircle2, Loader2, PlayCircle } from 'lucide-react';
import { apiFetch } from '../api';

interface Mistake {
    id: number;
    question_text: string;
    module_id: number;
    module_title: string;
}

export default function Mistakes() {
    const navigate = useNavigate();
    const [mistakes, setMistakes] = useState<Mistake[]>([]);
    const [isLoading, setIsLoading] = useState(true);
    const [error, setError] = useState<string | null>(null);

    useEffect(() => {
        const fetchMistakes = async () => {
            try {
                setIsLoading(true);
                const data = await apiFetch('/analytics/mistakes');
                setMistakes(Array.isArray(data) ? data : []);
            } catch (err: any) {
                setError(err.message || 'Не вдалося завантажити помилки.');
            } finally {
                setIsLoading(false);
            }
        };
        fetchMistakes();
    }, []);

    if (isLoading) return <div className="flex justify-center mt-20"><Loader2 className="w-8 h-8 animate-spin text-primary" /></div>;

    // Групуємо помилки по модулях
    const groupedMistakes = mistakes.reduce((acc, m) => {
        if (!acc[m.module_id]) {
            acc[m.module_id] = { title: m.module_title, items: [] };
        }
        acc[m.module_id].items.push(m);
        return acc;
    }, {} as Record<number, { title: string, items: Mistake[] }>);

    return (
        <div className="max-w-3xl mx-auto mt-8 space-y-8 p-4">
            <div className="flex items-center gap-4">
                <button onClick={() => navigate('/')} className="p-2 bg-surface border border-surfaceBorder rounded-xl hover:bg-surfaceBorder transition-colors">
                    <ArrowLeft className="w-5 h-5 text-textMain" />
                </button>
                <div>
                    <h1 className="text-3xl font-bold text-textMain flex items-center gap-3">
                        Робота над помилками <AlertTriangle className="w-6 h-6 text-yellow-500" />
                    </h1>
                    <p className="text-textMuted mt-1">Виправляйте помилки, проходячи тести по модулях.</p>
                </div>
            </div>

            {error && <div className="p-4 bg-red-500/10 border border-red-500/20 rounded-2xl text-red-400">{error}</div>}

            {mistakes.length === 0 ? (
                <div className="bg-surface border border-surfaceBorder rounded-3xl p-12 text-center">
                    <CheckCircle2 className="w-16 h-16 text-green-500 mx-auto mb-4" />
                    <h3 className="text-xl font-bold text-textMain">Чудова робота!</h3>
                    <p className="text-textMuted mt-2">У вас немає невирішених помилок.</p>
                </div>
            ) : (
                <div className="space-y-6">
                    {Object.entries(groupedMistakes).map(([moduleId, group]) => (
                        <div key={moduleId} className="bg-surface border border-surfaceBorder rounded-2xl p-6">
                            <div className="flex flex-col sm:flex-row sm:items-center justify-between mb-4 gap-4">
                                <div>
                                    <h3 className="text-xl font-bold text-textMain">{group.title}</h3>
                                    <p className="text-sm text-textMuted">Помилок: {group.items.length}</p>
                                </div>
                                <button
                                    onClick={() => navigate(`/quiz/mistakes?module_id=${moduleId}`)}
                                    className="inline-flex shrink-0 items-center justify-center gap-2 rounded-xl bg-primary px-5 py-2.5 text-sm font-semibold text-white hover:bg-primaryHover transition-colors"
                                >
                                    <PlayCircle className="w-5 h-5" /> Виправити помилки
                                </button>
                            </div>

                            <ul className="space-y-2 mt-4 pt-4 border-t border-surfaceBorder">
                                {group.items.map(item => (
                                    <li key={item.id} className="text-textMain bg-background p-3 rounded-xl border border-surfaceBorder">
                                        <span className="text-red-500 mr-2">✕</span> {item.question_text}
                                    </li>
                                ))}
                            </ul>
                        </div>
                    ))}
                </div>
            )}
        </div>
    );
}