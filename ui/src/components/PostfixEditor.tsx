import { useEffect, useRef, useState } from 'react';
import Editor, { Monaco } from '@monaco-editor/react';
import { compileExpression, getOperators, getEntityFields } from '@/api/client';
import { cn } from '@/lib/utils';
import type { editor, languages, IPosition } from 'monaco-editor';

interface PostfixEditorProps {
  value: string;
  onChange: (value: string) => void;
  height?: string;
  className?: string;
}

export function PostfixEditor({ value, onChange, height = '200px', className }: PostfixEditorProps) {
  const editorRef = useRef<editor.IStandaloneCodeEditor | null>(null);
  const [operators, setOperators] = useState<string[]>([]);
  const [fields, setFields] = useState<string[]>([]);
  const [validationError, setValidationError] = useState<string | null>(null);
  const [warnings, setWarnings] = useState<string[]>([]);

  // Load operators and fields for autocomplete
  useEffect(() => {
    const loadAutocomplete = async () => {
      try {
        const [opsResponse, fieldsResponse] = await Promise.all([
          getOperators(),
          getEntityFields(),
        ]);
        setOperators(opsResponse.operators || []);
        setFields(fieldsResponse.fields || []);
      } catch (err) {
        console.error('Failed to load autocomplete data:', err);
      }
    };
    loadAutocomplete();
  }, []);

  // Validate expression on change
  useEffect(() => {
    const validateExpression = async () => {
      if (!value.trim()) {
        setValidationError(null);
        setWarnings([]);
        return;
      }

      try {
        const response = await compileExpression(value);
        if (response.valid) {
          setValidationError(null);
          setWarnings(response.warnings || []);
        } else {
          setValidationError(response.error || 'Invalid expression');
          setWarnings([]);
        }
      } catch {
        setValidationError(null);
        setWarnings([]);
      }
    };

    const debounce = setTimeout(validateExpression, 500);
    return () => clearTimeout(debounce);
  }, [value]);

  const handleEditorMount = (editor: editor.IStandaloneCodeEditor, monaco: Monaco) => {
    editorRef.current = editor;

    // Register a custom language for postfix notation
    monaco.languages.register({ id: 'postfix' });

    // Define tokens for syntax highlighting
    monaco.languages.setMonarchTokensProvider('postfix', {
      defaultToken: 'identifier',
      tokenizer: {
        root: [
          // Comments
          [/\/\/.*$/, 'comment'],

          // Strings
          [/"[^"]*"/, 'string'],

          // Numbers
          [/-?\d+\.?\d*/, 'number'],

          // Boolean literals
          [/\b(true|false)\b/, 'keyword'],

          // Control flow
          [/\b(if|ifelse|forall|loop|exit|repeat|for)\b/, 'keyword.control'],

          // Entity operators
          [/\b(entitypush|entitypop|createentity|findmatch|findall|allocate|deallocate)\b/, 'keyword.entity'],

          // Stack operators
          [/\b(dup|pop|swap|exch|roll|copy|index|clear|count)\b/, 'keyword.stack'],

          // Name references
          [/\/[a-zA-Z_][a-zA-Z0-9_*.]*/, 'variable.name'],

          // Entity.field references
          [/[a-zA-Z_][a-zA-Z0-9_]*\.[a-zA-Z_][a-zA-Z0-9_]*/, 'variable.field'],

          // Operators
          [/[+\-*/<>=!]+/, 'operator'],

          // Blocks
          [/[{}[\]]/, 'delimiter.bracket'],

          // Identifiers
          [/[a-zA-Z_][a-zA-Z0-9_]*/, 'identifier'],
        ],
      },
    });

    // Define theme colors
    monaco.editor.defineTheme('postfix-dark', {
      base: 'vs-dark',
      inherit: true,
      rules: [
        { token: 'comment', foreground: '6A9955' },
        { token: 'string', foreground: 'CE9178' },
        { token: 'number', foreground: 'B5CEA8' },
        { token: 'keyword', foreground: '569CD6' },
        { token: 'keyword.control', foreground: 'C586C0' },
        { token: 'keyword.entity', foreground: '4EC9B0' },
        { token: 'keyword.stack', foreground: 'DCDCAA' },
        { token: 'variable.name', foreground: '9CDCFE' },
        { token: 'variable.field', foreground: '4FC1FF' },
        { token: 'operator', foreground: 'D4D4D4' },
        { token: 'identifier', foreground: 'D4D4D4' },
      ],
      colors: {},
    });

    monaco.editor.setTheme('postfix-dark');

    // Register autocomplete provider
    monaco.languages.registerCompletionItemProvider('postfix', {
      provideCompletionItems: (model: editor.ITextModel, position: IPosition) => {
        const word = model.getWordUntilPosition(position);
        const range = {
          startLineNumber: position.lineNumber,
          endLineNumber: position.lineNumber,
          startColumn: word.startColumn,
          endColumn: word.endColumn,
        };

        const suggestions: languages.CompletionItem[] = [];

        // Add operators
        operators.forEach((op) => {
          suggestions.push({
            label: op,
            kind: monaco.languages.CompletionItemKind.Function,
            insertText: op,
            range,
            detail: 'Operator',
          });
        });

        // Add entity fields
        fields.forEach((field) => {
          suggestions.push({
            label: field,
            kind: monaco.languages.CompletionItemKind.Variable,
            insertText: field,
            range,
            detail: 'Entity field',
          });
        });

        return { suggestions };
      },
    });
  };

  return (
    <div className={cn("flex flex-col", className)}>
      <div className="flex-1 border border-border rounded-md overflow-hidden">
        <Editor
          height={height}
          defaultLanguage="postfix"
          value={value}
          onChange={(v) => onChange(v || '')}
          onMount={handleEditorMount}
          options={{
            minimap: { enabled: false },
            fontSize: 13,
            lineNumbers: 'off',
            glyphMargin: false,
            folding: false,
            lineDecorationsWidth: 0,
            lineNumbersMinChars: 0,
            scrollBeyondLastLine: false,
            automaticLayout: true,
            wordWrap: 'on',
            theme: 'postfix-dark',
          }}
        />
      </div>
      {(validationError || warnings.length > 0) && (
        <div className="mt-2 text-xs">
          {validationError && (
            <div className="text-red-500">{validationError}</div>
          )}
          {warnings.map((warning, i) => (
            <div key={i} className="text-yellow-500">{warning}</div>
          ))}
        </div>
      )}
    </div>
  );
}
