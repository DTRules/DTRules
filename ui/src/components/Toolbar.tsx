import { useState } from 'react';
import { Button } from '@/components/ui/button';
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from '@/components/ui/dialog';
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu';
import { ScrollArea } from '@/components/ui/scroll-area';
import { ProjectPicker } from '@/components/ProjectPicker';
import { useProjectStore } from '@/stores/projectStore';
import { useOnboardingStore } from '@/stores/onboardingStore';
import {
  FolderOpen,
  Save,
  FileCheck,
  Play,
  RefreshCw,
  HelpCircle,
  BookOpen,
  Home,
  MessageCircleQuestion,
  ChevronLeft,
  ChevronRight,
} from 'lucide-react';

const faqItems = [
  {
    question: 'What is a Decision Table?',
    answer: 'A decision table is a visual way to represent business rules. Each column represents a complete rule with conditions (IF) and actions (THEN). This format makes rules easy to read, test, and maintain without programming knowledge.',
  },
  {
    question: 'What do Y, N, -, and X mean in the grid?',
    answer: 'Y = condition must be true, N = condition must be false, - = don\'t care (any value), X = execute this action. Read each column top-to-bottom: "IF these conditions THEN these actions".',
  },
  {
    question: 'What are Entities?',
    answer: 'Entities define your data vocabulary - the information your rules work with. Like database tables or classes, they describe structures such as "client" (with age, citizenship) or "income" (with amounts). Think of entities as nouns and decision tables as verbs.',
  },
  {
    question: 'How do I test my rules?',
    answer: 'Use the Test & Execute tab. Provide sample data matching your entity structure, click Execute, and verify the results. Enable tracing to see exactly which rules fired and why.',
  },
  {
    question: 'How do tables call other tables?',
    answer: 'Decision tables can reference other tables in their actions, creating a hierarchy. For example, "Compute_Eligibility" might call "Check_Age" and "Check_Income" tables. The Tree View visualizes these connections.',
  },
  {
    question: 'What is the CHIP example project?',
    answer: 'CHIP (Children\'s Health Insurance Program) is a sample project demonstrating real-world eligibility rules. It includes entities for clients, cases, and income, plus decision tables for age, citizenship, and income eligibility checks.',
  },
  {
    question: 'How do I save my work?',
    answer: 'Click the Save button in the toolbar. Your entities and decision tables are saved to XML files that can be version-controlled (Git) and deployed to production systems.',
  },
  {
    question: 'What\'s the difference between FIRST and ALL table types?',
    answer: 'FIRST stops after the first matching rule (like if-else-if). ALL executes every matching rule (useful when multiple actions should run). Choose based on your business logic needs.',
  },
];

export function Toolbar() {
  const {
    openProject,
    saveProject,
    projectPath,
    readOnly,
    isLoading,
    loadEDD,
    loadDecisionTables,
    closeProject,
    canGoBack,
    canGoForward,
    goBack,
    goForward,
  } = useProjectStore();
  const { startTutorial, showWelcomeScreen } = useOnboardingStore();
  const [openDialogOpen, setOpenDialogOpen] = useState(false);
  const [faqDialogOpen, setFaqDialogOpen] = useState(false);

  const handleOpenProject = async (path: string) => {
    const success = await openProject(path);
    if (success) {
      setOpenDialogOpen(false);
    }
  };

  const handleRefresh = async () => {
    await loadEDD();
    await loadDecisionTables();
  };

  const handleStartTutorial = () => {
    if (projectPath) {
      startTutorial();
    }
  };

  const handleShowWelcome = () => {
    closeProject();
    showWelcomeScreen();
  };

  return (
    <div className="h-14 border-b border-border/50 flex items-center px-4 gap-2 bg-gradient-to-r from-background via-background to-blue-950/10">
      {/* Logo/Title */}
      <div className="font-bold text-lg mr-4 bg-gradient-to-r from-blue-400 to-purple-400 bg-clip-text text-transparent">
        DTRules
      </div>

      {/* Browser-style Navigation */}
      <div className="flex items-center mr-2 bg-muted/30 rounded-lg p-1">
        <Button
          variant="ghost"
          size="icon"
          onClick={goBack}
          disabled={!canGoBack()}
          className="h-8 w-8"
          title="Go back"
        >
          <ChevronLeft className={`h-5 w-5 ${!canGoBack() ? 'text-muted-foreground/40' : ''}`} />
        </Button>
        <Button
          variant="ghost"
          size="icon"
          onClick={goForward}
          disabled={!canGoForward()}
          className="h-8 w-8"
          title="Go forward"
        >
          <ChevronRight className={`h-5 w-5 ${!canGoForward() ? 'text-muted-foreground/40' : ''}`} />
        </Button>
      </div>

      {/* File Actions Group */}
      <div className="flex items-center bg-muted/30 rounded-lg p-1 gap-1">
        {/* Open Project */}
        <Dialog open={openDialogOpen} onOpenChange={setOpenDialogOpen}>
          <DialogTrigger asChild>
            <Button
              variant="ghost"
              size="sm"
              className="hover:bg-accent/50"
              disabled={readOnly}
              title={readOnly ? 'Server is read-only' : undefined}
            >
              <FolderOpen className="h-4 w-4 mr-2" />
              Open
            </Button>
          </DialogTrigger>
        <DialogContent className="sm:max-w-[640px]">
          <DialogHeader>
            <DialogTitle>Open Project</DialogTitle>
            <DialogDescription>
              Pick a recent project, or browse to a directory containing EDD and DT XML files.
            </DialogDescription>
          </DialogHeader>
          <ProjectPicker onOpen={handleOpenProject} />
        </DialogContent>
      </Dialog>

        {/* Save */}
        <Button
          variant="ghost"
          size="sm"
          onClick={() => saveProject()}
          disabled={!projectPath || isLoading || readOnly}
          title={readOnly ? 'Server is read-only' : undefined}
          data-tutorial="toolbar-save"
          className="hover:bg-accent/50"
        >
          <Save className="h-4 w-4 mr-2" />
          Save
        </Button>

        {/* Refresh */}
        <Button
          variant="ghost"
          size="sm"
          onClick={handleRefresh}
          disabled={!projectPath || isLoading}
          data-tutorial="toolbar-refresh"
          className="hover:bg-accent/50"
        >
          <RefreshCw className="h-4 w-4 mr-2" />
          Refresh
        </Button>
      </div>

      <div className="flex-1" />

      {/* Action Group */}
      <div className="flex items-center bg-muted/30 rounded-lg p-1 gap-1">
        {/* Validate */}
        <Button
          variant="ghost"
          size="sm"
          disabled={!projectPath || isLoading}
          className="hover:bg-accent/50"
        >
          <FileCheck className="h-4 w-4 mr-2" />
          Validate
        </Button>

        {/* Execute */}
        <Button
          variant="ghost"
          size="sm"
          disabled={!projectPath || isLoading}
          className="hover:bg-blue-500/20 hover:text-blue-400 transition-colors"
        >
          <Play className="h-4 w-4 mr-2" />
          Execute
        </Button>
      </div>

      {/* FAQ Dialog */}
      <Dialog open={faqDialogOpen} onOpenChange={setFaqDialogOpen}>
        <DialogTrigger asChild>
          <Button variant="ghost" size="sm">
            <MessageCircleQuestion className="h-4 w-4 mr-2" />
            FAQ
          </Button>
        </DialogTrigger>
        <DialogContent className="sm:max-w-[600px] max-h-[80vh]">
          <DialogHeader>
            <DialogTitle>Frequently Asked Questions</DialogTitle>
            <DialogDescription>
              Common questions about DTRules and decision tables
            </DialogDescription>
          </DialogHeader>
          <ScrollArea className="h-[400px] pr-4">
            <div className="space-y-3">
              {faqItems.map((item, index) => (
                <div key={index} className="p-4 rounded-xl bg-muted/30 border border-border/50 hover:border-blue-500/30 transition-colors">
                  <h4 className="font-medium text-sm mb-2 text-foreground">{item.question}</h4>
                  <p className="text-sm text-muted-foreground leading-relaxed">{item.answer}</p>
                </div>
              ))}
            </div>
          </ScrollArea>
        </DialogContent>
      </Dialog>

      {/* Help Menu */}
      <DropdownMenu>
        <DropdownMenuTrigger asChild>
          <Button variant="ghost" size="sm" data-tutorial="help-menu">
            <HelpCircle className="h-4 w-4 mr-2" />
            Help
          </Button>
        </DropdownMenuTrigger>
        <DropdownMenuContent align="end">
          <DropdownMenuItem onClick={handleStartTutorial} disabled={!projectPath}>
            <BookOpen className="h-4 w-4 mr-2" />
            Start Tutorial
          </DropdownMenuItem>
          <DropdownMenuSeparator />
          <DropdownMenuItem onClick={() => setFaqDialogOpen(true)}>
            <MessageCircleQuestion className="h-4 w-4 mr-2" />
            FAQ
          </DropdownMenuItem>
          <DropdownMenuSeparator />
          <DropdownMenuItem onClick={handleShowWelcome}>
            <Home className="h-4 w-4 mr-2" />
            Show Welcome Screen
          </DropdownMenuItem>
        </DropdownMenuContent>
      </DropdownMenu>
    </div>
  );
}
