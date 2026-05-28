import { useState, useEffect } from 'react';
import { useNavigate, useParams } from 'react-router-dom';
import { Trash2, Plus, Loader2, Save, ArrowLeft, BookOpen } from 'lucide-react';
import { apiFetch } from '../api';

export default function ModuleEditor() {
    const { id } = useParams();
    const navigate = useNavigate();
    const isEdit = !!id;

    const [title, setTitle] = useState('');
    const [desc, setDesc] = useState('');
    const [theory, setTheory] = useState('');
    const [cards, setCards] = useState([{ question: '', answer: '' }]);
    const [loading, setLoading] = useState(false);

    useEffect(() => {
        if (isEdit) loadData();
    }, [id, isEdit]);

    const loadData = async () => {
        setLoading(true);
        try {
            const modules = await apiFetch('/modules');
            const mod = modules.find((m: any) => m.id.toString() === id);
            if (mod) {
                setTitle(mod.title);
                setDesc(mod.description);
                setTheory(mod.theory || '');
            }
            const flashcards = await apiFetch(`/modules/flashcards?module_id=${id}`);
            // Важливо: якщо прийшли дані, встановлюємо їх
            if (Array.isArray(flashcards) && flashcards.length > 0) {
                setCards(flashcards);
            }
        } catch (err) {
            console.error("Помилка завантаження:", err);
        } finally {
            setLoading(false);
        }
    };

    // Оновлена функція зміни картки - тепер надійніша
    const handleCardChange = (index: number, field: 'question' | 'answer', value: string) => {
        const newCards = [...cards];
        newCards[index][field] = value;
        setCards(newCards);
    };

    const handleSubmit = async (e: React.FormEvent) => {
        e.preventDefault();
        setLoading(true);

        // DEBUG: Перевірка перед відправкою
        console.log("DEBUG: Відправляємо на сервер:", { title, desc, theory, cards });

        try {
            const payload = {
                title,
                description: desc,
                theory,
                cards: cards // Це саме те, що чекає бекенд
            };

            if (isEdit) {
                await apiFetch(`/modules/${id}`, {
                    method: 'PUT',
                    body: JSON.stringify(payload)
                });
            } else {
                await apiFetch('/modules', {
                    method: 'POST',
                    body: JSON.stringify(payload)
                });
            }
            navigate('/teacher');
        } catch (err) {
            console.error("Помилка збереження:", err);
            alert("Помилка при збереженні (перевірте консоль F12)");
        } finally {
            setLoading(false);
        }
    };

    return (
        <div className="max-w-3xl mx-auto p-8 space-y-6">
            <button onClick={() => navigate(-1)} className="flex items-center gap-2 text-textMuted hover:text-white">
                <ArrowLeft size={20} /> Назад
            </button>

            <h1 className="text-2xl font-bold">{isEdit ? 'Редагувати модуль' : 'Створити новий модуль'}</h1>

            <form onSubmit={handleSubmit} className="bg-surface p-6 rounded-3xl border border-surfaceBorder space-y-4">
                <input required className="w-full bg-mainBg p-3 rounded-xl border border-surfaceBorder focus:border-primary outline-none" placeholder="Назва курсу" value={title} onChange={e => setTitle(e.target.value)} />
                <textarea className="w-full bg-mainBg p-3 rounded-xl border border-surfaceBorder focus:border-primary outline-none" placeholder="Опис" value={desc} onChange={e => setDesc(e.target.value)} />

                <div className="space-y-2 pt-2">
                    <label className="text-sm font-bold text-textMuted flex items-center gap-2">
                        <BookOpen size={16} /> Теорія
                    </label>
                    <textarea
                        className="w-full bg-mainBg p-3 rounded-xl border border-surfaceBorder focus:border-primary outline-none min-h-[200px]"
                        value={theory}
                        onChange={e => setTheory(e.target.value)}
                    />
                </div>

                <h3 className="font-bold pt-4">Картки</h3>
                {cards.map((c, i) => (
                    <div key={i} className="flex gap-2">
                        <input
                            className="flex-1 bg-mainBg p-2 rounded-lg border border-surfaceBorder"
                            placeholder="Термін"
                            value={c.question}
                            onChange={e => handleCardChange(i, 'question', e.target.value)}
                        />
                        <input
                            className="flex-1 bg-mainBg p-2 rounded-lg border border-surfaceBorder"
                            placeholder="Визначення"
                            value={c.answer}
                            onChange={e => handleCardChange(i, 'answer', e.target.value)}
                        />
                        <button type="button" onClick={() => setCards(cards.filter((_, idx) => idx !== i))} className="text-red-500">
                            <Trash2 size={18} />
                        </button>
                    </div>
                ))}

                <button type="button" onClick={() => setCards([...cards, { question: '', answer: '' }])} className="text-primary underline flex items-center gap-1 text-sm">
                    <Plus size={16} /> Додати картку
                </button>

                <button disabled={loading} className="w-full bg-green-600 hover:bg-green-700 py-3 rounded-xl text-white font-bold flex justify-center gap-2 transition-all">
                    {loading ? <Loader2 className="animate-spin" /> : <Save size={20} />}
                    Зберегти
                </button>
            </form>
        </div>
    );
}