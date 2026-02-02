import { useEffect, useState, useCallback } from 'react';
import Joyride, { CallBackProps, STATUS, Step, TooltipRenderProps } from 'react-joyride';
import { useOnboardingStore, useHasHydrated } from '@/stores/onboardingStore';
import { useProjectStore } from '@/stores/projectStore';
import { Button } from '@/components/ui/button';

// Custom tooltip component with modern glassmorphism styling
function CustomTooltip({
  continuous,
  index,
  step,
  backProps,
  primaryProps,
  skipProps,
  tooltipProps,
  size,
}: TooltipRenderProps) {
  const isLastStep = index === size - 1;

  return (
    <div
      {...tooltipProps}
      className="bg-background/90 backdrop-blur-xl text-foreground border border-border/50 rounded-2xl shadow-2xl shadow-black/40 max-w-md overflow-hidden"
    >
      {/* Gradient accent bar at top */}
      <div className="h-1 bg-gradient-to-r from-blue-500 via-purple-500 to-blue-500" />

      {/* Header */}
      <div className="p-5 pb-3">
        <div className="flex items-center justify-between mb-1">
          {step.title && (
            <h3 className="font-semibold text-lg bg-gradient-to-r from-blue-400 to-purple-400 bg-clip-text text-transparent">
              {step.title}
            </h3>
          )}
          <span className="text-xs text-muted-foreground bg-muted/50 px-2 py-0.5 rounded-full">
            {index + 1} / {size}
          </span>
        </div>
      </div>

      {/* Content */}
      <div className="px-5 pb-5 text-sm leading-relaxed text-foreground/90">
        {step.content}
      </div>

      {/* Progress bar */}
      <div className="px-5 pb-3">
        <div className="h-1 bg-muted rounded-full overflow-hidden">
          <div
            className="h-full bg-gradient-to-r from-blue-500 to-purple-500 transition-all duration-300"
            style={{ width: `${((index + 1) / size) * 100}%` }}
          />
        </div>
      </div>

      {/* Footer with keyboard hint */}
      <div className="flex items-center justify-between p-4 pt-3 border-t border-border/30 bg-muted/20">
        <div className="flex flex-col gap-1">
          <Button
            {...skipProps}
            variant="ghost"
            size="sm"
            className="text-muted-foreground hover:text-foreground"
          >
            Skip Tour
          </Button>
          <span className="text-[10px] text-muted-foreground/60 ml-1">Press Esc</span>
        </div>
        <div className="flex flex-col items-end gap-1">
          <div className="flex gap-2">
            {index > 0 && (
              <Button {...backProps} variant="outline" size="sm" className="tutorial-back-btn border-border/50">
                Back
              </Button>
            )}
            {continuous && (
              <Button
                {...primaryProps}
                size="sm"
                className="tutorial-next-btn bg-gradient-to-r from-blue-600 to-purple-600 hover:from-blue-500 hover:to-purple-500 text-white border-0"
              >
                {isLastStep ? 'Finish' : 'Next'}
              </Button>
            )}
          </div>
          <span className="text-[10px] text-muted-foreground/60">Press Enter or Arrow keys</span>
        </div>
      </div>
    </div>
  );
}

// Step definitions with tab requirements
interface TutorialStep extends Step {
  tab?: 'edd' | 'dt' | 'test' | 'tree';
}

/**
 * Waits for an element to be visible with non-zero dimensions.
 * Used to ensure tab content is fully rendered before Joyride advances.
 */
const waitForElement = (selector: string, timeout = 500): Promise<boolean> => {
  return new Promise((resolve) => {
    const start = Date.now();
    const check = () => {
      const el = document.querySelector(selector);
      if (el) {
        const rect = el.getBoundingClientRect();
        if (rect.width > 0 && rect.height > 0) {
          resolve(true);
          return;
        }
      }
      if (Date.now() - start < timeout) {
        requestAnimationFrame(check);
      } else {
        resolve(false);
      }
    };
    check();
  });
};

// Steps that should scroll to the bottom of the page (0-indexed)
// User sees steps 5, 9, 10, 13, 17, 18 but array is 0-indexed so: 4, 8, 9, 12, 16, 17
const SCROLL_TO_BOTTOM_STEPS = [4, 8, 9, 12, 16, 17];

/**
 * Scrolls to the bottom of scrollable containers.
 */
const scrollToBottom = () => {
  // Try scrolling the main content area first
  const mainContent = document.querySelector('main') as HTMLElement;
  if (mainContent && mainContent.scrollHeight > mainContent.clientHeight) {
    mainContent.scrollTop = mainContent.scrollHeight;
  }

  // Also scroll any scrollable parent containers
  const scrollableContainers = document.querySelectorAll('[class*="overflow-auto"], [class*="overflow-y-auto"], [class*="overflow-scroll"]');
  scrollableContainers.forEach((container) => {
    const el = container as HTMLElement;
    if (el.scrollHeight > el.clientHeight) {
      el.scrollTop = el.scrollHeight;
    }
  });

  // Also scroll the window/document
  window.scrollTo({
    top: document.documentElement.scrollHeight,
    behavior: 'instant',
  });
};

/**
 * Scrolls to ensure the tooltip is visible after it renders.
 * For specific steps, scrolls to the bottom of the page instead.
 */
const scrollTooltipIntoView = (stepIndex: number) => {
  // For specific steps, scroll to bottom of page
  if (SCROLL_TO_BOTTOM_STEPS.includes(stepIndex)) {
    scrollToBottom();
    return;
  }

  // Find the tooltip element
  const tooltip = document.querySelector('.react-joyride__tooltip') as HTMLElement;
  if (tooltip) {
    const rect = tooltip.getBoundingClientRect();
    const viewportHeight = window.innerHeight;

    // If tooltip is below the viewport, scroll it into view
    if (rect.bottom > viewportHeight) {
      const scrollAmount = rect.bottom - viewportHeight + 50; // 50px padding
      window.scrollBy({
        top: scrollAmount,
        behavior: 'instant',
      });
    }
    // If tooltip is above the viewport, scroll up
    else if (rect.top < 0) {
      window.scrollBy({
        top: rect.top - 50, // 50px padding from top
        behavior: 'instant',
      });
    }
  }
};

// Comprehensive tutorial steps - 20 steps exploring all tabs
const tutorialSteps: TutorialStep[] = [
  // === INTRODUCTION (2 steps) ===
  {
    target: '[data-tutorial="toolbar"]',
    content: 'Welcome to DTRules! This is the main toolbar where you can open projects, save your work, and access help. Let\'s take a tour of the interface.',
    placement: 'bottom',
    disableBeacon: true,
    title: 'Welcome to DTRules',
  },
  {
    target: '[data-tutorial="project-explorer"]',
    content: 'The Project Explorer shows your project structure. Entities define your data model, and Decision Tables contain your business rules. Click any item to edit it.',
    placement: 'right',
    disableBeacon: true,
    title: 'Project Explorer',
  },

  // === ENTITY EDITOR TAB (4 steps) ===
  {
    target: '[data-tutorial="tab-edd"]',
    content: 'Let\'s start with the Entity Editor. This tab defines your data structures - the "vocabulary" your rules will use.',
    placement: 'bottom',
    disableBeacon: true,
    title: 'Entity Editor Tab',
    // NOTE: Don't switch tabs here - tab button is always visible. Next step will switch.
  },
  {
    target: '[data-tutorial="entity-list"]',
    content: 'This panel lists all entities in your project. In CHIP, you\'ll see "client" (the applicant), "case" (the application), and "income" (financial data). Click an entity to view its details.',
    placement: 'right',
    disableBeacon: true,
    title: 'Entity List',
    tab: 'edd',
  },
  {
    target: '[data-tutorial="entity-fields"]',
    content: 'When you select an entity, its attributes appear here. Each attribute has a name, type (String, Number, Boolean, etc.), and optional default value. These become the variables you use in decision tables.',
    placement: 'left',
    disableBeacon: true,
    title: 'Entity Attributes',
    tab: 'edd',
  },
  {
    target: '[data-tutorial="entity-editor"]',
    content: 'For example, the "client" entity has attributes like "age" (Number), "citizenship" (String), and "has_insurance" (Boolean). Think of entities as nouns in your business language.',
    placement: 'bottom',
    disableBeacon: true,
    title: 'Understanding Entities',
    tab: 'edd',
  },

  // === DECISION TABLES TAB (5 steps) ===
  {
    target: '[data-tutorial="tab-dt"]',
    content: 'Now let\'s explore Decision Tables - this is where you create your business rules. Each table is like a spreadsheet that\'s easy to read and maintain.',
    placement: 'bottom',
    disableBeacon: true,
    title: 'Decision Tables Tab',
    // NOTE: Don't switch tabs here - tab button is always visible. Next step will switch.
  },
  {
    target: '[data-tutorial="dt-list"]',
    content: 'This panel lists all decision tables. Tables are organized by purpose - "Compute_Eligibility" is the main entry point, while helper tables like "Check_Age" handle specific checks.',
    placement: 'right',
    disableBeacon: true,
    title: 'Table List',
    tab: 'dt',
  },
  {
    target: '[data-tutorial="dt-grid"]',
    content: 'The decision table grid shows CONDITIONS at the top and ACTIONS below. Each COLUMN is a complete rule. Read top-to-bottom: "IF these conditions THEN these actions".',
    placement: 'left',
    disableBeacon: true,
    title: 'Decision Table Grid',
    tab: 'dt',
  },
  {
    target: '[data-tutorial="dt-grid"]',
    content: 'In the grid: "Y" = condition must be true, "N" = must be false, "-" = don\'t care. For actions: "X" = execute this action, "-" = skip it.',
    placement: 'right',
    disableBeacon: true,
    title: 'Reading the Grid',
    tab: 'dt',
  },
  {
    target: '[data-tutorial="dt-list"]',
    content: 'Click any table name to view and edit it. Try selecting different tables to see how conditions and actions vary. Each table focuses on one specific decision.',
    placement: 'right',
    disableBeacon: true,
    title: 'Selecting Tables',
    tab: 'dt',
  },

  // === TEST & EXECUTE TAB (4 steps) ===
  {
    target: '[data-tutorial="tab-test"]',
    content: 'The Test & Execute tab lets you validate your rules. Click this tab to provide sample data, run your tables, and verify results.',
    placement: 'bottom',
    disableBeacon: true,
    title: 'Test & Execute Tab',
    // NOTE: Don't switch tabs here - tab button is always visible. Next step will switch.
  },
  {
    target: '[data-tutorial="test-input"]',
    content: 'Enter your test data here in JSON format. The structure mirrors your entities - provide a client object with age, citizenship, etc., plus case and income data.',
    placement: 'right',
    disableBeacon: true,
    title: 'Test Input',
    tab: 'test',
  },
  {
    target: '[data-tutorial="test-results"]',
    content: 'After execution, results appear here. You\'ll see which rules fired and what values were set. Check if "eligible" is true or false based on your test data.',
    placement: 'left',
    disableBeacon: true,
    title: 'Test Results',
    tab: 'test',
  },
  {
    target: '[data-tutorial="test-panel"]',
    content: 'Enable tracing to see exactly which rules fired during execution. This is essential for debugging - you can trace step-by-step how a decision was made.',
    placement: 'bottom',
    disableBeacon: true,
    title: 'Execution Tracing',
    tab: 'test',
  },

  // === TREE VIEW TAB (3 steps) ===
  {
    target: '[data-tutorial="tab-tree"]',
    content: 'The Tree View visualizes how your decision tables connect. Click this tab to see which tables call which.',
    placement: 'bottom',
    disableBeacon: true,
    title: 'Tree View Tab',
    // NOTE: Don't switch tabs here - tab button is always visible. Next step will switch.
  },
  {
    target: '[data-tutorial="tree-visualization"]',
    content: 'This diagram shows the call hierarchy. For CHIP: "Compute_Eligibility" at the top calls age, income, and citizenship checking tables. Click nodes to navigate to that table.',
    placement: 'left-start',
    disableBeacon: true,
    disableOverlay: true,
    title: 'Rule Flow Diagram',
    tab: 'tree',
  },
  {
    target: '[data-tutorial="tree-visualization"]',
    content: 'Understanding this flow helps you debug complex rule chains and see how changes to one table might affect others in the hierarchy.',
    placement: 'left-start',
    disableBeacon: true,
    disableOverlay: true,
    title: 'Debugging with Tree View',
    tab: 'tree',
  },

  // === WRAP UP (2 steps) ===
  {
    target: '[data-tutorial="toolbar-save"]',
    content: 'Don\'t forget to save! Your entities and decision tables are stored in XML files that can be version-controlled with Git and deployed to production.',
    placement: 'bottom',
    disableBeacon: true,
    title: 'Saving Your Work',
  },
  {
    target: '[data-tutorial="help-menu"]',
    content: 'You now understand the fundamentals! Explore the CHIP project, try modifying rules and testing them. Use Help > Start Tutorial to revisit this guide anytime. Happy rule building!',
    placement: 'bottom',
    disableBeacon: true,
    title: 'Tutorial Complete!',
  },
];

export function GuidedTutorial() {
  const hasHydrated = useHasHydrated();
  const {
    uiTourActive,
    conceptPhaseActive,
    stopTutorial,
    completeTutorial,
    setTutorialStepIndex,
  } = useOnboardingStore();
  const { projectPath, setActiveTab } = useProjectStore();
  const [stepIndex, setStepIndex] = useState(0);
  const [isTransitioning, setIsTransitioning] = useState(false);

  // Stop tutorial if no project is open
  useEffect(() => {
    if (uiTourActive && !projectPath) {
      stopTutorial();
    }
  }, [uiTourActive, projectPath, stopTutorial]);

  // Reset state when tutorial starts
  useEffect(() => {
    if (uiTourActive) {
      setStepIndex(0);
    }
  }, [uiTourActive]);

  // Global keyboard handler for tutorial navigation
  useEffect(() => {
    if (!uiTourActive || conceptPhaseActive || isTransitioning) return;

    const handleKeyDown = (e: KeyboardEvent) => {
      // Don't handle if user is typing in an input
      if (e.target instanceof HTMLInputElement || e.target instanceof HTMLTextAreaElement) {
        return;
      }

      switch (e.key) {
        case 'Enter':
        case ' ':
        case 'ArrowRight':
        case 'ArrowDown':
          e.preventDefault();
          // Simulate clicking the Next/Finish button
          const nextBtn = document.querySelector('.tutorial-next-btn') as HTMLButtonElement;
          if (nextBtn) {
            nextBtn.click();
          }
          break;
        case 'ArrowLeft':
        case 'ArrowUp':
          e.preventDefault();
          // Simulate clicking the Back button
          const backBtn = document.querySelector('.tutorial-back-btn') as HTMLButtonElement;
          if (backBtn) {
            backBtn.click();
          }
          break;
        case 'Escape':
          e.preventDefault();
          stopTutorial();
          break;
      }
    };

    window.addEventListener('keydown', handleKeyDown);
    return () => window.removeEventListener('keydown', handleKeyDown);
  }, [uiTourActive, conceptPhaseActive, isTransitioning, stopTutorial]);

  const handleJoyrideCallback = useCallback(async (data: CallBackProps) => {
    const { status, index, action, type } = data;

    if (status === STATUS.FINISHED || status === STATUS.SKIPPED) {
      completeTutorial();
      setActiveTab('edd'); // Navigate to first tab after tutorial
      return;
    }

    // Handle step navigation
    if (type === 'step:after') {
      const nextIndex = index + (action === 'prev' ? -1 : 1);

      // Check if we're finishing the last step (clicking Next/Finish on the last step)
      if (action === 'next' && index === tutorialSteps.length - 1) {
        completeTutorial();
        setActiveTab('edd'); // Navigate to first tab after tutorial
        return;
      }

      if (nextIndex >= 0 && nextIndex < tutorialSteps.length) {
        const nextStep = tutorialSteps[nextIndex] as TutorialStep;

        // Hide tooltip during transition to prevent flash
        setIsTransitioning(true);

        // Switch tab if needed
        if (nextStep.tab) {
          setActiveTab(nextStep.tab);
          // Wait for element to be visible after tab switch to prevent black screen
          await waitForElement(nextStep.target as string);
        }

        setStepIndex(nextIndex);
        setTutorialStepIndex(nextIndex);

        // Show tooltip after positioning is complete
        setTimeout(() => {
          setIsTransitioning(false);
          // Scroll tooltip into view after it renders
          setTimeout(() => {
            scrollTooltipIntoView(nextIndex);
          }, 100);
        }, 100);
      }
      return;
    }

    // Handle case where target element is not found - wait and retry before skipping
    if (type === 'error:target_not_found') {
      console.warn('Tutorial target not found for step', index);
      // Wait a frame for element to potentially appear
      requestAnimationFrame(async () => {
        const currentStep = tutorialSteps[index] as TutorialStep;
        const el = document.querySelector(currentStep.target as string);
        if (!el) {
          // Only skip if element is truly missing after waiting
          const nextIndex = index + 1;
          if (nextIndex < tutorialSteps.length) {
            const nextStep = tutorialSteps[nextIndex] as TutorialStep;
            if (nextStep.tab) {
              setActiveTab(nextStep.tab);
              await waitForElement(nextStep.target as string);
            }
            setStepIndex(nextIndex);
            setTutorialStepIndex(nextIndex);
            // Scroll tooltip into view after it renders
            setTimeout(() => {
              scrollTooltipIntoView(nextIndex);
            }, 150);
          }
        }
      });
    }

    if (action === 'close') {
      stopTutorial();
    }
  }, [completeTutorial, setActiveTab, setTutorialStepIndex, stopTutorial]);

  // Don't render until hydration completes
  if (!hasHydrated) {
    return null;
  }

  // Don't render during concept phase or when UI tour is inactive
  if (!uiTourActive || !projectPath || conceptPhaseActive) {
    return null;
  }

  return (
    <Joyride
      steps={tutorialSteps}
      stepIndex={stepIndex}
      run={!isTransitioning}
      continuous
      showSkipButton
      hideCloseButton
      disableOverlayClose={false}
      disableScrolling={true}
      disableScrollParentFix={true}
      scrollToFirstStep={false}
      spotlightPadding={8}
      spotlightClicks
      callback={handleJoyrideCallback}
      tooltipComponent={CustomTooltip}
      floaterProps={{
        disableAnimation: true,
        hideArrow: true,
        offset: 16,
        placement: 'auto',
        styles: {
          floater: {
            transition: 'none !important',
            filter: 'none',
            transitionProperty: 'none',
          },
          arrow: {
            display: 'none',
          },
        },
      }}
      styles={{
        options: {
          zIndex: 10000,
          overlayColor: 'rgba(0, 0, 0, 0.75)',
          spotlightShadow: '0 0 20px rgba(139, 92, 246, 0.8), 0 0 40px rgba(59, 130, 246, 0.6)',
        },
        spotlight: {
          borderRadius: 12,
          transition: 'none',
        },
        overlay: {
          transition: 'none',
          animation: 'none',
        },
        tooltip: {
          transition: 'none',
          animation: 'none',
        },
        tooltipContainer: {
          transition: 'none',
        },
        beacon: {
          display: 'none',
        },
        beaconInner: {
          display: 'none',
        },
        beaconOuter: {
          display: 'none',
        },
      }}
    />
  );
}

// Export the step count for use in navigation
export const UI_TOUR_STEP_COUNT = tutorialSteps.length;
