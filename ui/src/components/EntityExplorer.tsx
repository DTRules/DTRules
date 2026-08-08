/**
 * EntityExplorer - structural inspector for one entity instance at the
 * current replay position.
 *
 * Launched from a frame on the debugger's entity stack. Shows the
 * instance's actual fields and drills into what composes it: entity
 * references expand into THEIR fields, arrays expand into elements
 * (which may themselves be entities), with paging for long arrays.
 * Everything is fetched lazily — one request per expansion — so deep
 * structures stay cheap. Values reflect the current replay position;
 * the panel refreshes when the position moves (it is remounted keyed
 * by position).
 *
 * @module components/EntityExplorer
 */

import { useEffect, useState } from 'react';
import { debugArray, debugEntity, type ExplorerField, type ExplorerValue } from '@/api/client';
import { Button } from '@/components/ui/button';
import { ScrollArea } from '@/components/ui/scroll-area';
import { X } from 'lucide-react';

export function EntityExplorer({
  rootId,
  rootName,
  position,
  onClose,
}: {
  rootId: number;
  rootName: string;
  position: number;
  onClose: () => void;
}) {
  const [rootNumber, setRootNumber] = useState('');
  return (
    <div className="absolute inset-0 z-20 bg-background/60" onClick={onClose}>
      <div
        className="absolute right-0 top-0 bottom-0 w-[520px] border-l border-border bg-background shadow-2xl flex flex-col"
        onClick={(e) => e.stopPropagation()}
      >
        <div className="px-4 py-2 border-b border-border/50 flex items-center gap-2">
          <span className="text-sm font-semibold">Entity explorer</span>
          <span className="font-mono text-xs text-muted-foreground">
            {rootName}
            {rootNumber && <span title="EDD entity number"> №{rootNumber}</span>} #{rootId} · at node{' '}
            {position.toLocaleString()}
          </span>
          <Button variant="ghost" size="icon" className="h-6 w-6 ml-auto" onClick={onClose} title="Close (Esc)">
            <X className="h-4 w-4" />
          </Button>
        </div>
        <ScrollArea className="flex-1">
          <div className="p-3 font-mono text-xs">
            <EntityNode id={rootId} depth={0} defaultOpen onNumber={setRootNumber} />
          </div>
        </ScrollArea>
      </div>
    </div>
  );
}

/** One entity instance: name #id with its fields beneath. */
function EntityNode({ id, depth, defaultOpen, initialName, onNumber }: { id: number; depth: number; defaultOpen?: boolean; initialName?: string; onNumber?: (n: string) => void }) {
  const [open, setOpen] = useState(!!defaultOpen);
  const [fields, setFields] = useState<ExplorerField[] | null>(null);
  const [name, setName] = useState(initialName || '');
  const [number, setNumber] = useState('');
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (!open || fields) return;
    debugEntity(id).then((r) => {
      if (r.success) {
        setFields(r.fields || []);
        setName(r.name || '');
        const num = (r as { entityNumber?: string }).entityNumber || '';
        setNumber(num);
        onNumber?.(num);
      } else {
        setError(r.error || 'not found at this position');
      }
    });
  }, [open, fields, id]);

  return (
    <div style={{ paddingLeft: depth > 1 ? 12 : 0, display: defaultOpen ? 'block' : 'inline-block', verticalAlign: 'top' }}>
      {!defaultOpen && (
        <button className="text-blue-400 hover:underline" onClick={() => setOpen(!open)}>
          {open ? '▾' : '▸'} {name || 'entity'}
          {number && <span className="text-muted-foreground" title="EDD entity number"> №{number}</span>}
          <span className="text-muted-foreground">#{id}</span>
        </button>
      )}
      {error && <div className="text-amber-400 pl-4">{error}</div>}
      {open &&
        fields &&
        fields
          .filter((f) => !f.name.includes('*'))
          .map((f) => <FieldRow key={f.name} field={f} depth={depth + 1} />)}
    </div>
  );
}

/** One field row: scalar inline; entity/array expandable. */
function FieldRow({ field, depth }: { field: ExplorerField; depth: number }) {
  if (field.self) {
    // The conventional self-reference: identifies the instance, goes
    // nowhere — no navigation into itself.
    return (
      <div className="leading-5 flex gap-2" style={{ paddingLeft: 12 }}>
        <span className="text-muted-foreground shrink-0">{field.name}</span>
        <span className="text-muted-foreground/70 italic">this {field.entity}</span>
      </div>
    );
  }
  if (field.kind === 'entity' && field.id) {
    return (
      <div className="leading-5" style={{ paddingLeft: 12 }}>
        <span className="text-muted-foreground">{field.name} </span>
        <EntityNode id={field.id} depth={1} initialName={field.entity} />
      </div>
    );
  }
  if (field.kind === 'array' && field.arrayId) {
    return (
      <div className="leading-5" style={{ paddingLeft: 12 }}>
        <span className="text-muted-foreground">{field.name} </span>
        <ArrayNode arrayId={field.arrayId} length={field.length || 0} depth={depth} />
      </div>
    );
  }
  return (
    <div className="leading-5 flex gap-2" style={{ paddingLeft: 12 }}>
      <span className="text-muted-foreground shrink-0">{field.name}</span>
      <span className="truncate" title={`${field.type || ''} ${field.value || ''}`.trim()}>
        {field.value === '' || field.value === undefined ? <span className="text-muted-foreground/60">—</span> : field.value}
      </span>
    </div>
  );
}

/** An array: expandable element list with paging. */
function ArrayNode({ arrayId, length, depth }: { arrayId: number; length: number; depth: number }) {
  const [open, setOpen] = useState(false);
  const [elements, setElements] = useState<ExplorerValue[]>([]);
  const [total, setTotal] = useState(length);
  const [loading, setLoading] = useState(false);

  const loadMore = async () => {
    setLoading(true);
    const r = await debugArray(arrayId, elements.length, 100);
    setLoading(false);
    if (r.success) {
      setElements((e) => [...e, ...(r.elements || [])]);
      setTotal(r.total ?? length);
    }
  };

  useEffect(() => {
    if (open && elements.length === 0 && total > 0) loadMore();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [open]);

  return (
    <span>
      <button className="text-blue-400 hover:underline" onClick={() => setOpen(!open)}>
        {open ? '▾' : '▸'} [{total.toLocaleString()} element{total === 1 ? '' : 's'}]
      </button>
      {open && (
        <div>
          {elements.map((el, i) => (
            <div key={i} className="leading-5" style={{ paddingLeft: 12 }}>
              <span className="text-muted-foreground">[{i}] </span>
              {el.kind === 'entity' && el.id ? (
                <EntityNode id={el.id} depth={1} initialName={el.entity} />
              ) : el.kind === 'array' && el.arrayId ? (
                <ArrayNode arrayId={el.arrayId} length={el.length || 0} depth={depth + 1} />
              ) : (
                <span title={el.type}>{el.value || '—'}</span>
              )}
            </div>
          ))}
          {elements.length < total && (
            <button
              className="text-blue-400 hover:underline"
              style={{ paddingLeft: 12 }}
              onClick={loadMore}
              disabled={loading}
            >
              {loading ? 'loading…' : `load more (${elements.length} of ${total.toLocaleString()})`}
            </button>
          )}
        </div>
      )}
    </span>
  );
}
