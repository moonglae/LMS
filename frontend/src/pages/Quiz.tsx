import { useEffect, useState } from 'react';
import { useNavigate, useParams, useSearchParams, useLocation } from 'react-router-dom';
import { CheckCircle, XCircle, Loader2 } from 'lucide-react';
import { apiFetch } from '../api';

interface GeneratedQuestion {
    id: number;
    quiz_id: number;
    question_text: string;
    options: string[];
    correct: string;
}

interface AnswerResult {
    question_id: number;
    is_correct: boolean;
}

export default function Quiz() {
    const { id } = useParams();
    const navigate = useNavigate();
    const [searchParams] = useSearchParams();
    const location = useLocation(); // ДОДАНО: Дозволяє бачити весь шлях

    // Перевіряємо за реальним шляхом, а не за параметром id
    const isMistakesMode = location.pathname.includes('/quiz/mistakes');
    const targetModuleId = isMistakesMode ? searchParams.get('module_id') : id;

    const [questions, setQuestions] = useState<GeneratedQuestion[]>([]);
    const [currentQ, setCurrentQ] = useState(0);
    const [score, setScore] = useState(0);
    const [answers, setAnswers] = useState<AnswerResult[]>([]);
    const [isFinished, setIsFinished] = useState(false);
    const [isLoading, setIsLoading] = useState(true);
    const [isSaving, setIsSaving] = useState(false);
    const [error, setError] = useState<string | null>(null);

    useEffect(() => {
        const fetchQuiz = async () => {
            try {
                let data;
                if (isMistakesMode) {
                    const url = targetModuleId ? `/analytics/mistakes-quiz?module_id=${targetModuleId}` : '/analytics/mistakes-quiz';
                    data = await apiFetch(url);
                } else {
                    if (!targetModuleId || Number.isNaN(Number(targetModuleId))) {
                        setError('Невірний модуль.');
                        setIsLoading(false);
                        return;
                    }
                    data = await apiFetch(`/generate-quiz?module_id=${targetModuleId}`);
                }

                if (!data || data.length === 0) {
                    setError('Не вдалося знайти питання для цього тесту.');
                    setIsLoading(false);
                    return;
                }

                setQuestions(data);
            } catch (err: any) {
                setError(err.message || 'Помилка завантаження.');
            } finally {
                setIsLoading(false);
            }
        };

        fetchQuiz();
    }, [id, targetModuleId, isMistakesMode]);

    useEffect(() => {
        const submitResults = async () => {
            if (isFinished && questions.length > 0) {
                setIsSaving(true);
                try {
                    await apiFetch('/analytics/quiz/submit', {
                        method: 'POST',
                        body: JSON.stringify({
                            quiz_id: questions[0].quiz_id,
                            score: score,
                            total_questions: questions.length,
                            answers: answers
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
    }, [isFinished, questions, score, answers]);

    const handleAnswer = (selectedOption: string) => {
        const question = questions[currentQ];
        const isCorrect = selectedOption === question.correct;

        setAnswers(prev => [...prev, { question_id: question.id, is_correct: isCorrect }]);

        if (isCorrect) setScore((prev) => prev + 1);

        if (currentQ < questions.length - 1) {
            setCurrentQ((prev) => prev + 1);
        } else {
            setIsFinished(true);
        }
    };

    if (isLoading) return <div className="flex justify-center mt-20"><Loader2 className="w-8 h-8 animate-spin text-primary" /></div>;
    if (isSaving) return <div className="flex justify-center mt-20"><Loader2 className="w-8 h-8 animate-spin text-primary" /> Збереження...</div>;

    if (error) return (
        <div className="max-w-md mx-auto mt-20 bg-red-500/10 border border-red-500/20 rounded-3xl p-8 text-red-400 text-center">
            {error}
            <button onClick={() => navigate(isMistakesMode ? '/mistakes' : '/')} className="block w-full mt-4 bg-primary text-white py-2 rounded-xl">Назад</button>
        </div>
    );

    if (isFinished) {
        return (
            <div className="max-w-md mx-auto mt-20 bg-surface border border-surfaceBorder rounded-3xl p-8 text-center">
                {score === questions.length ? <CheckCircle className="w-16 h-16 text-green-500 mx-auto mb-4" /> : <XCircle className="w-16 h-16 text-yellow-500 mx-auto mb-4" />}
                <h2 className="text-2xl font-bold mb-2">Тест завершено!</h2>
                <p className="text-textMuted mb-6">Ваш результат: {score} з {questions.length}</p>
                <button onClick={() => navigate(isMistakesMode ? '/mistakes' : '/')} className="w-full bg-primary py-3 rounded-xl text-white font-semibold">
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
                <span className="text-primary font-semibold text-sm">
                    Питання {currentQ + 1} з {questions.length} {isMistakesMode && '(Виправлення)'}
                </span>
                <h2 className="text-2xl font-bold mt-2 text-textMain">{question.question_text}</h2>
            </div>

            <div className="space-y-4">
                {question.options.map((opt, index) => (
                    <button
                        key={index}
                        onClick={() => handleAnswer(opt)}
                        className="w-full text-left p-5 bg-surface border border-surfaceBorder rounded-2xl hover:border-primary hover:bg-primary/5 transition-all text-textMain text-lg font-medium"
                    >
                        {opt}
                    </button>
                ))}
            </div>
        </div>
    );
}