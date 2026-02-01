import { ConceptModals } from '@/components/ConceptModals';
import { GuidedTutorial } from '@/components/GuidedTutorial';
import { TutorialNavigation, ConceptPhaseProgress } from '@/components/TutorialNavigation';

/**
 * TutorialController orchestrates the two-phase tutorial system:
 *
 * Phase 1: Concept Modals (Educational)
 * - Centered modal dialogs teaching decision table fundamentals
 * - No element targeting, so no positioning issues
 * - 5 steps covering concepts, entities, tables, CHIP example, and workflow
 *
 * Phase 2: UI Tour (Simplified Joyride)
 * - Only targets stable elements (tabs, toolbar buttons)
 * - Avoids elements inside ScrollArea or overflow containers
 * - 6 steps pointing at tab triggers and toolbar
 *
 * Flow:
 * Tutorial Start -> Concept Modals (5 steps) -> UI Tour (6 steps) -> Complete
 */
export function TutorialController() {
  return (
    <>
      <ConceptModals />
      <GuidedTutorial />
      <TutorialNavigation />
      <ConceptPhaseProgress />
    </>
  );
}
