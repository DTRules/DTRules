import { Button } from '@/components/ui/button';
import { useOnboardingStore, useHasHydrated } from '@/stores/onboardingStore';
import { UI_TOUR_STEP_COUNT } from '@/components/GuidedTutorial';
import { X } from 'lucide-react';

const CONCEPT_STEPS = 5;

export function TutorialNavigation() {
  const hasHydrated = useHasHydrated();
  const {
    tutorialActive,
    conceptPhaseActive,
    uiTourActive,
    tutorialStepIndex,
    stopTutorial,
  } = useOnboardingStore();

  if (!hasHydrated) {
    return null;
  }

  // Don't show if tutorial is not active
  if (!tutorialActive) {
    return null;
  }

  // During concept phase, the ConceptModals handle their own navigation
  // This bar only shows during the UI tour phase
  if (conceptPhaseActive) {
    return null;
  }

  if (!uiTourActive) {
    return null;
  }

  const currentStep = tutorialStepIndex + 1;
  const totalSteps = UI_TOUR_STEP_COUNT;

  return (
    <div className="fixed bottom-0 left-0 right-0 z-[9999] bg-background/95 backdrop-blur border-t border-border">
      <div className="max-w-2xl mx-auto px-4 py-3 flex items-center justify-between">
        {/* Progress indicator */}
        <div className="flex items-center gap-3">
          <span className="text-sm font-medium">UI Tour</span>
          <div className="flex gap-1">
            {Array.from({ length: totalSteps }).map((_, i) => (
              <div
                key={i}
                className={`h-2 w-2 rounded-full transition-colors ${
                  i < currentStep
                    ? 'bg-primary'
                    : i === currentStep - 1
                    ? 'bg-primary'
                    : 'bg-muted'
                }`}
              />
            ))}
          </div>
          <span className="text-sm text-muted-foreground">
            Step {currentStep} of {totalSteps}
          </span>
        </div>

        {/* Navigation hint */}
        <div className="flex items-center gap-2">
          <span className="text-sm text-muted-foreground">
            Use the tooltip buttons to navigate
          </span>
          <Button
            variant="ghost"
            size="sm"
            onClick={stopTutorial}
            className="text-muted-foreground hover:text-foreground"
          >
            <X className="h-4 w-4 mr-1" />
            Exit Tour
          </Button>
        </div>
      </div>
    </div>
  );
}

// Separate component for concept phase navigation (shown outside the modal)
export function ConceptPhaseProgress() {
  const hasHydrated = useHasHydrated();
  const {
    conceptPhaseActive,
    currentConceptStep,
  } = useOnboardingStore();

  if (!hasHydrated) {
    return null;
  }

  if (!conceptPhaseActive) {
    return null;
  }

  return (
    <div className="fixed bottom-0 left-0 right-0 z-[9998] bg-background/95 backdrop-blur border-t border-border">
      <div className="max-w-2xl mx-auto px-4 py-3 flex items-center justify-center">
        <div className="flex items-center gap-3">
          <span className="text-sm font-medium">Learning Concepts</span>
          <div className="flex gap-1">
            {Array.from({ length: CONCEPT_STEPS }).map((_, i) => (
              <div
                key={i}
                className={`h-2 w-2 rounded-full transition-colors ${
                  i <= currentConceptStep ? 'bg-primary' : 'bg-muted'
                }`}
              />
            ))}
          </div>
          <span className="text-sm text-muted-foreground">
            Step {currentConceptStep + 1} of {CONCEPT_STEPS}
          </span>
        </div>
      </div>
    </div>
  );
}
