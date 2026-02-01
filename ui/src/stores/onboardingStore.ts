import { create } from 'zustand';
import { persist } from 'zustand/middleware';

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
        tutorialActive: true,
        conceptPhaseActive: true,
        conceptPhaseComplete: false,
        currentConceptStep: 0,
        uiTourActive: false,
        tutorialStepIndex: 0,
        offerTutorial: false,
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
      name: 'dtrules-onboarding',
    }
  )
);
