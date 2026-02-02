/**
 * Onboarding Store - Manages tutorial and welcome screen state.
 *
 * This store handles the two-phase tutorial system:
 * - Phase 1: Concept Modals - 5 modal dialogs teaching DTRules concepts
 * - Phase 2: UI Tour - 20-step interactive Joyride tour
 *
 * State is persisted to localStorage to remember tutorial completion.
 *
 * @module stores/onboardingStore
 *
 * @example
 * // Start the tutorial
 * const { startTutorial } = useOnboardingStore();
 * startTutorial();
 *
 * // Check if tutorial is complete
 * const { tutorialCompleted } = useOnboardingStore();
 */

import { create } from 'zustand';
import { persist } from 'zustand/middleware';
import { useEffect, useState } from 'react';

/**
 * Complete state interface for the onboarding store.
 */
interface OnboardingState {
  // State
  showWelcome: boolean;
  tutorialActive: boolean;
  tutorialStepIndex: number;
  tutorialCompleted: boolean;
  offerTutorial: boolean;
  dontAskAgain: boolean;

  // New two-phase tutorial state
  conceptPhaseActive: boolean;
  conceptPhaseComplete: boolean;
  currentConceptStep: number;  // 0-4
  uiTourActive: boolean;

  // Actions
  setShowWelcome: (show: boolean) => void;
  startTutorial: () => void;
  stopTutorial: () => void;
  completeTutorial: () => void;
  resetTutorial: () => void;
  setTutorialStepIndex: (index: number) => void;
  setOfferTutorial: (offer: boolean) => void;
  setDontAskAgain: (dontAsk: boolean) => void;
  showWelcomeScreen: () => void;

  // New two-phase tutorial actions
  nextConceptStep: () => void;
  prevConceptStep: () => void;
  completeConceptPhase: () => void;
  startUITour: () => void;
  stopUITour: () => void;
}

export const useOnboardingStore = create<OnboardingState>()(
  persist(
    (set) => ({
      // Initial state
      showWelcome: true,
      tutorialActive: false,
      tutorialStepIndex: 0,
      tutorialCompleted: false,
      offerTutorial: false,
      dontAskAgain: false,

      // New two-phase tutorial state
      conceptPhaseActive: false,
      conceptPhaseComplete: false,
      currentConceptStep: 0,
      uiTourActive: false,

      // Actions
      setShowWelcome: (show) => set({ showWelcome: show }),

      startTutorial: () => set({
        // Explicitly reset ALL state fields to prevent stale localStorage data from interfering
        showWelcome: false,
        tutorialActive: true,
        tutorialStepIndex: 0,
        tutorialCompleted: false,
        offerTutorial: false,
        dontAskAgain: false,
        conceptPhaseActive: true,
        conceptPhaseComplete: false,
        currentConceptStep: 0,
        uiTourActive: false,
      }),

      stopTutorial: () => set({
        tutorialActive: false,
        conceptPhaseActive: false,
        uiTourActive: false,
        tutorialStepIndex: 0,
        currentConceptStep: 0,
      }),

      completeTutorial: () => set({
        tutorialActive: false,
        conceptPhaseActive: false,
        uiTourActive: false,
        tutorialStepIndex: 0,
        currentConceptStep: 0,
        tutorialCompleted: true,
      }),

      resetTutorial: () => set({
        tutorialActive: false,
        conceptPhaseActive: false,
        conceptPhaseComplete: false,
        uiTourActive: false,
        tutorialStepIndex: 0,
        currentConceptStep: 0,
        tutorialCompleted: false,
        dontAskAgain: false,
      }),

      setTutorialStepIndex: (index) => set({ tutorialStepIndex: index }),

      setOfferTutorial: (offer) => set({ offerTutorial: offer }),

      setDontAskAgain: (dontAsk) => set({ dontAskAgain: dontAsk }),

      showWelcomeScreen: () => set({
        showWelcome: true,
        tutorialActive: false,
        conceptPhaseActive: false,
        uiTourActive: false,
      }),

      // New two-phase tutorial actions
      nextConceptStep: () => set((state) => ({
        currentConceptStep: Math.min(state.currentConceptStep + 1, 4),
      })),

      prevConceptStep: () => set((state) => ({
        currentConceptStep: Math.max(state.currentConceptStep - 1, 0),
      })),

      completeConceptPhase: () => set({
        conceptPhaseActive: false,
        conceptPhaseComplete: true,
        uiTourActive: true,
        tutorialStepIndex: 0,
      }),

      startUITour: () => set({
        uiTourActive: true,
        tutorialStepIndex: 0,
      }),

      stopUITour: () => set({
        uiTourActive: false,
        tutorialStepIndex: 0,
      }),
    }),
    {
      // Changed key name to force fresh start and clear stale data from old key
      name: 'dtrules-onboarding-v2',

      // Only persist user preferences, NOT transient tutorial state
      // This fixes the race condition where stale tutorial state from localStorage
      // would overwrite state set by startTutorial()
      partialize: (state) => ({
        tutorialCompleted: state.tutorialCompleted,
        dontAskAgain: state.dontAskAgain,
      }),
    }
  )
);

/**
 * Hook that returns true only after the store has finished hydrating from localStorage.
 * Use this to prevent rendering overlay components during the hydration race window.
 */
export const useHasHydrated = () => {
  // Initialize directly from the store's hydration status to avoid first-render delay
  const [hasHydrated, setHasHydrated] = useState(() =>
    useOnboardingStore.persist.hasHydrated()
  );

  useEffect(() => {
    // Subscribe to hydration completion in case it hasn't finished yet
    const unsubFinishHydration = useOnboardingStore.persist.onFinishHydration(() => {
      setHasHydrated(true);
    });

    return () => {
      unsubFinishHydration();
    };
  }, []);

  return hasHydrated;
};
