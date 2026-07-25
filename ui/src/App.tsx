import { useEffect, useState } from 'react';
import { useProjectStore } from '@/stores/projectStore';
import { useOnboardingStore } from '@/stores/onboardingStore';
import { ProjectExplorer } from '@/components/ProjectExplorer';
import { EDDEditor } from '@/components/EDDEditor';
import { DTEditor } from '@/components/DTEditor';
import { DebugPanel } from '@/components/DebugPanel';
import { TestPanel } from '@/components/TestPanel';
import { Toolbar } from '@/components/Toolbar';
import { StatusBar } from '@/components/StatusBar';
import { WelcomeScreen } from '@/components/WelcomeScreen';
import { TutorialController } from '@/components/TutorialController';
import { TutorialOfferDialog } from '@/components/TutorialOfferDialog';
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs';
import { Toaster } from '@/components/ui/toaster';
import { useToast } from '@/components/ui/use-toast';
import { getCurrentProject, healthCheck } from '@/api/client';
import { deriveProjectName } from '@/lib/utils';

function App() {
  const { activeTab, setActiveTabWithHistory, projectPath, error, clearError, isLoading, adoptProject, autoSelectFirstItems, setReadOnly } = useProjectStore();
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

  // Adopt a project the backend already has loaded (e.g. `dtrules edit <dir>`)
  // so the editor opens straight into it instead of the welcome screen.
  useEffect(() => {
    if (projectPath) return;
    (async () => {
      try {
        const current = await getCurrentProject();
        if (current.success) {
          setReadOnly(!!current.readOnly);
          if (current.path) {
            await adoptProject(
              current.path,
              current.eddFiles || [],
              current.dtFiles || [],
              current.mapFiles || []
            );
            await autoSelectFirstItems();
          }
        }
      } catch {
        // Backend absent or older — welcome screen handles it
      }
    })();
    // eslint-disable-next-line react-hooks/exhaustive-deps
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
          {/* Project identity: dominant single line above the editor tabs */}
          {projectPath && (
            <div className="px-4 py-2 border-b border-border/50 bg-gradient-to-r from-blue-950/30 to-transparent flex items-baseline gap-3 min-w-0">
              <span className="text-2xl font-bold leading-tight">{deriveProjectName(projectPath)}</span>
              <span className="text-xs font-mono text-muted-foreground truncate" title={projectPath}>
                {projectPath}
              </span>
            </div>
          )}
          <Tabs value={activeTab} onValueChange={(v) => setActiveTabWithHistory(v as typeof activeTab)} className="flex-1 flex flex-col">
            <div className="border-b border-border/50 px-4 bg-muted/20">
              <TabsList className="h-10">
                <TabsTrigger value="edd" data-tutorial="tab-edd">Entity Editor</TabsTrigger>
                <TabsTrigger value="dt" data-tutorial="tab-dt">Decision Tables</TabsTrigger>
                <TabsTrigger value="test" data-tutorial="tab-test">Test & Execute</TabsTrigger>
                <TabsTrigger value="debug" data-tutorial="tab-debug">Debug</TabsTrigger>
              </TabsList>
            </div>

            <div className="flex-1 overflow-hidden relative">
              {/* Keep all tabs mounted to avoid loading delays during tutorial */}
              <TabsContent value="edd" className="absolute inset-0 data-[state=inactive]:hidden" forceMount>
                <EDDEditor />
              </TabsContent>

              <TabsContent value="dt" className="absolute inset-0 data-[state=inactive]:hidden" forceMount>
                <DTEditor />
              </TabsContent>

              <TabsContent value="test" className="absolute inset-0 data-[state=inactive]:hidden" forceMount>
                <TestPanel />
              </TabsContent>


              <TabsContent value="debug" className="absolute inset-0 data-[state=inactive]:hidden" forceMount>
                <DebugPanel />
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
