import { useState, useEffect } from 'react';
import { useParams, useNavigate } from 'react-router-dom';
import { Loader2, ArrowLeft, BookOpen } from 'lucide-react';
import { apiFetch } from '../api';

export default function TheoryView() {
    const { id } = useParams();
    const navigate = useNavigate();
    const [theory, setTheory] = useState('');
    const [title, setTitle] = useState('');
    const [isLoading, setIsLoading] = useState(true);

    useEffect(() => {
        const fetchTheory = async () => {
            try {
                const modules = await apiFetch('/modules');
                const mod = modules.find((m: any) => m.id.toString() === id);
                if (mod) {
                    setTheory(mod.theory || 'У цьому модулі ще немає теорії.');
                    setTitle(mod.title);
                }
            } catch (err) {
                console.error("Помилка завантаження:", err);
            } finally {
                setIsLoading(false);
            }
        };
        fetchTheory();
    }, [id]);

    if (isLoading) return <div className="flex justify-center mt-20"><Loader2 className="animate-spin text-primary" /></div>;

    return (
        <div className="max-w-3xl mx-auto p-6 mt-8">
            <button onClick={() => navigate(-1)} className="flex items-center gap-2 text-textMuted hover:text-white transition-colors mb-6">
                <ArrowLeft size={20} /> Назад
            </button>

            <div className="bg-surface p-8 rounded-3xl border border-surfaceBorder shadow-xl">
                <h1 className="text-3xl font-bold mb-6 flex items-center gap-3 text-textMain">
                    <BookOpen className="text-primary" /> {title}
                </h1>
                {/* whitespace-pre-line зберігає переноси рядків з textarea */}
                <div className="text-textMain leading-relaxed whitespace-pre-line text-lg">
                    {theory}
                </div>
            </div>
        </div>
    );
}