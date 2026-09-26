import { useState, useEffect } from 'react';
import { useNavigate, useParams, useSearchParams, useLocation } from 'react-router-dom';
import { CheckCircle, XCircle, Loader2, Shuffle as ShuffleIcon, Settings, List, Type } from 'lucide-react';
import { apiFetch } from '../api';

// 1. Структура того, що приходить з бекенду
interface RawBackendQuestion {
    flashcard_id: number;
    question: string;
    options: string[];
    answer: string;
}

// 2. Структура для фронтенду після обробки режимів (choice/fill/mix)
interface GeneratedQuestion {
    flashcard_id: number;
    question_text: string;
    options: string[];
    correct: string;
    type: 'choice' | 'fill';
}

type QuizPhase = 'loading' | 'setup' | 'testing' | 'error';

export default function Quiz() {
    const { id } = useParams();
    const navigate = useNavigate();
    const [searchParams] = useSearchParams();
    const location = useLocation();

    const isMistakesMode = location.pathname.includes('/quiz/mistakes');
    const targetModuleId = isMistakesMode ? searchParams.get('module_id') : id;

    // Стан для налаштувань
    const [phase, setPhase] = useState<QuizPhase>('loading');
    const [rawQuestions, setRawQuestions] = useState<RawBackendQuestion[]>([]);
    const [mode, setMode] = useState<'choice' | 'fill' | 'mix'>('choice');
    const [limit, setLimit] = useState<number>(0);

    // Стан для проходження
    const [questions, setQuestions] = useState<GeneratedQuestion[]>([]);
    const [currentQ, setCurrentQ] = useState(0);
    const [score, setScore] = useState(0);
    const [mistakeIds, setMistakeIds] = useState<number[]>([]); // Збираємо ID карток з помилками

    const [isFinished, setIsFinished] = useState(false);
    const [isSaving, setIsSaving] = useState(false);
    const [error, setError] = useState<string | null>(null);
    const [isShuffling, setIsShuffling] = useState(false);
    const [fillAnswer, setFillAnswer] = useState('');

    // ВИПРАВЛЕННЯ: Вираховуємо максимальну кількість питань для поточного режиму
    const maxQuestions = rawQuestions.length * (mode === 'choice' ? 2 : 1);

    // ВИПРАВЛЕННЯ: Скидаємо ліміт до "Всі слова", якщо при зміні режиму поточний ліміт перевищує максимум
    useEffect(() => {
        if (limit > 0 && limit > maxQuestions) {
            setLimit(0);
        }
    }, [mode, maxQuestions, limit]);

    // Завантажуємо "сирі" слова з бази
    useEffect(() => {
        const fetchQuiz = async () => {
            try {
                if (!targetModuleId || Number.isNaN(Number(targetModuleId))) {
                    setError('Невірний ідентифікатор модуля.');
                    setPhase('setup');
                    return;
                }

                // Використовуємо правильні роути бекенду
                const url = isMistakesMode
                    ? `/analytics/mistakes-quiz?module_id=${targetModuleId}`
                    : `/content/quiz/generate?module_id=${targetModuleId}`;

                const data = await apiFetch(url);

                if (!data || data.length === 0) {
                    setError('Не вдалося знайти питання для цього тесту.');
                    setPhase('setup');
                    return;
                }

                setRawQuestions(data);
                setPhase('setup');
            } catch (err: any) {
                const errorMessage = err.message === 'Помилка запиту'
                    ? 'У цьому модулі замало карток для створення тесту. Будь ласка, додайте ще кілька слів (бажано мінімум 4).'
                    : err.message;

                setError(errorMessage);
                setPhase('error');
            }
        };

        fetchQuiz();
    }, [targetModuleId, isMistakesMode]);

    // Відправка результатів на бекенд
    useEffect(() => {
        const submitResults = async () => {
            if (isFinished && questions.length > 0) {
                setIsSaving(true);
                try {
                    // Відправляємо дані у форматі SubmitTestResultRequest
                    await apiFetch('/content/quiz/submit', {
                        method: 'POST',
                        body: JSON.stringify({
                            module_id: Number(targetModuleId),
                            score: score,
                            total_questions: questions.length,
                            mistake_flashcard_ids: mistakeIds
                        })
                    });
                } catch (err) {
                    console.error("Помилка відправки результатів:", err);
                } finally {
                    setIsSaving(false);
                }
            }
        };
        submitResults();
    }, [isFinished, questions.length, score, mistakeIds, targetModuleId]);

    // Генерація тесту на основі обраних налаштувань
    const handleStart = () => {
        let expanded: GeneratedQuestion[] = [];

        rawQuestions.forEach((q) => {
            // Пряме запитання (те, що згенерував бекенд)
            const normalQ: GeneratedQuestion = {
                flashcard_id: q.flashcard_id,
                question_text: q.question,
                correct: q.answer,
                options: q.options,
                type: 'choice'
            };

            // Зворотне запитання (Укр -> Англ)
            const reverseOptions = rawQuestions
                .map(item => item.question)
                .filter((opt) => opt !== q.question)
                .sort(() => Math.random() - 0.5)
                .slice(0, 3);

            const optionsForReverse = [q.question, ...reverseOptions].sort(() => Math.random() - 0.5);

            const reverseQ: GeneratedQuestion = {
                flashcard_id: q.flashcard_id,
                question_text: q.answer,
                correct: q.question,
                options: optionsForReverse,
                type: 'choice'
            };

            // Питання на вписування
            const fillQ: GeneratedQuestion = {
                flashcard_id: q.flashcard_id,
                question_text: q.answer,
                correct: q.question,
                options: [],
                type: 'fill'
            };

            if (mode === 'choice') {
                expanded.push(normalQ);
                expanded.push(reverseQ);
            } else if (mode === 'fill') {
                expanded.push(fillQ);
            } else if (mode === 'mix') {
                const rand = Math.random();
                if (rand < 0.33) {
                    expanded.push(normalQ);
                } else if (rand < 0.66) {
                    expanded.push(reverseQ);
                } else {
                    expanded.push(fillQ);
                }
            }
        });

        expanded.sort(() => Math.random() - 0.5); // Перемішуємо

        if (limit > 0 && expanded.length > limit) {
            expanded = expanded.slice(0, limit); // Обрізаємо за лімітом
        }

        setQuestions(expanded);
        setPhase('testing');
    };

    const handleAnswer = (selectedOption: string) => {
        const question = questions[currentQ];
        const isCorrect = question.type === 'fill'
            ? selectedOption.trim().toLowerCase() === question.correct.toLowerCase()
            : selectedOption === question.correct;

        if (isCorrect) {
            setScore((prev) => prev + 1);
        } else {
            // Зберігаємо ID картки, в якій зроблено помилку
            setMistakeIds(prev => {
                if (!prev.includes(question.flashcard_id)) {
                    return [...prev, question.flashcard_id];
                }
                return prev;
            });
        }

        setFillAnswer('');

        if (currentQ < questions.length - 1) {
            setCurrentQ((prev) => prev + 1);
        } else {
            setIsFinished(true);
        }
    };

    const handleFillSubmit = () => {
        if (!fillAnswer.trim()) return;
        handleAnswer(fillAnswer);
    };

    const handleShuffle = () => {
        if (currentQ >= questions.length - 1) return;

        setIsShuffling(true);
        const pastQuestions = questions.slice(0, currentQ);
        const remainingQuestions = questions.slice(currentQ);
        const shuffledRemaining = [...remainingQuestions].sort(() => Math.random() - 0.5);

        setQuestions([...pastQuestions, ...shuffledRemaining]);
        setFillAnswer('');
        setTimeout(() => setIsShuffling(false), 150);
    };

    // --- РЕНДЕРИНГ ---

    if (phase === 'loading') {
        return <div className="flex justify-center mt-20"><Loader2 className="w-8 h-8 animate-spin text-primary" /></div>;
    }

    if (error && phase !== 'setup') {
        return (
            <div className="max-w-md mx-auto mt-20 bg-red-500/10 border border-red-500/20 rounded-3xl p-8 text-red-400 text-center">
                {error}
                <button onClick={() => navigate(isMistakesMode ? '/mistakes' : '/')} className="block w-full mt-4 bg-primary text-white py-2 rounded-xl">Назад</button>
            </div>
        );
    }

    if (phase === 'setup' && !error) {
        return (
            <div className="max-w-2xl mx-auto mt-20 p-6 bg-surface border border-surfaceBorder rounded-3xl shadow-sm">
                <div className="flex items-center gap-3 mb-8">
                    <Settings className="w-8 h-8 text-primary" />
                    <h2 className="text-2xl font-bold text-textMain">Налаштування тесту</h2>
                </div>

                <div className="space-y-8">
                    <div>
                        <label className="block text-sm font-medium text-textMuted mb-3">Оберіть режим тренування</label>
                        <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
                            <button
                                onClick={() => setMode('choice')}
                                className={`p-4 rounded-xl border text-left transition-all ${mode === 'choice' ? 'border-primary bg-primary/10' : 'border-surfaceBorder hover:border-textMuted'}`}
                            >
                                <List className={`w-6 h-6 mb-2 ${mode === 'choice' ? 'text-primary' : 'text-textMuted'}`} />
                                <h3 className="font-bold text-textMain mb-1">Вибір</h3>
                                <p className="text-xs text-textMuted">Слово ↔ переклад (тести)</p>
                            </button>

                            <button
                                onClick={() => setMode('fill')}
                                className={`p-4 rounded-xl border text-left transition-all ${mode === 'fill' ? 'border-primary bg-primary/10' : 'border-surfaceBorder hover:border-textMuted'}`}
                            >
                                <Type className={`w-6 h-6 mb-2 ${mode === 'fill' ? 'text-primary' : 'text-textMuted'}`} />
                                <h3 className="font-bold text-textMain mb-1">Вписування</h3>
                                <p className="text-xs text-textMuted">Укр → Англ (письмо)</p>
                            </button>

                            <button
                                onClick={() => setMode('mix')}
                                className={`p-4 rounded-xl border text-left transition-all ${mode === 'mix' ? 'border-primary bg-primary/10' : 'border-surfaceBorder hover:border-textMuted'}`}
                            >
                                <ShuffleIcon className={`w-6 h-6 mb-2 ${mode === 'mix' ? 'text-primary' : 'text-textMuted'}`} />
                                <h3 className="font-bold text-textMain mb-1">Мікс</h3>
                                <p className="text-xs text-textMuted">Випадкові завдання</p>
                            </button>
                        </div>
                    </div>

                    <div>
                        <label className="block text-sm font-medium text-textMuted mb-2">Кількість питань</label>
                        <select
                            value={limit}
                            onChange={(e) => setLimit(Number(e.target.value))}
                            className="w-full p-3 bg-mainBg border border-surfaceBorder rounded-xl outline-none text-textMain focus:border-primary transition-colors"
                        >
                            <option value={0}>Всі слова ({maxQuestions} питань)</option>
                            {/* ВИПРАВЛЕННЯ: Показуємо опції тільки якщо карток вистачає */}
                            {maxQuestions >= 10 && <option value={10}>10 питань</option>}
                            {maxQuestions >= 20 && <option value={20}>20 питань</option>}
                            {maxQuestions >= 30 && <option value={30}>30 питань</option>}
                        </select>
                    </div>

                    <button
                        onClick={handleStart}
                        className="w-full py-4 bg-primary text-white rounded-xl font-bold text-lg hover:bg-primary/90 transition-colors"
                    >
                        Почати тест
                    </button>
                </div>
            </div>
        );
    }

    if (isSaving) return <div className="flex justify-center mt-20"><Loader2 className="w-8 h-8 animate-spin text-primary" /> Збереження...</div>;

    if (isFinished) {
        return (
            <div className="max-w-md mx-auto mt-20 bg-surface border border-surfaceBorder rounded-3xl p-8 text-center">
                {score === questions.length ? <CheckCircle className="w-16 h-16 text-green-500 mx-auto mb-4" /> : <XCircle className="w-16 h-16 text-yellow-500 mx-auto mb-4" />}
                <h2 className="text-2xl font-bold mb-2">Тест завершено!</h2>
                <p className="text-textMuted mb-6">Ваш результат: {score} з {questions.length}</p>
                <button onClick={() => navigate(isMistakesMode ? '/mistakes' : '/')} className="w-full bg-primary py-3 rounded-xl text-white font-semibold hover:bg-primaryHover transition-colors">
                    {isMistakesMode ? 'Перевірити помилки' : 'На головну'}
                </button>
            </div>
        );
    }

    if (!questions || questions.length === 0 || !questions[currentQ]) return null;

    const question = questions[currentQ];

    return (
        <div className="max-w-xl mx-auto mt-20 p-4">
            <div className="mb-8">
                <div className="flex items-center justify-between gap-3">
                    <span className="text-primary font-semibold text-sm">
                        Питання {currentQ + 1} з {questions.length} {isMistakesMode && '(Виправлення)'}
                    </span>
                    <button
                        onClick={handleShuffle}
                        className="inline-flex items-center gap-2 rounded-xl border border-surfaceBorder bg-surface px-3 py-2 text-sm font-medium text-textMain transition-colors hover:bg-surfaceBorder"
                    >
                        <ShuffleIcon className="w-4 h-4" />
                        Перемішати
                    </button>
                </div>
                <h2 className="text-2xl font-bold mt-2 text-textMain">{question.question_text}</h2>
            </div>

            {isShuffling && <p className="mb-4 text-sm text-textMuted">Перемішую питання...</p>}

            <div className="space-y-4">
                {question.type === 'fill' ? (
                    <div className="flex flex-col gap-4">
                        <input
                            type="text"
                            value={fillAnswer}
                            onChange={(e) => setFillAnswer(e.target.value)}
                            placeholder="Введіть переклад англійською..."
                            className="w-full p-5 bg-mainBg border border-surfaceBorder rounded-2xl outline-none focus:border-primary transition-colors text-textMain text-lg font-medium"
                            onKeyDown={(e) => {
                                if (e.key === 'Enter' && fillAnswer.trim()) {
                                    handleFillSubmit();
                                }
                            }}
                            autoFocus
                        />
                        <button
                            onClick={handleFillSubmit}
                            disabled={!fillAnswer.trim()}
                            className="w-full py-4 bg-primary text-white rounded-xl font-bold text-lg disabled:opacity-50 disabled:cursor-not-allowed hover:bg-primary/90 transition-colors"
                        >
                            Відповісти
                        </button>
                    </div>
                ) : (
                    question.options.map((opt, index) => (
                        <button
                            key={index}
                            onClick={() => handleAnswer(opt)}
                            className="w-full text-left p-5 bg-surface border border-surfaceBorder rounded-2xl hover:border-primary hover:bg-primary/5 transition-all text-textMain text-lg font-medium"
                        >
                            {opt}
                        </button>
                    ))
                )}
            </div>
        </div>
    );
}
