/**
 * ReportPanel - the EDD-driven report generator over the loaded trace.
 *
 * The user composes a report from the EDD: pick an entity (every instance
 * the run created) or an array source (elements of entity.attr), choose
 * fields, filter with predicates, sort. Specs save into the project
 * (reports/*.report.json) and run identically against a baseline or a
 * speculative run — the server returns a row-level diff when a
 * speculation is active.
 *
 * @module components/ReportPanel
 */

import { useEffect, useState } from 'react';
import { useProjectStore } from '@/stores/projectStore';
import {
  debugReport,
  listReportSpecs,
  saveReportSpec,
  type ReportDiffResult,
  type ReportResult,
  type ReportSection,
  type ReportSpec,
} from '@/api/client';
import { Button } from '@/components/ui/button';
import { cn } from '@/lib/utils';
import { Play, Plus, Save, Trash2 } from 'lucide-react';

const OPS = ['==', '!=', '>', '>=', '<', '<=', 'contains'];

function emptySection(): ReportSection {
  return { entity: '', source: '', fields: [], where: [], sort: '', key: '' };
}

export function ReportPanel() {
  const { entities, readOnly } = useProjectStore();
  const [name, setName] = useState('report');
  const [sections, setSections] = useState<ReportSection[]>([emptySection()]);
  const [saved, setSaved] = useState<{ name: string; spec: ReportSpec }[]>([]);
  const [result, setResult] = useState<ReportResult | null>(null);
  const [diff, setDiff] = useState<ReportDiffResult | null>(null);
  const [running, setRunning] = useState(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    listReportSpecs().then((r) => r.success && setSaved(r.specs || []));
  }, []);

  const updateSection = (i: number, patch: Partial<ReportSection>) => {
    setSections((s) => s.map((sec, j) => (j === i ? { ...sec, ...patch } : sec)));
  };

  const fieldsOf = (sec: ReportSection): string[] => {
    // Field picker source: the section's entity, or for an array source
    // the entity the user picked as element type (kept in `entity` too).
    const ent = entities.find((e) => e.name.toLowerCase() === (sec.entity || '').toLowerCase());
    return ent ? ent.fields.map((f) => f.name).filter(Boolean) : [];
  };

  const run = async () => {
    setRunning(true);
    setError(null);
    const spec: ReportSpec = {
      name,
      sections: sections.map((s) => ({
        ...s,
        // Array sources use entity only for the field picker.
        entity: s.source ? '' : s.entity,
        title: s.title || (s.source ? s.source : s.entity),
      })),
    };
    const r = await debugReport(spec);
    setRunning(false);
    if (!r.success) {
      setError(r.error || 'Report failed');
      return;
    }
    setResult(r.report || null);
    setDiff(r.diff || null);
  };

  const save = async () => {
    const r = await saveReportSpec(name, { name, sections });
    if (r.success) listReportSpecs().then((x) => x.success && setSaved(x.specs || []));
  };

  const load = (specName: string) => {
    const s = saved.find((x) => x.name === specName);
    if (!s) return;
    setName(s.spec.name || s.name);
    setSections(s.spec.sections?.length ? s.spec.sections : [emptySection()]);
  };

  return (
    <div className="p-4 space-y-4 max-w-5xl">
      {/* Spec header: name, saved specs, actions */}
      <div className="flex items-center gap-2 flex-wrap">
        <span className="text-lg font-bold">Report</span>
        <input
          value={name}
          onChange={(e) => setName(e.target.value)}
          className="h-7 px-2 rounded border border-input bg-transparent font-mono text-sm w-48"
          title="Report name (saved as reports/<name>.report.json)"
        />
        {saved.length > 0 && (
          <select
            className="h-7 px-1 rounded border border-input bg-background text-xs"
            value=""
            onChange={(e) => e.target.value && load(e.target.value)}
            title="Load a saved report spec"
          >
            <option value="">Load saved…</option>
            {saved.map((s) => (
              <option key={s.name} value={s.name}>
                {s.name}
              </option>
            ))}
          </select>
        )}
        <Button variant="outline" size="sm" className="h-7" onClick={run} disabled={running}>
          <Play className="h-3 w-3 mr-1" /> {running ? 'Running…' : 'Run'}
        </Button>
        {!readOnly && (
          <Button variant="outline" size="sm" className="h-7" onClick={save} title="Save the spec into the project">
            <Save className="h-3 w-3 mr-1" /> Save
          </Button>
        )}
      </div>

      {/* Sections */}
      {sections.map((sec, i) => {
        const fieldOptions = fieldsOf(sec);
        const chosen = sec.fields || [];
        return (
          <div key={i} className="border border-border/40 rounded-md p-3 space-y-2">
            <div className="flex items-center gap-2 flex-wrap text-xs">
              <span className="font-semibold text-muted-foreground">Rows: every</span>
              <select
                className="h-7 px-1 rounded border border-input bg-background"
                value={sec.entity || ''}
                onChange={(e) => updateSection(i, { entity: e.target.value, fields: [] })}
                title="Entity — every instance the run created (or the element type of the array source)"
              >
                <option value="">pick an entity…</option>
                {entities.map((e) => (
                  <option key={e.name} value={e.name}>
                    {e.name}
                  </option>
                ))}
              </select>
              <span className="text-muted-foreground">in the run — or the elements of:</span>
              <input
                value={sec.source || ''}
                onChange={(e) => updateSection(i, { source: e.target.value })}
                placeholder="an array, e.g. staking_transaction.to"
                className="h-7 px-2 rounded border border-input bg-transparent font-mono w-64"
                title="Report the ELEMENTS of this array attribute instead of all instances (still pick the entity so its fields show below)"
              />
              {sections.length > 1 && (
                <Button
                  variant="ghost"
                  size="icon"
                  className="h-6 w-6 ml-auto"
                  onClick={() => setSections((s) => s.filter((_, j) => j !== i))}
                  title="Remove section"
                >
                  <Trash2 className="h-3 w-3" />
                </Button>
              )}
            </div>

            {/* Field picker */}
            {!sec.entity && (
              <div className="text-xs text-muted-foreground">
                Pick an entity above and its EDD fields appear here — click the ones you want as columns.
              </div>
            )}
            {fieldOptions.length > 0 && (
              <div className="space-y-1">
              <div className="text-[10px] text-muted-foreground">
                Fields — click to include ({chosen.length === 0 ? 'none selected = every field' : `${chosen.length} selected, in click order`}):
              </div>
              <div className="flex flex-wrap gap-1">
                {fieldOptions.map((f) => {
                  const on = chosen.includes(f);
                  return (
                    <button
                      key={f}
                      className={cn(
                        'px-1.5 py-0.5 rounded-full border text-[11px] font-mono',
                        on ? 'border-blue-400/60 text-blue-300 bg-blue-500/10' : 'border-border text-muted-foreground hover:text-foreground'
                      )}
                      onClick={() =>
                        updateSection(i, { fields: on ? chosen.filter((x) => x !== f) : [...chosen, f] })
                      }
                    >
                      {f}
                    </button>
                  );
                })}
              </div>
              </div>
            )}

            {/* Filters */}
            <div className="space-y-1">
              {(sec.where || []).map((w, wi) => (
                <div key={wi} className="flex items-center gap-1 text-xs">
                  <span className="text-muted-foreground">where</span>
                  <input
                    value={w.field}
                    onChange={(e) =>
                      updateSection(i, {
                        where: sec.where!.map((x, j) => (j === wi ? { ...x, field: e.target.value } : x)),
                      })
                    }
                    placeholder="field"
                    className="h-6 px-1.5 w-40 rounded border border-input bg-transparent font-mono"
                  />
                  <select
                    className="h-6 rounded border border-input bg-background"
                    value={w.op}
                    onChange={(e) =>
                      updateSection(i, {
                        where: sec.where!.map((x, j) => (j === wi ? { ...x, op: e.target.value } : x)),
                      })
                    }
                  >
                    {OPS.map((o) => (
                      <option key={o}>{o}</option>
                    ))}
                  </select>
                  <input
                    value={w.value}
                    onChange={(e) =>
                      updateSection(i, {
                        where: sec.where!.map((x, j) => (j === wi ? { ...x, value: e.target.value } : x)),
                      })
                    }
                    placeholder="value"
                    className="h-6 px-1.5 w-32 rounded border border-input bg-transparent font-mono"
                  />
                  <button
                    className="text-muted-foreground hover:text-foreground"
                    onClick={() => updateSection(i, { where: sec.where!.filter((_, j) => j !== wi) })}
                  >
                    ×
                  </button>
                </div>
              ))}
              <div className="flex items-center gap-3 text-xs">
                <button
                  className="text-blue-400 hover:underline"
                  onClick={() =>
                    updateSection(i, { where: [...(sec.where || []), { field: '', op: '==', value: '' }] })
                  }
                >
                  + filter
                </button>
                <span className="text-muted-foreground">sort:</span>
                <select
                  className="h-6 rounded border border-input bg-background"
                  value={sec.sort || ''}
                  onChange={(e) => updateSection(i, { sort: e.target.value })}
                >
                  <option value="">—</option>
                  {chosen.map((f) => (
                    <option key={f}>{f}</option>
                  ))}
                </select>
                <span className="text-muted-foreground" title="Field that identifies a row across runs (for diffs)">
                  key:
                </span>
                <select
                  className="h-6 rounded border border-input bg-background"
                  value={sec.key || ''}
                  onChange={(e) => updateSection(i, { key: e.target.value })}
                >
                  <option value="">first field</option>
                  {chosen.map((f) => (
                    <option key={f}>{f}</option>
                  ))}
                </select>
              </div>
            </div>
          </div>
        );
      })}
      <Button variant="ghost" size="sm" className="h-7 text-xs" onClick={() => setSections((s) => [...s, emptySection()])}>
        <Plus className="h-3 w-3 mr-1" /> Add section
      </Button>

      {error && <div className="p-2 rounded bg-red-500/10 border border-red-500/30 text-red-400 text-sm">{error}</div>}

      {/* Diff (speculative runs) */}
      {diff && diff.sections.some((s) => (s.added?.length || 0) + (s.removed?.length || 0) + (s.changed?.length || 0) > 0) && (
        <div className="border border-amber-500/40 rounded-md p-3 space-y-2">
          <div className="text-sm font-semibold text-amber-400">Changes vs baseline</div>
          {diff.sections.map((s, si) => (
            <div key={si} className="text-xs font-mono space-y-0.5">
              {(s.added?.length || 0) + (s.removed?.length || 0) + (s.changed?.length || 0) > 0 && (
                <div className="text-muted-foreground">{s.title}</div>
              )}
              {(s.added || []).map((r, i) => (
                <div key={`a${i}`} className="text-green-500">
                  + {s.fields.map((f) => `${f}=${r[f]}`).join('  ')}
                </div>
              ))}
              {(s.removed || []).map((r, i) => (
                <div key={`r${i}`} className="text-red-400">
                  − {s.fields.map((f) => `${f}=${r[f]}`).join('  ')}
                </div>
              ))}
              {(s.changed || []).map((ch, i) => (
                <div key={`c${i}`} className="text-amber-300">
                  ~ {ch.key}: {ch.fields.map((f) => `${f} ${ch.before[f]} → ${ch.after[f]}`).join('  ')}
                </div>
              ))}
            </div>
          ))}
        </div>
      )}

      {/* Results */}
      {result &&
        result.sections.map((s, si) => (
          <div key={si} className="space-y-1">
            <div className="text-sm font-semibold">
              {s.title}{' '}
              <span className="text-xs text-muted-foreground font-normal">
                {(s.rows || []).length}
                {(s.rows || []).length !== s.total ? ` of ${s.total}` : ''} rows
              </span>
            </div>
            {s.error ? (
              <div className="text-xs text-amber-400">{s.error}</div>
            ) : (
              <div className="overflow-x-auto border border-border/40 rounded-md">
                <table className="border-collapse w-full text-xs font-mono">
                  <thead>
                    <tr>
                      {s.fields.map((f) => (
                        <th key={f} className="border border-border/40 bg-muted/30 px-2 py-0.5 text-left text-muted-foreground">
                          {f}
                        </th>
                      ))}
                    </tr>
                  </thead>
                  <tbody>
                    {(s.rows || []).map((row, ri) => (
                      <tr key={ri} className="hover:bg-accent/30">
                        {s.fields.map((f) => (
                          <td key={f} className="border border-border/40 px-2 py-0.5 whitespace-nowrap">
                            {row[f]}
                          </td>
                        ))}
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            )}
          </div>
        ))}
    </div>
  );
}
