// src/pages/Flashcards.tsx
import React, { useEffect, useState } from 'react';
import { useParams, useNavigate } from 'react-router-dom';
import { Volume2, ArrowLeft, ArrowRight, RotateCcw, Loader2, Shuffle } from 'lucide-react';
import { apiFetch } from '../api';
import type { Flashcard } from '../types';

export default function Flashcards() {
    const { id } = useParams<{ id?: string }>();
    const navigate = useNavigate();
    const moduleId = Number(id);

    const [cards, setCards] = useState<Flashcard[]>([]);
    const [currentIndex, setCurrentIndex] = useState<number>(0);
    const [isFlipped, setIsFlipped] = useState<boolean>(false);
    const [isLoading, setIsLoading] = useState<boolean>(true);
    const [error, setError] = useState<string | null>(null);

    useEffect(() => {
        const fetchCards = async () => {
            try {
                const data = await apiFetch(`/modules/flashcards?module_id=${moduleId}`);
                const arr: Flashcard[] = Array.isArray(data) ? data : [];
                setCards(arr);
                setCurrentIndex(0);
            } catch (err: any) {
                console.error(err);
                setError(err?.message || 'Не вдалося завантажити картки.');
            } finally {
                setIsLoading(false);
            }
        };

        if (moduleId && !Number.isNaN(moduleId)) {
            fetchCards();
        } else {
            setError('Невідомий модуль для карток.');
            setIsLoading(false);
        }
    }, [moduleId]);

    useEffect(() => {
        if (cards.length === 0) {
            setCurrentIndex(0);
            return;
        }
        setCurrentIndex((idx) => Math.max(0, Math.min(idx, cards.length - 1)));
    }, [cards]);

    // РОЗУМНЕ ОЗВУЧЕННЯ
    const speakText = (text: string) => {
        if (typeof window === 'undefined' || !('speechSynthesis' in window)) {
            setError('Ваш браузер не підтримує озвучення тексту.');
            return;
        }

        const synth = window.speechSynthesis;
        synth.cancel(); // Зупиняємо попереднє озвучення, якщо воно ще грає

        const utterance = new SpeechSynthesisUtterance(text);

        // Перевіряємо, чи є в тексті кирилиця (українські літери)
        const isUkrainian = /[а-яА-ЯіІїЇєЄґҐ]/.test(text);

        // Якщо є українські літери - ставимо uk-UA, інакше en-US
        utterance.lang = isUkrainian ? 'uk-UA' : 'en-US';
        utterance.rate = 0.9; // Трохи сповільнюємо для кращого сприйняття

        synth.speak(utterance);
    };

    const shuffleCards = () => {
        setCards((prev) => {
            const shuffled = [...prev].sort(() => Math.random() - 0.5);
            return shuffled;
        });
        setCurrentIndex(0);
        setIsFlipped(false);
    };

    const nextCard = () => {
        if (!cards.length) return;
        setIsFlipped(false);
        setCurrentIndex((prev: number) => {
            const next = prev + 1;
            return next >= cards.length ? 0 : next;
        });
    };

    const prevCard = () => {
        if (!cards.length) return;
        setIsFlipped(false);
        setCurrentIndex((prev: number) => {
            const next = prev - 1;
            return next < 0 ? cards.length - 1 : next;
        });
    };

    if (isLoading) {
        return (
            <div className="flex justify-center mt-20">
                <Loader2 className="w-8 h-8 animate-spin text-primary" />
            </div>
        );
    }

    if (error) {
        return (
            <div className="max-w-2xl mx-auto mt-20 bg-red-500/10 border border-red-500/20 rounded-3xl p-8 text-red-400">
                {error}
            </div>
        );
    }

    if (!cards.length) {
        return (
            <div className="max-w-2xl mx-auto mt-20 bg-surface border border-surfaceBorder rounded-3xl p-10 text-center">
                <h2 className="text-xl font-semibold text-textMain mb-2">Цей модуль ще не має карток</h2>
                <p className="text-textMuted">Поверніться назад або додайте нову картку у редакторі курсу.</p>
                <button
                    type="button"
                    onClick={() => navigate('/teacher')}
                    className="mt-6 bg-primary hover:bg-primaryHover text-white px-6 py-3 rounded-xl font-medium transition-colors"
                >
                    Створити картки
                </button>
            </div>
        );
    }

    const currentCard = cards[currentIndex];
    const cardContainerStyle = { perspective: '1000px' } as React.CSSProperties;
    const cardStyle = {
        width: '100%',
        height: '100%',
        position: 'relative' as const,
        transformStyle: 'preserve-3d' as const,
        transition: 'transform 0.5s',
        transform: isFlipped ? 'rotateY(180deg)' : 'rotateY(0deg)',
    } as React.CSSProperties;
    const faceStyle = {
        position: 'absolute' as const,
        inset: 0,
        backfaceVisibility: 'hidden' as const,
        WebkitBackfaceVisibility: 'hidden' as const,
    } as React.CSSProperties;
    const backFaceStyle = {
        ...faceStyle,
        transform: 'rotateY(180deg)',
    } as React.CSSProperties;

    return (
        <div className="max-w-3xl mx-auto mt-10 space-y-6">
            <div className="flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
                <button
                    type="button"
                    onClick={() => navigate('/')}
                    className="text-textMuted hover:text-textMain flex items-center gap-2"
                >
                    <ArrowLeft className="w-5 h-5" /> Назад до курсів
                </button>
                <div className="flex flex-col sm:flex-row sm:items-center gap-3">
                    <span className="text-textMuted font-medium">Картка {currentIndex + 1} з {cards.length}</span>
                    <button
                        type="button"
                        onClick={(e) => { e.stopPropagation(); shuffleCards(); }}
                        className="inline-flex items-center gap-2 rounded-xl bg-surface border border-surfaceBorder px-4 py-2 text-sm font-medium text-textMain hover:bg-surfaceBorder transition-colors"
                    >
                        <Shuffle className="w-4 h-4" /> Перемішати
                    </button>
                </div>
            </div>

            <div
                className="relative h-96 w-full"
                style={cardContainerStyle}
                onClick={() => setIsFlipped((prev) => !prev)}
            >
                <div style={cardStyle}>
                    {/* ПЕРЕДНЯ СТОРОНА */}
                    <div style={faceStyle} className="bg-surface border border-surfaceBorder rounded-3xl p-8 flex flex-col items-center justify-center shadow-xl">
                        <button
                            type="button"
                            onClick={(e) => { e.stopPropagation(); speakText(currentCard.question); }}
                            className="absolute top-6 right-6 p-3 bg-primary/10 text-primary rounded-full hover:bg-primary/20 transition-colors z-30"
                        >
                            <Volume2 className="w-6 h-6" />
                        </button>
                        <h2 className="text-4xl font-bold text-textMain text-center">{currentCard.question}</h2>
                        <p className="text-textMuted mt-4 flex items-center gap-2"><RotateCcw className="w-4 h-4" /> Натисніть, щоб перевернути</p>
                    </div>

                    {/* ЗАДНЯ СТОРОНА */}
                    <div style={backFaceStyle} className="bg-primary/5 border border-primary/20 rounded-3xl p-8 flex flex-col items-center justify-center shadow-xl">
                        <button
                            type="button"
                            onClick={(e) => { e.stopPropagation(); speakText(currentCard.answer); }}
                            className="absolute top-6 right-6 p-3 bg-primary/10 text-primary rounded-full hover:bg-primary/20 transition-colors z-30"
                        >
                            <Volume2 className="w-6 h-6" />
                        </button>
                        <h2 className="text-3xl font-semibold text-textMain text-center">{currentCard.answer}</h2>
                        <p className="text-textMuted mt-4 flex items-center gap-2"><RotateCcw className="w-4 h-4" /> Це задня сторона картки. Натисніть, щоб повернути.</p>
                    </div>
                </div>
            </div>

            <div className="flex justify-center gap-4 relative z-20">
                <button
                    type="button"
                    onClick={(e) => { e.stopPropagation(); prevCard(); }}
                    className="inline-flex items-center justify-center rounded-2xl border border-surfaceBorder bg-surface px-5 py-3 text-textMain hover:border-primary hover:bg-surfaceBorder transition-colors"
                >
                    <ArrowLeft className="w-5 h-5" /> Попередня
                </button>
                <button
                    type="button"
                    onClick={(e) => { e.stopPropagation(); nextCard(); }}
                    className="inline-flex items-center justify-center rounded-2xl bg-primary px-5 py-3 text-white hover:bg-primaryHover transition-colors"
                >
                    Наступна <ArrowRight className="w-5 h-5" />
                </button>
            </div>
        </div>
    );
}