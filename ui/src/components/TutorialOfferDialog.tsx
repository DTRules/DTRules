import { Button } from '@/components/ui/button';
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog';
import { useOnboardingStore } from '@/stores/onboardingStore';
import { BookOpen, X, XCircle } from 'lucide-react';

export function TutorialOfferDialog() {
  const {
    offerTutorial,
    setOfferTutorial,
    startTutorial,
    setDontAskAgain,
  } = useOnboardingStore();

  const handleStartTour = () => {
    setOfferTutorial(false);
    startTutorial();
  };

  const handleSkipForNow = () => {
    setOfferTutorial(false);
  };

  const handleDontAskAgain = () => {
    setDontAskAgain(true);
    setOfferTutorial(false);
  };

  return (
    <Dialog open={offerTutorial} onOpenChange={setOfferTutorial}>
      <DialogContent className="sm:max-w-md">
        <DialogHeader>
          <DialogTitle className="flex items-center gap-2">
            <BookOpen className="h-5 w-5 text-primary" />
            Would you like a guided tour?
          </DialogTitle>
          <DialogDescription>
            Take a quick interactive tour to learn about the key features of DTRules.
            The tour will guide you through entities, decision tables, visualizations, and testing.
          </DialogDescription>
        </DialogHeader>
        <DialogFooter className="flex-col sm:flex-row gap-2">
          <Button variant="default" onClick={handleStartTour} className="w-full sm:w-auto">
            Start Tour
          </Button>
          <Button variant="secondary" onClick={handleSkipForNow} className="w-full sm:w-auto">
            <X className="h-4 w-4 mr-2" />
            Skip for now
          </Button>
          <Button variant="ghost" onClick={handleDontAskAgain} className="w-full sm:w-auto text-muted-foreground">
            <XCircle className="h-4 w-4 mr-2" />
            Don't ask again
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
