import { useState } from 'react';
import { Button } from '@/components/ui/button';
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { useProjectStore } from '@/stores/projectStore';
import { useOnboardingStore } from '@/stores/onboardingStore';
import { FileText, Table2, GitBranch, FolderOpen, Play, BookOpen } from 'lucide-react';

const CHIP_PROJECT_PATH = '/home/paul/DTRules/sampleprojects/CHIP/xml';

interface FeatureCardProps {
  icon: React.ReactNode;
  title: string;
  description: string;
}

function FeatureCard({ icon, title, description }: FeatureCardProps) {
  return (
    <div className="flex flex-col items-center p-6 bg-card rounded-lg border border-border hover:border-primary/50 transition-colors">
      <div className="p-3 bg-primary/10 rounded-full mb-4">
        {icon}
      </div>
      <h3 className="text-lg font-semibold mb-2">{title}</h3>
      <p className="text-sm text-muted-foreground text-center">{description}</p>
    </div>
  );
}

export function WelcomeScreen() {
  const { openProject } = useProjectStore();
  const { setShowWelcome, setOfferTutorial, tutorialCompleted, dontAskAgain } = useOnboardingStore();
  const [customPathDialogOpen, setCustomPathDialogOpen] = useState(false);
  const [customPath, setCustomPath] = useState('');
  const [isLoading, setIsLoading] = useState(false);

  const handleOpenChipProject = async () => {
    setIsLoading(true);
    const success = await openProject(CHIP_PROJECT_PATH);
    setIsLoading(false);
    if (success) {
      setShowWelcome(false);
      if (!tutorialCompleted && !dontAskAgain) {
        setOfferTutorial(true);
      }
    }
  };

  const handleSkipTutorial = async () => {
    setIsLoading(true);
    const success = await openProject(CHIP_PROJECT_PATH);
    setIsLoading(false);
    if (success) {
      setShowWelcome(false);
    }
  };

  const handleOpenCustomProject = async () => {
    if (!customPath) return;
    setIsLoading(true);
    const success = await openProject(customPath);
    setIsLoading(false);
    if (success) {
      setCustomPathDialogOpen(false);
      setShowWelcome(false);
      if (!tutorialCompleted && !dontAskAgain) {
        setOfferTutorial(true);
      }
    }
  };

  return (
    <div className="h-screen flex flex-col items-center justify-center bg-background p-8">
      <div className="max-w-4xl w-full space-y-12">
        {/* Hero Section */}
        <div className="text-center space-y-4">
          <div className="flex items-center justify-center gap-3 mb-6">
            <div className="p-3 bg-primary rounded-lg">
              <Table2 className="h-10 w-10 text-primary-foreground" />
            </div>
            <h1 className="text-5xl font-bold">DTRules</h1>
          </div>
          <p className="text-xl text-muted-foreground max-w-2xl mx-auto">
            A powerful visual editor for creating and managing decision tables,
            entity definitions, and business rules with an intuitive interface.
          </p>
        </div>

        {/* Feature Cards */}
        <div className="grid grid-cols-1 md:grid-cols-3 gap-6">
          <FeatureCard
            icon={<FileText className="h-6 w-6 text-primary" />}
            title="Entity Definitions"
            description="Define and manage entities with attributes, types, and validation rules in a visual editor."
          />
          <FeatureCard
            icon={<Table2 className="h-6 w-6 text-primary" />}
            title="Decision Tables"
            description="Create condition-action matrices to encode complex business logic in an easy-to-read format."
          />
          <FeatureCard
            icon={<GitBranch className="h-6 w-6 text-primary" />}
            title="Tree Visualization"
            description="Visualize decision table dependencies and execution flow as an interactive tree diagram."
          />
        </div>

        {/* Action Buttons */}
        <div className="flex flex-col items-center gap-4">
          <Button
            size="lg"
            className="w-80"
            onClick={handleOpenChipProject}
            disabled={isLoading}
          >
            <Play className="h-5 w-5 mr-2" />
            Open CHIP Sample Project
          </Button>

          <div className="flex gap-4">
            <Button
              variant="secondary"
              onClick={handleSkipTutorial}
              disabled={isLoading}
            >
              <BookOpen className="h-4 w-4 mr-2" />
              Skip Tutorial
            </Button>

            <Button
              variant="outline"
              onClick={() => setCustomPathDialogOpen(true)}
              disabled={isLoading}
            >
              <FolderOpen className="h-4 w-4 mr-2" />
              Open Custom Project
            </Button>
          </div>
        </div>

        {/* Footer hint */}
        <p className="text-center text-sm text-muted-foreground">
          New to DTRules? Click "Open CHIP Sample Project" to explore with a guided tour.
        </p>
      </div>

      {/* Custom Path Dialog */}
      <Dialog open={customPathDialogOpen} onOpenChange={setCustomPathDialogOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Open Custom Project</DialogTitle>
            <DialogDescription>
              Enter the path to a DTRules project directory containing EDD and DT XML files.
            </DialogDescription>
          </DialogHeader>
          <div className="grid gap-4 py-4">
            <div className="grid gap-2">
              <Label htmlFor="customProjectPath">Project Path</Label>
              <Input
                id="customProjectPath"
                placeholder="/path/to/project/xml"
                value={customPath}
                onChange={(e) => setCustomPath(e.target.value)}
                onKeyDown={(e) => e.key === 'Enter' && handleOpenCustomProject()}
              />
            </div>
          </div>
          <DialogFooter>
            <Button variant="outline" onClick={() => setCustomPathDialogOpen(false)}>
              Cancel
            </Button>
            <Button onClick={handleOpenCustomProject} disabled={!customPath || isLoading}>
              Open Project
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  );
}
