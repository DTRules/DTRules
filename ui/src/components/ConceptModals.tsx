import { useEffect, useCallback } from 'react';
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
  DialogFooter,
} from '@/components/ui/dialog';
import { Button } from '@/components/ui/button';
import { useOnboardingStore, useHasHydrated } from '@/stores/onboardingStore';

interface ConceptStep {
  title: string;
  content: React.ReactNode;
}

const conceptSteps: ConceptStep[] = [
  {
    title: 'What are Decision Tables?',
    content: (
      <div className="space-y-4">
        <p>
          Decision tables are a visual way to represent complex business rules. Instead of
          nested if-else code, you express logic in a readable table format where:
        </p>
        <ul className="list-disc pl-6 space-y-2">
          <li>Each <strong>column</strong> is a complete rule</li>
          <li>Business analysts can review and modify rules without programming</li>
          <li>Rules are separated from application code</li>
        </ul>
        <p className="text-muted-foreground">
          This makes rules easier to understand, test, and maintain.
        </p>
      </div>
    ),
  },
  {
    title: 'Entities & Tables',
    content: (
      <div className="space-y-4">
        <p>A DTRules project has two main parts:</p>
        <div className="grid gap-4 mt-4">
          <div className="p-4 rounded-lg border bg-muted/50">
            <h4 className="font-semibold mb-2">Entities</h4>
            <p className="text-sm text-muted-foreground">
              Define your data vocabulary (client, case, income). Like database tables
              or classes - they describe what information your rules work with.
            </p>
          </div>
          <div className="p-4 rounded-lg border bg-muted/50">
            <h4 className="font-semibold mb-2">Decision Tables</h4>
            <p className="text-sm text-muted-foreground">
              Define your business rules using that vocabulary. Each table encodes
              logic like "who is eligible" or "what benefits apply."
            </p>
          </div>
        </div>
        <p className="text-center text-muted-foreground italic">
          Think: entities = nouns, tables = verbs
        </p>
      </div>
    ),
  },
  {
    title: 'Reading a Decision Table',
    content: (
      <div className="space-y-4">
        <p>In a decision table grid, cells have special meanings:</p>
        <div className="grid grid-cols-2 gap-3 mt-4">
          <div className="p-3 rounded border bg-green-500/10 border-green-500/30">
            <span className="font-mono font-bold text-green-500">Y</span>
            <span className="ml-2 text-sm">Condition must be true</span>
          </div>
          <div className="p-3 rounded border bg-red-500/10 border-red-500/30">
            <span className="font-mono font-bold text-red-500">N</span>
            <span className="ml-2 text-sm">Condition must be false</span>
          </div>
          <div className="p-3 rounded border bg-muted">
            <span className="font-mono font-bold">-</span>
            <span className="ml-2 text-sm">Don't care (any value OK)</span>
          </div>
          <div className="p-3 rounded border bg-blue-500/10 border-blue-500/30">
            <span className="font-mono font-bold text-blue-500">X</span>
            <span className="ml-2 text-sm">Execute this action</span>
          </div>
        </div>
        <p className="text-muted-foreground mt-4">
          Read each column top-to-bottom: <em>"IF these conditions THEN these actions"</em>
        </p>
      </div>
    ),
  },
  {
    title: 'The CHIP Example',
    content: (
      <div className="space-y-4">
        <p>
          The CHIP (Children's Health Insurance Program) project demonstrates a real
          eligibility system:
        </p>
        <ul className="list-disc pl-6 space-y-2">
          <li>
            <strong>Entities:</strong> client (age, citizenship), case, income
          </li>
          <li>
            <strong>Main table:</strong> <code className="bg-muted px-1.5 py-0.5 rounded text-sm">Compute_Eligibility</code> calls
            helper tables for age, income, and citizenship checks
          </li>
          <li>
            <strong>Tests:</strong> Different scenarios verify age limits, income
            thresholds, and eligibility rules
          </li>
        </ul>
        <p className="text-muted-foreground">
          Explore CHIP to see how real-world rules are structured and tested.
        </p>
      </div>
    ),
  },
  {
    title: 'Your Workflow',
    content: (
      <div className="space-y-4">
        <p>Your development workflow with DTRules:</p>
        <div className="space-y-3 mt-4">
          <div className="flex items-center gap-4">
            <span className="flex-shrink-0 w-8 h-8 rounded-full bg-primary text-primary-foreground flex items-center justify-center font-semibold">1</span>
            <div>
              <div className="font-medium">Define Entities</div>
              <div className="text-sm text-muted-foreground">Set up your data structures</div>
            </div>
          </div>
          <div className="flex items-center gap-4">
            <span className="flex-shrink-0 w-8 h-8 rounded-full bg-primary text-primary-foreground flex items-center justify-center font-semibold">2</span>
            <div>
              <div className="font-medium">Create Decision Tables</div>
              <div className="text-sm text-muted-foreground">Build your business rules</div>
            </div>
          </div>
          <div className="flex items-center gap-4">
            <span className="flex-shrink-0 w-8 h-8 rounded-full bg-primary text-primary-foreground flex items-center justify-center font-semibold">3</span>
            <div>
              <div className="font-medium">Test with Sample Data</div>
              <div className="text-sm text-muted-foreground">Validate your logic works correctly</div>
            </div>
          </div>
          <div className="flex items-center gap-4">
            <span className="flex-shrink-0 w-8 h-8 rounded-full bg-primary text-primary-foreground flex items-center justify-center font-semibold">4</span>
            <div>
              <div className="font-medium">Save & Deploy</div>
              <div className="text-sm text-muted-foreground">Export to XML for production</div>
            </div>
          </div>
        </div>
      </div>
    ),
  },
];

export function ConceptModals() {
  const hasHydrated = useHasHydrated();
  const {
    conceptPhaseActive,
    currentConceptStep,
    nextConceptStep,
    prevConceptStep,
    completeConceptPhase,
    stopTutorial,
  } = useOnboardingStore();

  const currentStep = conceptSteps[currentConceptStep];
  const isFirstStep = currentConceptStep === 0;
  const isLastStep = currentConceptStep === conceptSteps.length - 1;

  const handleNext = useCallback(() => {
    if (isLastStep) {
      completeConceptPhase();
    } else {
      nextConceptStep();
    }
  }, [isLastStep, completeConceptPhase, nextConceptStep]);

  const handleBack = useCallback(() => {
    if (!isFirstStep) {
      prevConceptStep();
    }
  }, [isFirstStep, prevConceptStep]);

  const handleSkip = useCallback(() => {
    stopTutorial();
  }, [stopTutorial]);

  // Keyboard navigation for concept modals
  useEffect(() => {
    if (!conceptPhaseActive) return;

    const handleKeyDown = (e: KeyboardEvent) => {
      switch (e.key) {
        case 'Enter':
        case ' ':
        case 'ArrowRight':
        case 'ArrowDown':
          e.preventDefault();
          handleNext();
          break;
        case 'ArrowLeft':
        case 'ArrowUp':
          e.preventDefault();
          handleBack();
          break;
        case 'Escape':
          e.preventDefault();
          handleSkip();
          break;
      }
    };

    window.addEventListener('keydown', handleKeyDown);
    return () => window.removeEventListener('keydown', handleKeyDown);
  }, [conceptPhaseActive, handleNext, handleBack, handleSkip]);

  if (!hasHydrated) {
    return null;
  }

  if (!conceptPhaseActive) {
    return null;
  }

  return (
    <Dialog open={conceptPhaseActive} onOpenChange={() => {}}>
      <DialogContent
        className="sm:max-w-[500px]"
        hideCloseButton
        onPointerDownOutside={(e) => e.preventDefault()}
        onEscapeKeyDown={(e) => { e.preventDefault(); handleSkip(); }}
      >
        <DialogHeader>
          <div className="flex items-center justify-between mb-2">
            <span className="text-xs text-muted-foreground">
              Step {currentConceptStep + 1} of {conceptSteps.length}
            </span>
          </div>
          <DialogTitle>{currentStep.title}</DialogTitle>
        </DialogHeader>
        <DialogDescription asChild>
          <div className="py-4">
            {currentStep.content}
          </div>
        </DialogDescription>
        <DialogFooter className="flex-row justify-between sm:justify-between">
          <div className="flex flex-col gap-1">
            <Button variant="ghost" onClick={handleSkip}>
              Skip Tutorial
            </Button>
            <span className="text-[10px] text-muted-foreground/60 ml-1">Press Esc</span>
          </div>
          <div className="flex flex-col items-end gap-1">
            <div className="flex gap-2">
              <Button
                variant="outline"
                onClick={handleBack}
                disabled={isFirstStep}
              >
                Back
              </Button>
              <Button onClick={handleNext}>
                {isLastStep ? 'Start UI Tour' : 'Next'}
              </Button>
            </div>
            <span className="text-[10px] text-muted-foreground/60">Press Enter or Arrow keys</span>
          </div>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
