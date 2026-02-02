import { useEffect, useState, useCallback } from 'react';
import Joyride, { CallBackProps, STATUS, Step, TooltipRenderProps } from 'react-joyride';
import { useOnboardingStore } from '@/stores/onboardingStore';
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

      {/* Footer */}
      <div className="flex items-center justify-between p-4 pt-3 border-t border-border/30 bg-muted/20">
        <Button
          {...skipProps}
          variant="ghost"
          size="sm"
          className="text-muted-foreground hover:text-foreground"
        >
          Skip Tour
        </Button>
        <div className="flex gap-2">
          {index > 0 && (
            <Button {...backProps} variant="outline" size="sm" className="border-border/50">
              Back
            </Button>
          )}
          {continuous && (
            <Button
              {...primaryProps}
              size="sm"
              className="bg-gradient-to-r from-blue-600 to-purple-600 hover:from-blue-500 hover:to-purple-500 text-white border-0"
            >
              {isLastStep ? 'Finish' : 'Next'}
            </Button>
          )}
        </div>
      </div>
    </div>
  );
}

// Step definitions with tab requirements
interface TutorialStep extends Step {
  tab?: 'edd' | 'dt' | 'test' | 'tree';
}

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
    target: '[data-tutorial="dt-editor"]',
    content: 'In the grid: "Y" = condition must be true, "N" = must be false, "-" = don\'t care. For actions: "X" = execute this action, "-" = skip it.',
    placement: 'bottom',
    disableBeacon: true,
    title: 'Reading the Grid',
    tab: 'dt',
  },
  {
    target: '[data-tutorial="dt-editor"]',
    content: 'Tables can call other tables in their actions, creating a hierarchy. "Compute_Eligibility" calls "Check_Age", "Check_Income", and "Check_Citizenship" to make its decision.',
    placement: 'bottom',
    disableBeacon: true,
    title: 'Table Hierarchy',
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
    placement: 'left',
    disableBeacon: true,
    title: 'Rule Flow Diagram',
    tab: 'tree',
  },
  {
    target: '[data-tutorial="tree-visualization"]',
    content: 'Understanding this flow helps you debug complex rule chains and see how changes to one table might affect others in the hierarchy.',
    placement: 'bottom',
    disableBeacon: true,
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
  const {
    uiTourActive,
    stopTutorial,
    completeTutorial,
    setTutorialStepIndex,
  } = useOnboardingStore();
  const { projectPath, setActiveTab } = useProjectStore();
  const [stepIndex, setStepIndex] = useState(0);
  const [showTutorial, setShowTutorial] = useState(true);

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
      setShowTutorial(true);
    }
  }, [uiTourActive]);

  const goToStep = useCallback((nextIndex: number) => {
    const nextStep = tutorialSteps[nextIndex] as TutorialStep;

    // Hide tutorial, switch tab, wait for render, then show tutorial at new step
    setShowTutorial(false);

    if (nextStep.tab) {
      setActiveTab(nextStep.tab);
    }

    // Wait for tab content to fully render before showing tutorial
    setTimeout(() => {
      setStepIndex(nextIndex);
      setTutorialStepIndex(nextIndex);
      setShowTutorial(true);
    }, 600);
  }, [setActiveTab, setTutorialStepIndex]);

  const handleJoyrideCallback = useCallback((data: CallBackProps) => {
    const { status, index, action, type } = data;

    if (status === STATUS.FINISHED || status === STATUS.SKIPPED) {
      completeTutorial();
      return;
    }

    // Handle step navigation
    if (type === 'step:after') {
      const nextIndex = index + (action === 'prev' ? -1 : 1);
      if (nextIndex >= 0 && nextIndex < tutorialSteps.length) {
        goToStep(nextIndex);
      }
    }

    // Handle case where target element is not found - skip to next
    if (type === 'error:target_not_found') {
      console.warn('Tutorial target not found for step', index);
      const nextIndex = index + 1;
      if (nextIndex < tutorialSteps.length) {
        goToStep(nextIndex);
      }
    }

    if (action === 'close') {
      stopTutorial();
    }
  }, [completeTutorial, goToStep, stopTutorial]);

  // Don't render anything if tutorial not active or during transitions
  if (!uiTourActive || !projectPath || !showTutorial) {
    return null;
  }

  return (
    <Joyride
      steps={tutorialSteps}
      stepIndex={stepIndex}
      run={true}
      continuous
      showSkipButton
      hideCloseButton
      disableOverlayClose={false}
      disableScrollParentFix
      disableScrolling
      scrollOffset={100}
      spotlightPadding={8}
      spotlightClicks
      callback={handleJoyrideCallback}
      tooltipComponent={CustomTooltip}
      floaterProps={{
        disableAnimation: true,
        hideArrow: true,
        offset: 16,
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
        },
        overlay: {
          // Keep default mixBlendMode: 'hard-light' for spotlight hole effect
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
