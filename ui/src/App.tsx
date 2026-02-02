import { useEffect, useState } from 'react';
import { useProjectStore } from '@/stores/projectStore';
import { useOnboardingStore } from '@/stores/onboardingStore';
import { ProjectExplorer } from '@/components/ProjectExplorer';
import { EDDEditor } from '@/components/EDDEditor';
import { DTEditor } from '@/components/DTEditor';
import { TestPanel } from '@/components/TestPanel';
import { TreeVisualization } from '@/components/TreeVisualization';
import { Toolbar } from '@/components/Toolbar';
import { StatusBar } from '@/components/StatusBar';
import { WelcomeScreen } from '@/components/WelcomeScreen';
import { TutorialController } from '@/components/TutorialController';
import { TutorialOfferDialog } from '@/components/TutorialOfferDialog';
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs';
import { Toaster } from '@/components/ui/toaster';
import { useToast } from '@/components/ui/use-toast';
import { healthCheck } from '@/api/client';

function App() {
  const { activeTab, setActiveTabWithHistory, projectPath, error, clearError, isLoading } = useProjectStore();
  const { showWelcome } = useOnboardingStore();
  const { toast } = useToast();
  const [backendConnected, setBackendConnected] = useState(false);

  // Check backend connection
  useEffect(() => {
    const checkBackend = async () => {
      try {
        const response = await healthCheck();
        setBackendConnected(response.status === 'ok');
      } catch {
        setBackendConnected(false);
      }
    };

    checkBackend();
    const interval = setInterval(checkBackend, 5000);
    return () => clearInterval(interval);
  }, []);

  // Show errors as toasts
  useEffect(() => {
    if (error) {
      toast({
        variant: 'destructive',
        title: 'Error',
        description: error,
      });
      clearError();
    }
  }, [error, toast, clearError]);

  // Show welcome screen if no project is open and showWelcome is true
  if (showWelcome && !projectPath) {
    return (
      <div className="h-screen bg-background text-foreground dark">
        <WelcomeScreen />
        <Toaster />
      </div>
    );
  }

  return (
    <div className="h-screen flex flex-col bg-background text-foreground dark">
      {/* Toolbar */}
      <div data-tutorial="toolbar">
        <Toolbar />
      </div>

      {/* Main content */}
      <div className="flex-1 flex overflow-hidden">
        {/* Project Explorer */}
        <div
          className="w-64 border-r border-border/50 overflow-hidden flex flex-col"
          data-tutorial="project-explorer"
        >
          <ProjectExplorer />
        </div>

        {/* Editor Area */}
        <div className="flex-1 flex flex-col overflow-hidden">
          <Tabs value={activeTab} onValueChange={(v) => setActiveTabWithHistory(v as typeof activeTab)} className="flex-1 flex flex-col">
            <div className="border-b border-border/50 px-4 bg-muted/20">
              <TabsList className="h-10">
                <TabsTrigger value="edd" data-tutorial="tab-edd">Entity Editor</TabsTrigger>
                <TabsTrigger value="dt" data-tutorial="tab-dt">Decision Tables</TabsTrigger>
                <TabsTrigger value="test" data-tutorial="tab-test">Test & Execute</TabsTrigger>
                <TabsTrigger value="tree" data-tutorial="tab-tree">Tree View</TabsTrigger>
              </TabsList>
            </div>

            <div className="flex-1 overflow-hidden">
              <TabsContent value="edd" className="h-full m-0 p-0">
                <EDDEditor />
              </TabsContent>

              <TabsContent value="dt" className="h-full m-0 p-0">
                <DTEditor />
              </TabsContent>

              <TabsContent value="test" className="h-full m-0 p-0">
                <TestPanel />
              </TabsContent>

              <TabsContent value="tree" className="h-full m-0 p-0">
                <TreeVisualization />
              </TabsContent>
            </div>
          </Tabs>
        </div>
      </div>

      {/* Status Bar */}
      <StatusBar
        projectPath={projectPath}
        isLoading={isLoading}
        backendConnected={backendConnected}
      />

      <Toaster />
      <TutorialController />
      <TutorialOfferDialog />
    </div>
  );
}

export default App;
