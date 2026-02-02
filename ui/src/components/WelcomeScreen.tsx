import { useState, useEffect } from 'react';
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
import { cn } from '@/lib/utils';
import { getSampleProjects, type SampleProject } from '@/api/client';

interface FeatureCardProps {
  icon: React.ReactNode;
  title: string;
  description: string;
  accentColor?: 'blue' | 'purple' | 'green';
}

function FeatureCard({ icon, title, description, accentColor = 'blue' }: FeatureCardProps) {
  const borderColorMap = {
    blue: 'hover:border-blue-500/30',
    purple: 'hover:border-purple-500/30',
    green: 'hover:border-green-500/30',
  };

  const iconBgMap = {
    blue: 'bg-blue-500/10',
    purple: 'bg-purple-500/10',
    green: 'bg-green-500/10',
  };

  return (
    <div className={cn(
      "flex flex-col items-center text-center p-6 glass rounded-2xl",
      "transition-all duration-300 hover:shadow-lg hover:shadow-black/20",
      borderColorMap[accentColor]
    )}>
      <div className={`p-3 ${iconBgMap[accentColor]} rounded-xl mb-4`}>
        {icon}
      </div>
      <h3 className="text-lg font-semibold mb-2">{title}</h3>
      <p className="text-sm text-muted-foreground">{description}</p>
    </div>
  );
}

export function WelcomeScreen() {
  const { openProject, autoSelectFirstItems } = useProjectStore();
  const { setShowWelcome, startTutorial } = useOnboardingStore();
  const [customPathDialogOpen, setCustomPathDialogOpen] = useState(false);
  const [customPath, setCustomPath] = useState('');
  const [isLoading, setIsLoading] = useState(false);
  const [sampleProjects, setSampleProjects] = useState<SampleProject[]>([]);
  const [samplesLoaded, setSamplesLoaded] = useState(false);

  // Fetch available sample projects from the backend on mount
  useEffect(() => {
    const fetchSamples = async () => {
      try {
        const result = await getSampleProjects();
        if (result.success && result.samples) {
          setSampleProjects(result.samples);
        }
      } catch (error) {
        console.error('Failed to fetch sample projects:', error);
      }
      setSamplesLoaded(true);
    };
    fetchSamples();
  }, []);

  // Find the CHIP project from the discovered samples
  const chipProject = sampleProjects.find(p => p.name === 'CHIP');

  const handleOpenChipProject = async () => {
    // If CHIP project was discovered, use its path directly
    if (chipProject) {
      setIsLoading(true);
      const success = await openProject(chipProject.path);
      if (success) {
        await autoSelectFirstItems();
        setShowWelcome(false);
        // Always start the tutorial when opening CHIP sample project
        startTutorial();
      }
      setIsLoading(false);
      return;
    }
    // If not discovered, open dialog for manual entry
    setCustomPathDialogOpen(true);
  };

  const handleSkipTutorial = async () => {
    // If CHIP project was discovered, use its path directly
    if (chipProject) {
      setIsLoading(true);
      const success = await openProject(chipProject.path);
      if (success) {
        await autoSelectFirstItems();
        setShowWelcome(false);
        // Don't start tutorial - user explicitly chose to skip
      }
      setIsLoading(false);
      return;
    }
    // If not discovered, open dialog for manual entry
    setCustomPathDialogOpen(true);
  };

  const handleOpenCustomProject = async () => {
    if (!customPath) return;
    setIsLoading(true);
    const success = await openProject(customPath);
    if (success) {
      // Auto-select first items so all editors have content visible
      await autoSelectFirstItems();
      setCustomPathDialogOpen(false);
      setShowWelcome(false);
    }
    setIsLoading(false);
  };

  return (
    <div className="h-screen flex flex-col items-center justify-center bg-gradient-to-br from-background via-background to-blue-950/20 p-8">
      <div className="max-w-4xl w-full space-y-10">
        {/* Hero Section */}
        <div className="text-center space-y-4">
          <div className="flex items-center justify-center gap-4 mb-6">
            <div className="p-4 bg-gradient-to-br from-blue-500 to-purple-600 rounded-2xl shadow-lg shadow-blue-500/25">
              <Table2 className="h-12 w-12 text-white" />
            </div>
            <h1 className="text-6xl font-bold bg-gradient-to-r from-blue-400 via-purple-400 to-blue-400 bg-clip-text text-transparent">
              DTRules
            </h1>
          </div>
          <p className="text-xl text-muted-foreground max-w-2xl mx-auto">
            A powerful visual editor for creating and managing decision tables,
            entity definitions, and business rules with an intuitive interface.
          </p>
        </div>

        {/* Feature Cards */}
        <div className="grid grid-cols-1 md:grid-cols-3 gap-6">
          <FeatureCard
            icon={<FileText className="h-6 w-6 text-blue-400" />}
            title="Entity Definitions"
            description="Define and manage entities with attributes, types, and validation rules in a visual editor."
            accentColor="blue"
          />
          <FeatureCard
            icon={<Table2 className="h-6 w-6 text-purple-400" />}
            title="Decision Tables"
            description="Create condition-action matrices to encode complex business logic in an easy-to-read format."
            accentColor="purple"
          />
          <FeatureCard
            icon={<GitBranch className="h-6 w-6 text-green-400" />}
            title="Tree Visualization"
            description="Visualize decision table dependencies and execution flow as an interactive tree diagram."
            accentColor="green"
          />
        </div>

        {/* Action Buttons */}
        <div className="flex flex-col items-center gap-4">
          <Button
            size="lg"
            className="w-80 bg-gradient-to-r from-blue-600 to-purple-600 hover:from-blue-500 hover:to-purple-500 text-white border-0 shadow-lg hover:shadow-xl hover:shadow-blue-500/25 transition-all duration-300"
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
              className="hover:bg-secondary/80"
            >
              <BookOpen className="h-4 w-4 mr-2" />
              Skip Tutorial
            </Button>

            <Button
              variant="outline"
              onClick={() => setCustomPathDialogOpen(true)}
              disabled={isLoading}
              className="border-border/50 hover:border-border hover:bg-accent/50"
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
